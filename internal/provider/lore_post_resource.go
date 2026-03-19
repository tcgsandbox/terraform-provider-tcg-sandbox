package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &lorePostResource{}
	_ resource.ResourceWithConfigure = &lorePostResource{}
)

func NewLorePostResource() resource.Resource {
	return &lorePostResource{}
}

type lorePostResource struct {
	client *Client
}

type lorePostResourceModel struct {
	ID      types.String `tfsdk:"id"`
	GameID  types.String `tfsdk:"game_id"`
	Title   types.String `tfsdk:"title"`
	Content types.String `tfsdk:"content"`
}

func (r *lorePostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *lorePostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lore_post"
}

func (r *lorePostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a lore post in a TCG Sandbox game.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the lore post.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"game_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the game this lore post belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The title of the lore post.",
				Required:            true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "The markdown content of the lore post. This is a write-only field; the API does not return it.",
				Required:            true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *lorePostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan lorePostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	gameID := plan.GameID.ValueString()

	body := CreateLorePostJSONRequestBody{
		Title:   plan.Title.ValueString(),
		Content: plan.Content.ValueString(),
	}

	httpResp, err := r.client.CreateLorePost(ctx, gameID, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create lore post", err.Error())
		return
	}

	createResp, err := ParseCreateLorePostResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse create lore post response", err.Error())
		return
	}

	if createResp.JSON201 == nil {
		resp.Diagnostics.AddError("Failed to create lore post", fmt.Sprintf("Unexpected status: %s, body: %s", createResp.Status(), string(createResp.Body)))
		return
	}

	postID := createResp.JSON201.Id

	// Re-read the lore post to get the server's canonical state
	readResp, err := r.client.GetLorePost(ctx, gameID, postID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read lore post after create", err.Error())
		return
	}

	getResp, err := ParseGetLorePostResponse(readResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse lore post response after create", err.Error())
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read lore post after create", fmt.Sprintf("Unexpected status: %s", getResp.Status()))
		return
	}

	state := plan
	mapLorePostToState(getResp.JSON200, &state.ID, &state.GameID, &state.Title)
	// Preserve content from plan since API doesn't return it
	state.Content = plan.Content

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *lorePostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state lorePostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.GetLorePost(ctx, state.GameID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read lore post", err.Error())
		return
	}

	getResp, err := ParseGetLorePostResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse lore post response", err.Error())
		return
	}

	if getResp.HTTPResponse.StatusCode == http.StatusNotFound || getResp.HTTPResponse.StatusCode == http.StatusBadRequest {
		resp.State.RemoveResource(ctx)
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read lore post", fmt.Sprintf("Unexpected status: %s", getResp.Status()))
		return
	}

	// Preserve content from prior state since API doesn't return it
	content := state.Content
	mapLorePostToState(getResp.JSON200, &state.ID, &state.GameID, &state.Title)
	if !content.IsNull() && !content.IsUnknown() {
		state.Content = content
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *lorePostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan lorePostResourceModel
	var state lorePostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	gameID := state.GameID.ValueString()
	postID := state.ID.ValueString()

	title := plan.Title.ValueString()
	content := plan.Content.ValueString()

	body := UpdateLorePostJSONRequestBody{
		Title:   &title,
		Content: &content,
	}

	httpResp, err := r.client.UpdateLorePost(ctx, gameID, postID, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update lore post", err.Error())
		return
	}

	updateResp, err := ParseUpdateLorePostResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse update lore post response", err.Error())
		return
	}

	if updateResp.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to update lore post", fmt.Sprintf("Unexpected status: %s, body: %s", updateResp.Status(), string(updateResp.Body)))
		return
	}

	// Re-read to get final state
	readResp, err := r.client.GetLorePost(ctx, gameID, postID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read lore post after update", err.Error())
		return
	}

	getResp, err := ParseGetLorePostResponse(readResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse lore post response after update", err.Error())
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read lore post after update", fmt.Sprintf("Unexpected status: %s", getResp.Status()))
		return
	}

	newState := plan
	mapLorePostToState(getResp.JSON200, &newState.ID, &newState.GameID, &newState.Title)
	// Preserve content from plan since API doesn't return it
	newState.Content = plan.Content

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *lorePostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state lorePostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.DeleteLorePost(ctx, state.GameID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete lore post", err.Error())
		return
	}

	deleteResp, err := ParseDeleteLorePostResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse delete lore post response", err.Error())
		return
	}

	if deleteResp.HTTPResponse.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("Failed to delete lore post", fmt.Sprintf("Unexpected status: %s, body: %s", deleteResp.Status(), string(deleteResp.Body)))
	}
}
