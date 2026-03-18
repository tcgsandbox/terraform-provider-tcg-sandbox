package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGameDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "tcg-sandbox_game" "test" {
  name                      = "Terraform Acc DS Test Game"
  description               = "Data source acceptance test"
  banner_image_path         = "testdata/test_banner.png"
  banner_vertical_alignment = 30
  attributes = {
    "speed" = "number"
  }
}

data "tcg-sandbox_game" "test" {
  id = tcg-sandbox_game.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.tcg-sandbox_game.test", "id",
						"tcg-sandbox_game.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.tcg-sandbox_game.test", "name",
						"tcg-sandbox_game.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.tcg-sandbox_game.test", "description",
						"tcg-sandbox_game.test", "description",
					),
					resource.TestCheckResourceAttrPair(
						"data.tcg-sandbox_game.test", "banner_image_public_url",
						"tcg-sandbox_game.test", "banner_image_public_url",
					),
					resource.TestCheckResourceAttr("data.tcg-sandbox_game.test", "banner_vertical_alignment", "30"),
				),
			},
		},
	})
}
