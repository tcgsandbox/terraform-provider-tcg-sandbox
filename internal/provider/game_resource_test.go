package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const providerConfig = `
provider "tcg-sandbox" {}
`

func TestAccGameResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: providerConfig + `
resource "tcg-sandbox_game" "test" {
  name                      = "Terraform Acc Test Game"
  description               = "Created by acceptance test"
  banner_image_path         = "testdata/test_banner.png"
  banner_vertical_alignment = 50
  attributes = {
    "power" = "number"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("tcg-sandbox_game.test", "id"),
					resource.TestCheckResourceAttrSet("tcg-sandbox_game.test", "owner"),
					resource.TestCheckResourceAttrSet("tcg-sandbox_game.test", "banner_image_public_url"),
					resource.TestCheckResourceAttr("tcg-sandbox_game.test", "name", "Terraform Acc Test Game"),
					resource.TestCheckResourceAttr("tcg-sandbox_game.test", "description", "Created by acceptance test"),
					resource.TestCheckResourceAttr("tcg-sandbox_game.test", "banner_vertical_alignment", "50"),
					resource.TestCheckResourceAttr("tcg-sandbox_game.test", "attributes.power", "number"),
				),
			},
			// Update name and description
			{
				Config: providerConfig + `
resource "tcg-sandbox_game" "test" {
  name                      = "Updated Acc Test Game"
  description               = "Updated by acceptance test"
  banner_image_path         = "testdata/test_banner.png"
  banner_vertical_alignment = 50
  attributes = {
    "power" = "number"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tcg-sandbox_game.test", "name", "Updated Acc Test Game"),
					resource.TestCheckResourceAttr("tcg-sandbox_game.test", "description", "Updated by acceptance test"),
				),
			},
			// Import state
			{
				ResourceName:            "tcg-sandbox_game.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"banner_image_path", "rules"},
			},
		},
	})
}
