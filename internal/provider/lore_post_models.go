package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Shared model types used by both the lore_post resource and data source.

// mapLorePostToState maps an API LorePost response to common Terraform state fields.
func mapLorePostToState(post *LorePost, id *types.String, gameID *types.String, title *types.String) {
	*id = types.StringValue(post.Id)
	*gameID = types.StringValue(post.GameId)
	*title = types.StringValue(post.Title)
}
