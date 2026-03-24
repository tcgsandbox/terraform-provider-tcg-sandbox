package provider

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &gameResource{}
	_ resource.ResourceWithConfigure   = &gameResource{}
	_ resource.ResourceWithImportState = &gameResource{}
)

func NewGameResource() resource.Resource {
	return &gameResource{}
}

type gameResource struct {
	client *Client
}

type gameResourceModel struct {
	ID                      types.String            `tfsdk:"id"`
	Name                    types.String            `tfsdk:"name"`
	Description             types.String            `tfsdk:"description"`
	BannerImagePath         types.String            `tfsdk:"banner_image_path"`
	BannerImagePublicUrl    types.String            `tfsdk:"banner_image_public_url"`
	BannerVerticalAlignment types.Int64             `tfsdk:"banner_vertical_alignment"`
	Attributes              map[string]types.String `tfsdk:"attributes"`
	Owner                   types.String            `tfsdk:"owner"`
	Playable                types.Bool              `tfsdk:"playable"`
	Options                 *gameOptionsModel       `tfsdk:"options"`
	Rules                   *gameRulesModel         `tfsdk:"rules"`
	GamePlayData            *gamePlayDataModel      `tfsdk:"game_play_data"`
}

const bannerImageHashKey = "banner_image_hash"

func (r *gameResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *gameResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_game"
}

func (r *gameResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a game in TCG Sandbox.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the game.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the game.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the game.",
				Required:            true,
			},
			"banner_image_path": schema.StringAttribute{
				MarkdownDescription: "Path to a local image file to use as the banner image. The file will be read and sent as a base64 data URL.",
				Required:            true,
			},
			"banner_image_public_url": schema.StringAttribute{
				MarkdownDescription: "The public URL of the banner image after upload.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"banner_vertical_alignment": schema.Int64Attribute{
				MarkdownDescription: "Banner vertical alignment value.",
				Required:            true,
			},
			"attributes": schema.MapAttribute{
				MarkdownDescription: "A map of attribute names to their types ('string', 'number', or 'boolean'). Changing this requires replacing the resource.",
				Required:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"owner": schema.StringAttribute{
				MarkdownDescription: "The user ID of the game owner.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"playable": schema.BoolAttribute{
				MarkdownDescription: "Whether the game is ready to be played.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"options": schema.SingleNestedBlock{
				MarkdownDescription: "Configuration options for how the game displays cards and other elements.",
				Attributes: map[string]schema.Attribute{
					"card_display_mode": schema.StringAttribute{
						MarkdownDescription: "Controls how cards are displayed ('managed' or 'imageonly').",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"card_display_context": schema.StringAttribute{
						MarkdownDescription: "Controls where the display mode applies ('everywhere' or 'ingameonly').",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"rules": schema.SingleNestedBlock{
				MarkdownDescription: "Game rules in markdown format.",
				Attributes: map[string]schema.Attribute{
					"content": schema.StringAttribute{
						MarkdownDescription: "Markdown content for the game rules.",
						Optional:            true,
					},
				},
			},
			"game_play_data": schema.SingleNestedBlock{
				MarkdownDescription: "The grid configuration and player count settings for the game board layout.",
				Attributes: map[string]schema.Attribute{
					"player_count": schema.Int64Attribute{
						MarkdownDescription: "The number of players for the game (1-4).",
						Optional:            true,
					},
					"slots": schema.ListNestedAttribute{
						MarkdownDescription: "Array of grid slots defining the game board layout.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"row": schema.Int64Attribute{
									MarkdownDescription: "The row position of the slot in the grid (0-based index).",
									Required:            true,
								},
								"column": schema.Int64Attribute{
									MarkdownDescription: "The column position of the slot in the grid (0-based index).",
									Required:            true,
								},
								"width": schema.Int64Attribute{
									MarkdownDescription: "The width of the slot in grid units (how many columns it spans).",
									Required:            true,
								},
								"height": schema.Int64Attribute{
									MarkdownDescription: "The height of the slot in grid units (how many rows it spans).",
									Required:            true,
								},
								"type": schema.StringAttribute{
									MarkdownDescription: "The type of content the slot holds ('cards' or 'counters').",
									Required:            true,
								},
								"max_count": schema.Int64Attribute{
									MarkdownDescription: "The maximum number of items this slot can hold.",
									Required:            true,
								},
								"visibility": schema.StringAttribute{
									MarkdownDescription: "Whether the slot is visible to all players ('public') or only its owner ('private').",
									Required:            true,
								},
								"player_owner": schema.Int64Attribute{
									MarkdownDescription: "The player number (1-based) who owns this slot, or null if unowned.",
									Optional:            true,
								},
							},
						},
					},
				},
			},
		},
	}
}

