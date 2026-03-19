package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mapCardToState maps an API Card response to Terraform state fields common
// to both the resource and data source models. The caller is responsible for
// setting any local-only fields (e.g. card_image_path) afterwards.
func mapCardToState(card *Card) (
	id types.String,
	gameID types.String,
	setID types.String,
	name types.String,
	description types.String,
	cardImagePublicURL types.String,
	attributes map[string]types.String,
) {
	id = types.StringValue(card.Id)
	gameID = types.StringValue(card.GameId)
	setID = types.StringValue(card.SetId)
	name = types.StringValue(card.Name)
	if card.Description != nil && *card.Description != "" {
		description = types.StringValue(*card.Description)
	} else {
		description = types.StringNull()
	}
	cardImagePublicURL = optionalString(card.CardImagePublicUrl)

	attributes = make(map[string]types.String, len(card.Attributes))
	for k, v := range card.Attributes {
		// Attribute values come back as interface{}, convert to string.
		switch val := v.(type) {
		case string:
			attributes[k] = types.StringValue(val)
		default:
			attributes[k] = types.StringValue(fmt.Sprintf("%v", val))
		}
	}

	return
}
