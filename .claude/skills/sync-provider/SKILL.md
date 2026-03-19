---
name: sync-provider
description: Sync the Terraform provider with the latest API changes. Analyze the generated client for new or modified endpoints, then implement missing resources and data sources in parallel. Use this skill when the user runs /sync-provider, asks to sync or update the provider with API changes, or wants to implement missing or changed Terraform resources/data sources.
---

Sync the Terraform provider with the latest API changes. Analyze the generated client for new or modified endpoints, then implement missing resources and data sources in parallel.

## 1. Analyze

Run the analysis script and capture its output:
```
bash .claude/references/analyze-api-changes.sh
```

The full report is also saved to `/tmp/terraform-provider-api-analysis.md` for reference without consuming context.

Parse the output for two categories of work:
- **"HAS API CHANGES"** entries under `EXISTING IMPLEMENTATIONS` — entities already implemented that need updates
- **Entries under `MISSING IMPLEMENTATIONS`** — new entities needing resource/data source files

If there are no changes and no missing entities, report "Provider is up to date" and stop.

**Filtering:** Skip entities that only have a `List` operation and no `Get`/`Create`/`Delete` — these are not meaningful Terraform resources or data sources and should be excluded from implementation.

## 2. Build work items

For each entity needing work, read `internal/provider/client_generated.go` and extract:
- The entity's Go struct (e.g., `type LorePost struct { ... }`)
- Create/Update request body types (e.g., `CreateLorePostJSONBody`)
- All `func (c *Client)` method signatures for this entity
- Whether the analysis flagged it as a sub-resource (parent params)

Collect these into a work item list. Each item is either `new` or `update`.

## 3. Dispatch subagents in parallel

Launch one Agent tool call per entity, all in the same message. Use `subagent_type: "general-purpose"`.

Each subagent gets one of the two prompts below, with the placeholders filled in.

### Prompt for NEW entities

```
You are implementing a Terraform provider entity. Read these reference files first:
- internal/provider/game_resource.go (resource pattern)
- internal/provider/game_data_source.go (data source pattern)
- internal/provider/game_models.go (shared models pattern)
- internal/provider/game_models_test.go (unit test pattern for pure mapper functions)
- internal/provider/game_resource_test.go (acceptance test pattern for resources)
- internal/provider/game_data_source_test.go (acceptance test pattern for data sources)
- CLAUDE.md (project conventions)

Then read the generated client types for your entity from internal/provider/client_generated.go.

## Entity: {ENTITY_SNAKE} (PascalCase: {ENTITY_PASCAL})

Parent params: {PARENT_PARAMS or "none — top-level resource"}
Available client operations: {OPS_LIST}

Relevant generated types (verify these by reading client_generated.go):
{PASTED_TYPES_AND_SIGNATURES}

## What to create

1. **Resource** (`internal/provider/{ENTITY_SNAKE}_resource.go`) — create ONLY if the entity has Create + Get + Delete methods. Structure:
   - Interface checks, constructor, struct with `client *Client`
   - `{ENTITY_SNAKE}ResourceModel` struct with `tfsdk` tags
   - Configure, Metadata (type name: `req.ProviderTypeName + "_{ENTITY_SNAKE}"`), Schema
   - Create: build request body → call Create → parse response → check JSON field → re-read for final state
   - Read: call Get → parse → map to state. If 404/400, call `resp.State.RemoveResource(ctx)`
   - Update: only if Update method exists. Otherwise omit entirely and use `RequiresReplace` on mutable fields
   - Delete: call Delete → parse → check status
   - For sub-resources: add parent ID(s) as Required string schema attributes and pass them to client methods

2. **Data source** (`internal/provider/{ENTITY_SNAKE}_data_source.go`) — create ONLY if Get or List exists. Structure:
   - Interface checks, constructor, struct with `client *Client`
   - `{ENTITY_SNAKE}DataSourceModel` struct — ID is Required, everything else is Computed
   - Read: call Get → parse → map fields to state

3. **Models** (`internal/provider/{ENTITY_SNAKE}_models.go`) — create ONLY if both resource and data source exist, or if there are nested object types that need mapper functions. Contains:
   - Shared Terraform model structs
   - `map{Entity}FromAPI()` functions (API → Terraform)
   - `new{Entity}()` functions (Terraform → API)
   - Use `optionalString()` from game_models.go for nullable string pointers

4. **Unit tests** (`internal/provider/{ENTITY_SNAKE}_models_test.go`) — create for EVERY entity that has a models file with pure mapper functions. Follow the pattern in `game_models_test.go`:
   - Test each mapper function with: all fields populated, nil/optional fields, edge cases (empty maps, type coercion)
   - Use `t.Parallel()` on every test
   - Use simple `Test{FuncName}_{Scenario}` naming
   - No mocking — these are pure data transformation tests
   - Reuse test helpers (`strPtr`, `intPtr`) from `game_models_test.go` (same package)

5. **Acceptance tests** — create for EVERY new resource and data source:
   - **Resource test** (`internal/provider/{ENTITY_SNAKE}_resource_test.go`):
     - Follow `game_resource_test.go` pattern
     - Create parent resources first (game, then set if sub-resource)
     - Test Create+Verify step with `TestCheckResourceAttr` / `TestCheckResourceAttrSet`
     - If Update is supported, add an Update step
     - Use `providerConfig +` prefix for all configs
   - **Data source test** (`internal/provider/{ENTITY_SNAKE}_data_source_test.go`):
     - Follow `game_data_source_test.go` pattern
     - Create the resource, then read it via data source
     - Use `TestCheckResourceAttrPair` to verify resource↔data source field parity

## Critical patterns to follow
- Always re-read after Create/Update to get the server's canonical state
- Use `stringplanmodifier.UseStateForUnknown()` for Computed immutable fields (id, owner)
- Parse responses with `Parse{Method}Response()` and check `JSON{StatusCode}` (e.g., JSON200, JSON201)
- Treat nil JSON response as error: `fmt.Sprintf("Unexpected status: %s, body: %s", resp.Status(), string(resp.Body))`
- For optional `*string` fields from the API: treat BOTH nil AND empty string as null. Do NOT use `optionalString()` — instead use explicit checks: `if ptr != nil && *ptr != "" { types.StringValue(*ptr) } else { types.StringNull() }`. The API returns `""` for unset optional fields, which causes "was null, but now cty.StringVal("")" errors if mapped as-is.
- For `map[string]interface{}` attribute values sent to the API: attempt numeric conversion with `strconv.ParseFloat` and boolean conversion before sending, since the API validates attribute types against the game schema.

## Constraints
- Do NOT modify provider.go — the orchestrator registers new entities
- Do NOT modify any other entity's files
- Keep code simple — no abstractions beyond what game_resource.go uses
```

