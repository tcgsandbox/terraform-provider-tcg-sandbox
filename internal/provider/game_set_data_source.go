package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &gameSetDataSource{}
	_ datasource.DataSourceWithConfigure = &gameSetDataSource{}
)

func NewGameSetDataSource() datasource.DataSource {
	return &gameSetDataSource{}
}

type gameSetDataSource struct {
	client *Client
}

func (d *gameSetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *gameSetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_game_set"
}

func (d *gameSetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a game set from TCG Sandbox.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the game set.",
				Required:            true,
			},
			"game_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the parent game.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The display name of the set.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the set.",
				Computed:            true,
			},
			"attributes": schema.MapAttribute{
				MarkdownDescription: "A map of attribute names to their types ('string', 'number', or 'boolean').",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *gameSetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gameSetModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.client.GetGameSetById(ctx, state.GameID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read game set", err.Error())
		return
	}

	setResp, err := ParseGetGameSetByIdResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to parse game set response", err.Error())
		return
	}

	if setResp.JSON200 == nil {
		resp.Diagnostics.AddError("Unable to read game set", "Game set not found")
		return
	}

	mapGameSetFromAPI(setResp.JSON200, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