// readImageAsDataURL reads a file from disk and returns a data URL string.
func readImageAsDataURL(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading image file %s: %w", path, err)
	}

	ext := filepath.Ext(path)
	var mimeType string
	switch ext {
	case ".png":
		mimeType = "image/png"
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	case ".svg":
		mimeType = "image/svg+xml"
	default:
		mimeType = http.DetectContentType(data)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), nil
}

// hashImageFile returns the hex-encoded SHA-256 hash of the file at path.
func hashImageFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading image file %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// mapGameToResourceState maps an API Game response to the Terraform resource model.
func mapGameToResourceState(game *Game, state *gameResourceModel) {
	state.ID = types.StringValue(game.Id)
	state.Name = types.StringValue(game.Name)
	state.Description = types.StringValue(game.Description)
	state.BannerImagePublicUrl = types.StringValue(game.BannerImagePublicUrl)
	state.BannerVerticalAlignment = types.Int64Value(int64(game.BannerVerticalAlignment))
	state.Playable = types.BoolValue(game.Playable)
	state.Owner = optionalString(game.Owner)
	state.Options = mapOptionsFromAPI(game.Options)
	state.GamePlayData = mapGamePlayDataFromAPI(game.GamePlayData)
}

// ImportState imports a game resource by ID.
func (r *gameResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Create creates the resource and sets the initial Terraform state.
func (r *gameResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gameResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bannerDataURL, err := readImageAsDataURL(plan.BannerImagePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read banner image", err.Error())
		return
	}

	bannerHash, err := hashImageFile(plan.BannerImagePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to hash banner image", err.Error())
		return
	}

	attributes := make(map[string]GameAttributeType)
	for k, v := range plan.Attributes {
		attributes[k] = GameAttributeType(v.ValueString())
	}

	body := CreateGameJSONRequestBody{
		Name:                    plan.Name.ValueString(),
		Description:             plan.Description.ValueString(),
		BannerImage:             bannerDataURL,
		BannerVerticalAlignment: int(plan.BannerVerticalAlignment.ValueInt64()),
		Attributes:              attributes,
	}

	httpResp, err := r.client.CreateGame(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create game", err.Error())
		return
	}

	gameResp, err := ParseCreateGameResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse create game response", err.Error())
		return
	}

	if gameResp.JSON201 == nil {
		resp.Diagnostics.AddError("Failed to create game", fmt.Sprintf("Unexpected status: %s, body: %s", gameResp.Status(), string(gameResp.Body)))
		return
	}

	gameID := gameResp.JSON201.Id

	// Send options, rules, and grid in a single update call if any are specified
	updateBody := UpdateGameJSONRequestBody{}
	needsUpdate := false
	if plan.Options != nil {
		updateBody.Options = newGameOptions(plan.Options)
		needsUpdate = true
	}
	if plan.Rules != nil {
		rules := plan.Rules.Content.ValueString()
		updateBody.Rules = &rules
		needsUpdate = true
	}
	if plan.GamePlayData != nil {
		updateBody.GamePlayData = newGamePlayData(plan.GamePlayData)
		needsUpdate = true
	}
	if needsUpdate {
		updateResp, err := r.client.UpdateGame(ctx, gameID, updateBody)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update game after create", err.Error())
			return
		}
		parsedUpdate, err := ParseUpdateGameResponse(updateResp)
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse update game response", err.Error())
			return
		}
		if parsedUpdate.JSON200 == nil {
			resp.Diagnostics.AddError("Failed to update game after create", fmt.Sprintf("Unexpected status: %s, body: %s", parsedUpdate.Status(), string(parsedUpdate.Body)))
			return
		}
	}

	// Re-read the game to get the final state
	readResp, err := r.client.GetGameById(ctx, gameID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read game after create", err.Error())
		return
	}

	readGameResp, err := ParseGetGameByIdResponse(readResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse game response after create", err.Error())
		return
	}

	if readGameResp.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read game after create", fmt.Sprintf("Unexpected status: %s", readGameResp.Status()))
		return
	}

	state := plan
	mapGameToResourceState(readGameResp.JSON200, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	hashJSON, _ := json.Marshal(bannerHash)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, bannerImageHashKey, hashJSON)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *gameResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gameResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.GetGameById(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read game", err.Error())
		return
	}

	gameResp, err := ParseGetGameByIdResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse game response", err.Error())
		return
	}

	if gameResp.HTTPResponse.StatusCode == http.StatusNotFound || gameResp.HTTPResponse.StatusCode == http.StatusBadRequest {
		resp.State.RemoveResource(ctx)
		return
	}

	if gameResp.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read game", fmt.Sprintf("Unexpected status: %s", gameResp.Status()))
		return
	}

	// Preserve banner_image_path from prior state (local file path, not in any API).
	bannerImagePath := state.BannerImagePath
	mapGameToResourceState(gameResp.JSON200, &state)
	if !bannerImagePath.IsNull() && !bannerImagePath.IsUnknown() {
		state.BannerImagePath = bannerImagePath
	}

	// Fetch attributes from the implicitly-created "base" game set.
	gameID := state.ID.ValueString()
	setResp, err := r.client.GetGameSetById(ctx, gameID, "base")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read base game set", err.Error())
		return
	}
	parsedSet, err := ParseGetGameSetByIdResponse(setResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse base game set response", err.Error())
		return
	}
	if parsedSet.JSON200 != nil {
		attrs := make(map[string]types.String, len(parsedSet.JSON200.Attributes))
		for k, v := range parsedSet.JSON200.Attributes {
			attrs[k] = types.StringValue(string(v))
		}
		state.Attributes = attrs
	}

	// Fetch rules from the well-known GCS public URL.
	rulesURL := fmt.Sprintf("https://storage.googleapis.com/tcg-sandbox/games/%s/rules.md", gameID)
	rulesResp, err := http.Get(rulesURL)
	if err == nil {
		defer rulesResp.Body.Close()
		if rulesResp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(rulesResp.Body)
			if err == nil {
				state.Rules = &gameRulesModel{Content: types.StringValue(string(body))}
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *gameResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan gameResourceModel
	var state gameResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	gameID := state.ID.ValueString()
	name := plan.Name.ValueString()
	description := plan.Description.ValueString()
	bva := int(plan.BannerVerticalAlignment.ValueInt64())

	body := UpdateGameJSONRequestBody{
		Name:                    &name,
		Description:             &description,
		BannerVerticalAlignment: &bva,
	}

	if plan.Options != nil {
		body.Options = newGameOptions(plan.Options)
	}

	if plan.Rules != nil {
		rules := plan.Rules.Content.ValueString()
		body.Rules = &rules
	}

	if plan.GamePlayData != nil {
		body.GamePlayData = newGamePlayData(plan.GamePlayData)
	}

	// Re-read and send the banner image if the path or file contents changed
	newHash, err := hashImageFile(plan.BannerImagePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to hash banner image", err.Error())
		return
	}
	priorHashBytes, diags := req.Private.GetKey(ctx, bannerImageHashKey)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var priorHash string
	if priorHashBytes != nil {
		_ = json.Unmarshal(priorHashBytes, &priorHash)
	}

	if !plan.BannerImagePath.Equal(state.BannerImagePath) || newHash != priorHash {
		bannerDataURL, err := readImageAsDataURL(plan.BannerImagePath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to read banner image", err.Error())
			return
		}
		body.BannerImage = &bannerDataURL
	}

	httpResp, err := r.client.UpdateGame(ctx, gameID, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update game", err.Error())
		return
	}

	gameResp, err := ParseUpdateGameResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse update game response", err.Error())
		return
	}

	if gameResp.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to update game", fmt.Sprintf("Unexpected status: %s, body: %s", gameResp.Status(), string(gameResp.Body)))
		return
	}

	// Re-read to get final state
	readResp, err := r.client.GetGameById(ctx, gameID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read game after update", err.Error())
		return
	}

	readGameResp, err := ParseGetGameByIdResponse(readResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse game response after update", err.Error())
		return
	}

	if readGameResp.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read game after update", fmt.Sprintf("Unexpected status: %s", readGameResp.Status()))
		return
	}

	newState := plan
	mapGameToResourceState(readGameResp.JSON200, &newState)

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
	newHashJSON, _ := json.Marshal(newHash)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, bannerImageHashKey, newHashJSON)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *gameResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gameResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.DeleteGame(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete game", err.Error())
		return
	}

	deleteResp, err := ParseDeleteGameResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse delete game response", err.Error())
		return
	}

	if deleteResp.HTTPResponse.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("Failed to delete game", fmt.Sprintf("Unexpected status: %s, body: %s", deleteResp.Status(), string(deleteResp.Body)))
	}
}
