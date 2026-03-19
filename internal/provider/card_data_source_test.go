package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCardDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "tcg-sandbox_game" "test" {
  name                      = "Terraform Acc Card DS Test"
  description               = "Game for card data source test"
  banner_image_path         = "testdata/test_banner.png"
  banner_vertical_alignment = 50
  attributes = {
    "power" = "number"
  }
}

resource "tcg-sandbox_card" "test" {
  game_id = tcg-sandbox_game.test.id
  name    = "DS Test Card"
  attributes = {
    "power" = "75"
  }
}

data "tcg-sandbox_card" "test" {
  id      = tcg-sandbox_card.test.id
  game_id = tcg-sandbox_game.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.tcg-sandbox_card.test", "id",
						"tcg-sandbox_card.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.tcg-sandbox_card.test", "name",
						"tcg-sandbox_card.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.tcg-sandbox_card.test", "game_id",
						"tcg-sandbox_card.test", "game_id",
					),
				),
			},
		},
	})
}
