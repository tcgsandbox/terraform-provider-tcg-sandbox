package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Shared model types used by both the game resource and data source.

type gameRulesModel struct {
	Content types.String `tfsdk:"content"`
}

type gameOptionsModel struct {
	CardDisplayMode    types.String `tfsdk:"card_display_mode"`
	CardDisplayContext types.String `tfsdk:"card_display_context"`
}

type gamePlayDataModel struct {
	PlayerCount types.Int64     `tfsdk:"player_count"`
	Slots       []gameSlotModel `tfsdk:"slots"`
}

type gameSlotModel struct {
	Row         types.Int64  `tfsdk:"row"`
	Column      types.Int64  `tfsdk:"column"`
	Width       types.Int64  `tfsdk:"width"`
	Height      types.Int64  `tfsdk:"height"`
	Type        types.String `tfsdk:"type"`
	MaxCount    types.Int64  `tfsdk:"max_count"`
	Visibility  types.String `tfsdk:"visibility"`
	PlayerOwner types.Int64  `tfsdk:"player_owner"`
}

// mapOptionsFromAPI converts an API GameOptions to the shared Terraform model.
func mapOptionsFromAPI(opts *GameOptions) *gameOptionsModel {
	if opts == nil {
		return nil
	}

	model := &gameOptionsModel{}

	if opts.CardDisplayMode != nil {
		model.CardDisplayMode = types.StringValue(string(*opts.CardDisplayMode))
	} else {
		model.CardDisplayMode = types.StringNull()
	}

	if opts.CardDisplayContext != nil {
		model.CardDisplayContext = types.StringValue(string(*opts.CardDisplayContext))
	} else {
		model.CardDisplayContext = types.StringNull()
	}

	return model
}

// mapGamePlayDataFromAPI converts API GamePlayData to the shared Terraform gamePlayData model.
func mapGamePlayDataFromAPI(data *GamePlayData) *gamePlayDataModel {
	if data == nil {
		return nil
	}

	gamePlayData := &gamePlayDataModel{
		PlayerCount: types.Int64Value(int64(data.PlayerCount)),
		Slots:       make([]gameSlotModel, 0, len(data.Slots)),
	}

	for _, slot := range data.Slots {
		model := gameSlotModel{
			Row:        types.Int64Value(int64(slot.Row)),
			Column:     types.Int64Value(int64(slot.Column)),
			Width:      types.Int64Value(int64(slot.Width)),
			Height:     types.Int64Value(int64(slot.Height)),
			Type:       types.StringValue(string(slot.Type)),
			MaxCount:   types.Int64Value(int64(slot.MaxCount)),
			Visibility: types.StringValue(string(slot.Visibility)),
		}

		if slot.PlayerOwner != nil {
			model.PlayerOwner = types.Int64Value(int64(*slot.PlayerOwner))
		} else {
			model.PlayerOwner = types.Int64Null()
		}

		gamePlayData.Slots = append(gamePlayData.Slots, model)
	}

	return gamePlayData
}

// optionalString returns a types.StringValue if the pointer is non-nil,
// otherwise types.StringNull.
func optionalString(s *string) types.String {
	if s != nil {
		return types.StringValue(*s)
	}
	return types.StringNull()
}

// newGameOptions converts the Terraform options model to the generated API type.
func newGameOptions(opts *gameOptionsModel) *GameOptions {
	apiOpts := &GameOptions{}
	if !opts.CardDisplayMode.IsNull() && !opts.CardDisplayMode.IsUnknown() {
		mode := CardDisplayMode(opts.CardDisplayMode.ValueString())
		apiOpts.CardDisplayMode = &mode
	}
	if !opts.CardDisplayContext.IsNull() && !opts.CardDisplayContext.IsUnknown() {
		ctx := CardDisplayContext(opts.CardDisplayContext.ValueString())
		apiOpts.CardDisplayContext = &ctx
	}
	return apiOpts
}

// newGamePlayData converts the Terraform gamePlayData model to the generated API type.
func newGamePlayData(gamePlayData *gamePlayDataModel) *GamePlayData {
	slots := make([]GridSlot, 0, len(gamePlayData.Slots))
	for _, s := range gamePlayData.Slots {
		slot := GridSlot{
			Row:        int(s.Row.ValueInt64()),
			Column:     int(s.Column.ValueInt64()),
			Width:      int(s.Width.ValueInt64()),
			Height:     int(s.Height.ValueInt64()),
			Type:       SlotType(s.Type.ValueString()),
			MaxCount:   int(s.MaxCount.ValueInt64()),
			Visibility: SlotVisibility(s.Visibility.ValueString()),
		}
		if !s.PlayerOwner.IsNull() && !s.PlayerOwner.IsUnknown() {
			po := int(s.PlayerOwner.ValueInt64())
			slot.PlayerOwner = &po
		}
		slots = append(slots, slot)
	}
	return &GamePlayData{
		PlayerCount: int(gamePlayData.PlayerCount.ValueInt64()),
		Slots:       slots,
	}
}
