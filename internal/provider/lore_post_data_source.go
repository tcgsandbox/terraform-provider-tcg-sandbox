package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &lorePostDataSource{}
	_ datasource.DataSourceWithConfigure = &lorePostDataSource{}
)

func NewLorePostDataSource() datasource.DataSource {
	return &lorePostDataSource{}
}

type lorePostDataSource struct {
	client *Client
}

type lorePostDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	GameID types.String `tfsdk:"game_id"`
	Title  types.String `tfsdk:"title"`
}

func (d *lorePostDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *lorePostDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lore_post"
}

func (d *lorePostDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a lore post from a TCG Sandbox game.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the lore post.",
				Required:            true,
			},
			"game_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the game this lore post belongs to.",
				Required:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The title of the lore post.",
				Computed:            true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *lorePostDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state lorePostDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.client.GetLorePost(ctx, state.GameID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Lore Post", err.Error())
		return
	}

	getResp, err := ParseGetLorePostResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Parse Lore Post Response", err.Error())
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("Unable to Read Lore Post", "Lore post not found")
		return
	}

	post := getResp.JSON200
	mapLorePostToState(post, &state.ID, &state.GameID, &state.Title)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
