package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &cardResource{}
	_ resource.ResourceWithConfigure = &cardResource{}
)

func NewCardResource() resource.Resource {
	return &cardResource{}
}

type cardResource struct {
	client *Client
}

type cardResourceModel struct {
	ID                 types.String            `tfsdk:"id"`
	GameID             types.String            `tfsdk:"game_id"`
	SetID              types.String            `tfsdk:"set_id"`
	Name               types.String            `tfsdk:"name"`
	Description        types.String            `tfsdk:"description"`
	CardImagePath      types.String            `tfsdk:"card_image_path"`
	CardImagePublicUrl types.String            `tfsdk:"card_image_public_url"`
	Attributes         map[string]types.String `tfsdk:"attributes"`
}

const cardImageHashKey = "card_image_hash"

func (r *cardResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *cardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_card"
}

func (r *cardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a card in a game set in TCG Sandbox.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the card.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"game_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the parent game.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"set_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the set this card belongs to. Defaults to \"base\".",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("base"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the card.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the card.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"card_image_path": schema.StringAttribute{
				MarkdownDescription: "Path to a local image file to use as the card image. The file will be read and sent as a base64 data URL.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"card_image_public_url": schema.StringAttribute{
				MarkdownDescription: "The public URL of the card image after upload.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"attributes": schema.MapAttribute{
				MarkdownDescription: "A map of attribute names to their values.",
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
func (r *cardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cardResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the card image as a data URL if provided.
	var cardImageHash string
	var cardImagePtr *string
	if !plan.CardImagePath.IsNull() && !plan.CardImagePath.IsUnknown() {
		cardImageDataURL, err := readImageAsDataURL(plan.CardImagePath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to read card image", err.Error())
			return
		}
		cardImagePtr = &cardImageDataURL

		hash, err := hashImageFile(plan.CardImagePath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to hash card image", err.Error())
			return
		}
		cardImageHash = hash
	}

	// Convert attributes from map[string]types.String to map[string]interface{}.
	// Attempt numeric conversion so the API receives the correct types.
	attributes := make(map[string]interface{}, len(plan.Attributes))
	for k, v := range plan.Attributes {
		s := v.ValueString()
		if n, err := strconv.ParseFloat(s, 64); err == nil {
			attributes[k] = n
		} else if s == "true" || s == "false" {
			attributes[k] = s == "true"
		} else {
			attributes[k] = s
		}
	}

	body := CreateCardJSONRequestBody{
		Name:       plan.Name.ValueString(),
		Attributes: attributes,
		CardImage:  cardImagePtr,
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		desc := plan.Description.ValueString()
		body.Description = &desc
	}

	gameID := plan.GameID.ValueString()
	setID := plan.SetID.ValueString()

	httpResp, err := r.client.CreateCard(ctx, gameID, setID, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create card", err.Error())
		return
	}

	cardResp, err := ParseCreateCardResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse create card response", err.Error())
		return
	}

	if cardResp.JSON201 == nil {
		resp.Diagnostics.AddError("Failed to create card", fmt.Sprintf("Unexpected status: %s, body: %s", cardResp.Status(), string(cardResp.Body)))
		return
	}

	cardID := cardResp.JSON201.Id

	// Re-read the card to get the server's canonical state.
	readResp, err := r.client.GetCard(ctx, gameID, setID, cardID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read card after create", err.Error())
		return
	}

	readCardResp, err := ParseGetCardResponse(readResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse card response after create", err.Error())
		return
	}

	if readCardResp.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read card after create", fmt.Sprintf("Unexpected status: %s", readCardResp.Status()))
		return
	}

	// Map the API response to state.
	state := plan
	state.ID, state.GameID, state.SetID, state.Name, state.Description, state.CardImagePublicUrl, state.Attributes = mapCardToState(readCardResp.JSON200)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if cardImageHash != "" {
		hashJSON, _ := json.Marshal(cardImageHash)
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, cardImageHashKey, hashJSON)...)
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *cardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cardResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve card_image_path from prior state (local-only field, not returned by API).
	cardImagePath := state.CardImagePath

	httpResp, err := r.client.GetCard(ctx, state.GameID.ValueString(), state.SetID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read card", err.Error())
		return
	}

	cardResp, err := ParseGetCardResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse card response", err.Error())
		return
	}

	if cardResp.HTTPResponse.StatusCode == http.StatusNotFound || cardResp.HTTPResponse.StatusCode == http.StatusBadRequest {
		resp.State.RemoveResource(ctx)
		return
	}

	if cardResp.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read card", fmt.Sprintf("Unexpected status: %s", cardResp.Status()))
		return
	}

	state.ID, state.GameID, state.SetID, state.Name, state.Description, state.CardImagePublicUrl, state.Attributes = mapCardToState(cardResp.JSON200)

	// Restore the local-only card_image_path.
	if !cardImagePath.IsNull() && !cardImagePath.IsUnknown() {
		state.CardImagePath = cardImagePath
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is not supported — all mutable fields use RequiresReplace, so Terraform
// will destroy and recreate the resource instead of calling Update.
func (r *cardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"Card resources do not support in-place updates. All changes require resource replacement.",
	)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *cardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cardResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.DeleteCard(ctx, state.GameID.ValueString(), state.SetID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete card", err.Error())
		return
	}

	deleteResp, err := ParseDeleteCardResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse delete card response", err.Error())
		return
	}

	if deleteResp.HTTPResponse.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("Failed to delete card", fmt.Sprintf("Unexpected status: %s, body: %s", deleteResp.Status(), string(deleteResp.Body)))
	}
}
