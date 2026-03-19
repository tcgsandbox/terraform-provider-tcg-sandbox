#!/bin/bash
# Analyze API changes and identify missing Terraform implementations.
# This script is intended for agent consumption only.
#
# It syncs the API spec, regenerates the client, then produces a structured
# report of what changed, what's implemented, and what's missing.
#
# Compatible with bash 3.x (macOS default).

set -e

PROJECT_DIR="$(git rev-parse --show-toplevel)"
cd "$PROJECT_DIR"

CLIENT_FILE="internal/provider/client_generated.go"
PROVIDER_DIR="internal/provider"
REPORT_FILE="/tmp/terraform-provider-api-analysis.md"

# Entities to ignore — internal-only API endpoints not intended for Terraform use
IGNORED_ENTITIES="api_key lobby page user user_deck game_set_card game_set"

is_ignored() {
    for ignored in $IGNORED_ENTITIES; do
        [ "$1" = "$ignored" ] && return 0
    done
    return 1
}

# Temp files for entity tracking (avoid bash 4 associative arrays)
ENTITY_OPS_FILE=$(mktemp)
IMPLEMENTED_FILE=$(mktemp)
trap 'rm -f "$ENTITY_OPS_FILE" "$IMPLEMENTED_FILE"' EXIT

# Tee all output to both stdout and the report file
exec > >(tee "$REPORT_FILE") 2>&1

# --- Step 1: Sync and regenerate ---
echo "=== SYNCING API SPEC AND REGENERATING CLIENT ==="
bash scripts/sync-api-docs.sh
bash scripts/generate-client.sh
echo ""

# --- Step 2: Client diff ---
echo "=== CHANGED ==="
if git diff --quiet "$CLIENT_FILE" 2>/dev/null; then
    echo "No changes to $CLIENT_FILE"
    CLIENT_CHANGED=false
else
    CLIENT_CHANGED=true
    git diff --stat "$CLIENT_FILE"
    echo ""
    echo "Method signature changes:"
    git diff -U0 "$CLIENT_FILE" | grep -E '^\+.*func \(c \*Client\)' | sed 's/^\+/  + /' || true
    git diff -U0 "$CLIENT_FILE" | grep -E '^\-.*func \(c \*Client\)' | sed 's/^\-/  - /' || true
    echo ""
    echo "Type definition changes:"
    git diff -U0 "$CLIENT_FILE" | grep -E '^\+.*type \w+ struct' | sed 's/^\+/  + /' || true
    git diff -U0 "$CLIENT_FILE" | grep -E '^\-.*type \w+ struct' | sed 's/^\-/  - /' || true
fi
echo ""

# --- Helper: look up entity ops from temp file ---
get_entity_ops() {
    grep "^$1	" "$ENTITY_OPS_FILE" | cut -f2 || true
}

add_entity_op() {
    local entity="$1" verb="$2"
    local existing
    existing=$(get_entity_ops "$entity")
    if [ -n "$existing" ]; then
        # Remove old line and add updated one
        grep -v "^${entity}	" "$ENTITY_OPS_FILE" > "${ENTITY_OPS_FILE}.tmp" || true
        mv "${ENTITY_OPS_FILE}.tmp" "$ENTITY_OPS_FILE"
        echo "${entity}	${existing}, ${verb}" >> "$ENTITY_OPS_FILE"
    else
        echo "${entity}	${verb}" >> "$ENTITY_OPS_FILE"
    fi
}

# --- Helper: snake_case to PascalCase ---
to_pascal() {
    echo "$1" | awk '{for(i=1;i<=NF;i++){$i=toupper(substr($i,1,1))substr($i,2)}}1' FS='_' OFS=''
}

# --- Step 3: Extract all client methods and group by entity ---
methods=$(grep -oE 'func \(c \*Client\) [A-Za-z_]+' "$CLIENT_FILE" \
    | sed 's/func (c \*Client) //' \
    | grep -vE '^(Get|GetDocs|GetOpenapiYaml|applyEditors)$' \
    | grep -v 'WithBody$')

while IFS= read -r method; do
    verb=""
    rest=""
    case "$method" in
        Create*) verb="Create"; rest="${method#Create}" ;;
        Delete*) verb="Delete"; rest="${method#Delete}" ;;
        Update*) verb="Update"; rest="${method#Update}" ;;
        List*)   verb="List";   rest="${method#List}" ;;
        Get*)    verb="Get";    rest="${method#Get}" ;;
        Star*)   verb="Star";   rest="${method#Star}" ;;
        Unstar*) verb="Unstar"; rest="${method#Unstar}" ;;
        *) continue ;;
    esac

    # Strip common suffixes
    rest="${rest%ById}"
    # Strip plural suffixes for List operations
    if [ "$verb" = "List" ]; then
        case "$rest" in
            *ies) rest="${rest%ies}y" ;;  # Lobbies -> Lobby
            *s)   rest="${rest%s}" ;;      # Games -> Game
        esac
    fi

    [ -z "$rest" ] && continue

    # Convert PascalCase to snake_case (macOS compatible)
    entity=$(echo "$rest" | sed 's/\([A-Z]\)/_\1/g' | sed 's/^_//' | tr '[:upper:]' '[:lower:]')

    # Skip internal-only entities
    is_ignored "$entity" && continue

    add_entity_op "$entity" "$verb"