### Prompt for CHANGED entities

```
You are updating an existing Terraform provider entity to match API changes.

Read these files:
- internal/provider/{ENTITY_SNAKE}_resource.go
- internal/provider/{ENTITY_SNAKE}_data_source.go (if exists)
- internal/provider/{ENTITY_SNAKE}_models.go (if exists)
- internal/provider/{ENTITY_SNAKE}_models_test.go (if exists)
- internal/provider/{ENTITY_SNAKE}_resource_test.go (if exists)
- internal/provider/{ENTITY_SNAKE}_data_source_test.go (if exists)
- internal/provider/client_generated.go (search for {ENTITY_PASCAL} types and methods)
- CLAUDE.md (project conventions)

## Entity: {ENTITY_SNAKE}

The generated client has changed for this entity. Here are the relevant diff lines:
{DIFF_LINES}

## What to update

Compare the current implementation against the latest generated types in client_generated.go:
- Add schema attributes for new API fields
- Remove or update attributes for changed/removed fields
- Update model mapper functions to handle new fields
- If new CRUD endpoints were added (e.g., Update didn't exist before but now does), implement the handler
- If endpoints were removed, remove the corresponding handler
- Update unit tests in `{ENTITY_SNAKE}_models_test.go` to cover new/changed mapper fields
- Update acceptance tests to verify new/changed attributes

Follow the same patterns documented in CLAUDE.md. Do NOT modify provider.go or other entity files.
```

## 4. Register new entities

After all subagents finish, edit `internal/provider/provider.go` once:
- Add `New{Entity}Resource` to the `Resources()` return slice for each new resource
- Add `New{Entity}DataSource` to the `DataSources()` return slice for each new data source

Skip this step if only existing entities were updated (no new files created).

## 5. Verify

Run `make testacc` (which runs both unit and acceptance tests against the live API). If compilation fails, read the errors and fix them. Common issues:
- Missing imports (add the appropriate `terraform-plugin-framework` packages)
- Type mismatches between generated client types and Terraform model fields
- Unregistered resources in provider.go
- "Provider produced inconsistent result after apply" errors — usually caused by optional `*string` fields where the API returns `""` but Terraform expects null (see critical patterns above)
- Attribute type validation errors — card-like resources need numeric/boolean coercion when sending `map[string]interface{}` attribute values to the API

ALL tests (unit AND acceptance) MUST pass before reporting completion. Do NOT consider the task complete until `make testacc` exits with status 0.
