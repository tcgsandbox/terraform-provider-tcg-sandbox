package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &gameDataSource{}
	_ datasource.DataSourceWithConfigure = &gameDataSource{}
)

func NewGameDataSource() datasource.DataSource {
	return &gameDataSource{}
}

type gameDataSource struct {
	client *Client
}

type gameDataSourceModel struct {
	ID                      types.String       `tfsdk:"id"`
	Owner                   types.String       `tfsdk:"owner"`
	Name                    types.String       `tfsdk:"name"`
	Description             types.String       `tfsdk:"description"`
	BannerImagePublicUrl    types.String       `tfsdk:"banner_image_public_url"`
	BannerVerticalAlignment types.Int64        `tfsdk:"banner_vertical_alignment"`
	Playable                types.Bool         `tfsdk:"playable"`
	GamePlayData            *gamePlayDataModel `tfsdk:"game_play_data"`
	Options                 *gameOptionsModel  `tfsdk:"options"`
}

func (d *gameDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *gameDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_game"
}

func (d *gameDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the game.",
				Required:            true,
			},
			"owner": schema.StringAttribute{
				MarkdownDescription: "The user's unique ID who owns this game.",
				Computed:            true,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the game.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the game.",
				Computed:            true,
			},
			"banner_image_public_url": schema.StringAttribute{
				MarkdownDescription: "Path to the banner image in the publicly visible managed folder of GCS.",
				Computed:            true,
			},
			"banner_vertical_alignment": schema.Int64Attribute{
				MarkdownDescription: "Banner vertical alignment.",
				Computed:            true,
			},
			"playable": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether the game is ready to be played (has sufficient content like cards, sets, etc.).",
				Computed:            true,
			},
			"game_play_data": schema.SingleNestedAttribute{
				MarkdownDescription: "The grid configuration and player count settings for the game.",
				Computed:            true,
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"player_count": schema.Int64Attribute{
						MarkdownDescription: "The current player count setting for the game.",
						Computed:            true,
					},
					"slots": schema.ListNestedAttribute{
						MarkdownDescription: "Array of grid slots defining the game board layout.",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"row": schema.Int64Attribute{
									MarkdownDescription: "The row position of the slot in the grid (0-based index).",
									Computed:            true,
								},
								"column": schema.Int64Attribute{
									MarkdownDescription: "The column position of the slot in the grid (0-based index).",
									Computed:            true,
								},
								"width": schema.Int64Attribute{
									MarkdownDescription: "The width of the slot in grid units (how many columns it spans).",
									Computed:            true,
								},
								"height": schema.Int64Attribute{
									MarkdownDescription: "The height of the slot in grid units (how many rows it spans).",
									Computed:            true,
								},
								"type": schema.StringAttribute{
									MarkdownDescription: "The types of content a grid slot can hold.",
									Computed:            true,
								},
								"max_count": schema.Int64Attribute{
									MarkdownDescription: "The maximum number of items this slot can hold.",
									Computed:            true,
								},
								"visibility": schema.StringAttribute{
									MarkdownDescription: "Whether a grid slot is visible to all players (public) or only to its owner (private).",
									Computed:            true,
								},
								"player_owner": schema.Int64Attribute{
									MarkdownDescription: "The player number (1-based) who owns this slot, or null if no player owns it.",
									Computed:            true,
								},
							},
						},
					},
				},
			},
			"options": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration options for how the game displays cards and other elements.",
				Computed:            true,
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"card_display_mode": schema.StringAttribute{
						MarkdownDescription: "Controls how cards are displayed in the game (managed or imageonly).",
						Computed:            true,
					},
					"card_display_context": schema.StringAttribute{
						MarkdownDescription: "Controls where the display mode applies (everywhere or ingameonly).",
						Computed:            true,
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *gameDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gameDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.client.GetGameById(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Game", err.Error())
		return
	}

	gameResp, err := ParseGetGameByIdResponse(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Parse Game Response", err.Error())
		return
	}

	if gameResp.JSON200 == nil {
		resp.Diagnostics.AddError("Unable to Read Game", "Game not found")
		return
	}

	game := gameResp.JSON200

	state.ID = types.StringValue(game.Id)
	state.Name = types.StringValue(game.Name)
	state.Description = types.StringValue(game.Description)
	state.BannerImagePublicUrl = types.StringValue(game.BannerImagePublicUrl)
	state.BannerVerticalAlignment = types.Int64Value(int64(game.BannerVerticalAlignment))
	state.Playable = types.BoolValue(game.Playable)
	state.Owner = optionalString(game.Owner)
	state.GamePlayData = mapGamePlayDataFromAPI(game.GamePlayData)
	state.Options = mapOptionsFromAPI(game.Options)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