done <<< "$methods"

# --- Detect sub-resources by checking method signatures for parent ID params ---
detect_parents() {
    local entity_pascal
    entity_pascal=$(to_pascal "$1")

    local sig
    sig=$(grep -E "func \(c \*Client\) Create${entity_pascal}\(" "$CLIENT_FILE" 2>/dev/null || true)
    if [ -z "$sig" ]; then
        sig=$(grep -E "func \(c \*Client\) Create${entity_pascal}WithBody\(" "$CLIENT_FILE" 2>/dev/null || true)
    fi

    local parents=""
    while IFS= read -r param; do
        [ -n "$param" ] && parents="${parents:+$parents, }$param"
    done < <(echo "$sig" | grep -oE '[a-z]+Id' | grep -v 'reqEditors' || true)

    echo "$parents"
}

# --- Step 4: List existing implementations ---
for f in "$PROVIDER_DIR"/*_resource.go; do
    [ -f "$f" ] || continue
    name=$(basename "$f" _resource.go)
    echo "${name}	resource" >> "$IMPLEMENTED_FILE"
done
for f in "$PROVIDER_DIR"/*_data_source.go; do
    [ -f "$f" ] || continue
    name=$(basename "$f" _data_source.go)
    if grep -q "^${name}	" "$IMPLEMENTED_FILE"; then
        sed "s/^${name}	.*/${name}	resource + data_source/" "$IMPLEMENTED_FILE" > "${IMPLEMENTED_FILE}.tmp"
        mv "${IMPLEMENTED_FILE}.tmp" "$IMPLEMENTED_FILE"
    else
        echo "${name}	data_source" >> "$IMPLEMENTED_FILE"
    fi
done

# --- Step 5: Detect changes affecting existing implementations ---
echo "=== EXISTING IMPLEMENTATIONS ==="
if [ "$CLIENT_CHANGED" = true ]; then
    sort "$IMPLEMENTED_FILE" | while IFS='	' read -r entity impl_type; do
        entity_pascal=$(to_pascal "$entity")
        ops=$(get_entity_ops "$entity")

        # Check if any changed lines in the diff reference this entity's types or methods
        changed_lines=$(git diff -U0 "$CLIENT_FILE" | grep -E "^\+.*${entity_pascal}" | grep -v '^\+\+\+' || true)

        echo "  $entity: $impl_type (ops: ${ops:-unknown})"
        if [ -n "$changed_lines" ]; then
            echo "    *** HAS API CHANGES - may need updates ***"
            echo "$changed_lines" | head -10 | sed 's/^\+/      /'
            remaining=$(echo "$changed_lines" | tail -n +11 | wc -l | tr -d ' ')
            [ "$remaining" -gt 0 ] && echo "      ... and $remaining more changed lines"
        else
            echo "    No API changes detected"
        fi
        echo ""
    done
else
    sort "$IMPLEMENTED_FILE" | while IFS='	' read -r entity impl_type; do
        ops=$(get_entity_ops "$entity")
        echo "  $entity: $impl_type (ops: ${ops:-unknown}) - No API changes"
    done
fi
echo ""

# --- Step 6: Report missing implementations ---
echo "=== MISSING IMPLEMENTATIONS ==="
missing_found=false
sort "$ENTITY_OPS_FILE" | while IFS='	' read -r entity ops; do
    if ! grep -q "^${entity}	" "$IMPLEMENTED_FILE"; then
        missing_found=true
        parents=$(detect_parents "$entity")
        echo "  $entity"
        echo "    Operations: $ops"
        if [ -n "$parents" ]; then
            echo "    Sub-resource (parent params: $parents)"
        fi
        echo ""
    fi
done

# Check if anything was missing (the while loop runs in a subshell so we check differently)
has_missing=$(sort "$ENTITY_OPS_FILE" | while IFS='	' read -r entity ops; do
    if ! grep -q "^${entity}	" "$IMPLEMENTED_FILE"; then
        echo "yes"
        break
    fi
done)

if [ -z "$has_missing" ]; then
    echo "  All API entities have implementations."
fi

echo ""
echo "Report saved to $REPORT_FILE"
