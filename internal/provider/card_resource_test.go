package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCardResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify — uses implicit "base" set
			{
				Config: providerConfig + `
resource "tcg-sandbox_game" "test" {
  name                      = "Terraform Acc Card Test"
  description               = "Game for card acceptance test"
  banner_image_path         = "testdata/test_banner.png"
  banner_vertical_alignment = 50
  attributes = {
    "power" = "number"
  }
}

resource "tcg-sandbox_card" "test" {
  game_id = tcg-sandbox_game.test.id
  name    = "Test Card"
  description = "A test card"
  attributes = {
    "power" = "50"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("tcg-sandbox_card.test", "id"),
					resource.TestCheckResourceAttrSet("tcg-sandbox_card.test", "game_id"),
					resource.TestCheckResourceAttr("tcg-sandbox_card.test", "set_id", "base"),
					resource.TestCheckResourceAttr("tcg-sandbox_card.test", "name", "Test Card"),
					resource.TestCheckResourceAttr("tcg-sandbox_card.test", "description", "A test card"),
					resource.TestCheckResourceAttr("tcg-sandbox_card.test", "attributes.power", "50"),
				),
			},
		},
	})
}
