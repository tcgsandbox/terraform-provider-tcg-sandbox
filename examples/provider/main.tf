terraform {
  required_providers {
    tcg-sandbox = {
      source = "registry.terraform.io/bradlet/tcg-sandbox"
    }
  }
}

provider "tcg-sandbox" {
  host    = "http://localhost:3000"
  api_key = "tcg_EtTxDYZOmncraEpLtu9rCR34PGIL1-YaJNn97ot8mEA"
}

# Data source example
# --------------------

data "tcg-sandbox_game" "peeker" {
  id = "ac17442e-b032-4c12-bfc6-fae7ec4f34a7"
}

output "game_name" {
  value = data.tcg-sandbox_game.peeker.name
}

output "game_banner_url" {
  value = data.tcg-sandbox_game.peeker.banner_image_public_url
}

output "game_options" {
  value = data.tcg-sandbox_game.peeker.options
}

# Resource example
# -----------------

resource "tcg-sandbox_game" "terraformed" {
  name        = "Terraformed"
  description = "A cool game about land!"
  attributes = {
    "land_area_sq_ft" : "number"
  }
  banner_image_path         = "${path.root}/assets/game_banner.jpg"
  banner_vertical_alignment = 30


}
