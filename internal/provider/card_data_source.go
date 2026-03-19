package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &cardDataSource{}
	_ datasource.DataSourceWithConfigure = &cardDataSource{}
)

func NewCardDataSource() datasource.DataSource {
	return &cardDataSource{}
}

type cardDataSource struct {
	client *Client
}

type cardDataSourceModel struct {
	ID                 types.String            `tfsdk:"id"`
	GameID             types.String            `tfsdk:"game_id"`
	SetID              types.String            `tfsdk:"set_id"`
	Name               types.String            `tfsdk:"name"`
	Description        types.String            `tfsdk:"description"`
	CardImagePublicUrl types.String            `tfsdk:"card_image_public_url"`
	Attributes         map[string]types.String `tfsdk:"attributes"`
}

func (d *cardDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *cardDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_card"
}

func (d *cardDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a single card from a game set in TCG Sandbox.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the card.",
				Required:            true,
			},
			"game_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the parent game.",
				Required:            true,
			},
			"set_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the set this card belongs to. Defaults to \"base\".",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the card.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the card.",
				Computed:            true,
			},
			"card_image_public_url": schema.StringAttribute{
				MarkdownDescription: "The public URL of the card image.",
				Computed:            true,
			},
			"attributes": schema.MapAttribute{
				MarkdownDescription: "A map of attribute names to their values.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

// Read fetches a single card by its ID and populates the Terraform state.
func (d *cardDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state cardDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	setID := state.SetID.ValueString()
	if state.SetID.IsNull() || state.SetID.IsUnknown() {
		setID = "base"
	}

	httpResp, err := d.client.GetCard(ctx, state.GameID.ValueString(), setID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Card", err.Error())
		return
	}

	cardResp, err := ParseGetCardResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Parse Card Response", err.Error())
		return
	}

	if cardResp.JSON200 == nil {
		resp.Diagnostics.AddError("Unable to Read Card", "Card not found")
		return
	}

	state.ID, state.GameID, state.SetID, state.Name, state.Description, state.CardImagePublicUrl, state.Attributes = mapCardToState(cardResp.JSON200)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
