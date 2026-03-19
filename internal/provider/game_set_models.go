package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Shared model types used by both the game_set resource and data source.

type gameSetModel struct {
	ID          types.String            `tfsdk:"id"`
	GameID      types.String            `tfsdk:"game_id"`
	Name        types.String            `tfsdk:"name"`
	Description types.String            `tfsdk:"description"`
	Attributes  map[string]types.String `tfsdk:"attributes"`
}

// mapGameSetFromAPI maps an API GameSet response to the shared Terraform model.
func mapGameSetFromAPI(set *GameSet, state *gameSetModel) {
	state.ID = types.StringValue(set.Id)
	state.GameID = types.StringValue(set.GameId)
	state.Name = types.StringValue(set.Name)
	state.Description = optionalString(set.Description)

	attrs := make(map[string]types.String, len(set.Attributes))
	for k, v := range set.Attributes {
		attrs[k] = types.StringValue(string(v))
	}
	state.Attributes = attrs
}
