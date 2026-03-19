package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &gameSetResource{}
	_ resource.ResourceWithConfigure = &gameSetResource{}
)

func NewGameSetResource() resource.Resource {
	return &gameSetResource{}
}

type gameSetResource struct {
	client *Client
}

func (r *gameSetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *gameSetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_game_set"
}

func (r *gameSetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a game set in TCG Sandbox.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The user-chosen set ID/slug. This becomes the immutable identifier for the set.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"game_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the parent game.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The display name of the set.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the set.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"attributes": schema.MapAttribute{
				MarkdownDescription: "A map of attribute names to their types ('string', 'number', or 'boolean').",
				Required:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *gameSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gameSetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	attributes := make(map[string]GameAttributeType)
	for k, v := range plan.Attributes {
		attributes[k] = GameAttributeType(v.ValueString())
	}

	body := CreateGameSetJSONRequestBody{
		SetId:       plan.ID.ValueString(),
		DisplayName: plan.Name.ValueString(),
		Attributes:  attributes,
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		desc := plan.Description.ValueString()
		body.Description = &desc
	}

	gameID := plan.GameID.ValueString()

	httpResp, err := r.client.CreateGameSet(ctx, gameID, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create game set", err.Error())
		return
	}

	createResp, err := ParseCreateGameSetResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse create game set response", err.Error())
		return
	}

	if createResp.JSON201 == nil {
		resp.Diagnostics.AddError("Failed to create game set", fmt.Sprintf("Unexpected status: %s, body: %s", createResp.Status(), string(createResp.Body)))
		return
	}

	setID := createResp.JSON201.Id

	// Re-read the game set to get the server's canonical state
	readResp, err := r.client.GetGameSetById(ctx, gameID, setID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read game set after create", err.Error())
		return
	}

	readSetResp, err := ParseGetGameSetByIdResponse(readResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse game set response after create", err.Error())
		return
	}

	if readSetResp.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read game set after create", fmt.Sprintf("Unexpected status: %s", readSetResp.Status()))
		return
	}

	state := plan
	mapGameSetFromAPI(readSetResp.JSON200, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *gameSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gameSetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.GetGameSetById(ctx, state.GameID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read game set", err.Error())
		return
	}

	setResp, err := ParseGetGameSetByIdResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse game set response", err.Error())
		return
	}

	if setResp.HTTPResponse.StatusCode == http.StatusNotFound || setResp.HTTPResponse.StatusCode == http.StatusBadRequest {
		resp.State.RemoveResource(ctx)
		return
	}

	if setResp.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read game set", fmt.Sprintf("Unexpected status: %s", setResp.Status()))
		return
	}

	mapGameSetFromAPI(setResp.JSON200, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is not implemented because the API does not support updating game sets.
// All attributes use RequiresReplace, so Terraform will destroy and recreate as needed.
func (r *gameSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Game sets cannot be updated. All changes require replacing the resource.")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *gameSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gameSetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.DeleteGameSet(ctx, state.GameID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete game set", err.Error())
		return
	}

	deleteResp, err := ParseDeleteGameSetResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse delete game set response", err.Error())
		return
	}

	if deleteResp.HTTPResponse.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("Failed to delete game set", fmt.Sprintf("Unexpected status: %s, body: %s", deleteResp.Status(), string(deleteResp.Body)))
	}
}
