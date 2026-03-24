terraform {
  required_providers {
    tcg-sandbox = {
      source = "registry.terraform.io/bradlet/tcg-sandbox"
    }
    # In your real terraform stack:
    #tcg-sandbox = {
    #  source  = "tcgsandbox/tcg-sandbox"
    #  version = "~>0.0.1"
    #}
  }
}

provider "tcg-sandbox" {
  # https://api.tcg-sandbox.com
  host = "http://localhost:3000"
  # Dummy key: generate your own
  api_key = "tcg_EtTxDYZOmncraEpLtu9rCR34PGIL1-YaJNn97ot8mEA"
}

# -----------------------------------------------
# Game resource — with options, rules, and grid
# -----------------------------------------------

resource "tcg-sandbox_game" "terraformed" {
  name        = "Terraformed"
  description = "A strategic land-building card game."
  attributes = {
    "land_area_sq_ft" = "number"
    "terrain_type"    = "string"
  }
  banner_image_path         = "${path.root}/assets/game_banner.jpg"
  banner_vertical_alignment = 30

  options {
    card_display_mode    = "managed"
    card_display_context = "everywhere"
  }

  rules {
    content = <<-EOT
      # Terraformed Rules

      ## Objective
      Claim the most land area by the end of the game.

      ## Turn Structure
      1. Draw a card.
      2. Play a terrain tile.
      3. Score land area.
    EOT
  }

  grid {
    player_count = 2
    slots = [
      {
        row          = 0
        column       = 0
        width        = 3
        height       = 2
        type         = "cards"
        max_count    = 5
        visibility   = "public"
        player_owner = 1
      },
      {
        row          = 0
        column       = 3
        width        = 3
        height       = 2
        type         = "cards"
        max_count    = 5
        visibility   = "public"
        player_owner = 2
      },
      {
        row        = 2
        column     = 0
        width      = 2
        height     = 1
        type       = "counters"
        max_count  = 20
        visibility = "public"
      },
    ]
  }
}

# -----------------------------------------------
# Cards — in the default "base" set
# -----------------------------------------------

resource "tcg-sandbox_card" "plains" {
  game_id     = tcg-sandbox_game.terraformed.id
  name        = "Plains"
  description = "A flat, open expanse. Gain 3 land area."
  attributes = {
    "land_area_sq_ft" = "300"
    "terrain_type"    = "grassland"
  }
}

resource "tcg-sandbox_card" "forest" {
  game_id     = tcg-sandbox_game.terraformed.id
  name        = "Forest"
  description = "Dense woodland. Gain 5 land area but costs 1 action."
  attributes = {
    "land_area_sq_ft" = "500"
    "terrain_type"    = "forest"
  }
}

resource "tcg-sandbox_card" "mountain" {
  game_id         = tcg-sandbox_game.terraformed.id
  name            = "Mountain"
  card_image_path = "${path.root}/assets/game_banner.jpg"
  description     = "Rugged peaks. Gain 8 land area but blocks adjacency."
  attributes = {
    "land_area_sq_ft" = "800"
    "terrain_type"    = "alpine"
  }
}

# -----------------------------------------------
# Lore posts
# -----------------------------------------------

resource "tcg-sandbox_lore_post" "origin_story" {
  game_id = tcg-sandbox_game.terraformed.id
  title   = "The Great Terraforming"
  content = <<-EOT
    In the age before maps, bold settlers set out to claim the wild lands.
    Armed with nothing but ambition and a deck of terrain cards, they shaped
    the world into the kingdoms we know today.
  EOT
}

resource "tcg-sandbox_lore_post" "faction_wars" {
  game_id = tcg-sandbox_game.terraformed.id
  title   = "The Faction Wars"
  content = <<-EOT
    When two settlers claimed the same mountain, the Faction Wars began.
    Only the most cunning land strategist would emerge victorious.
  EOT
}

# -----------------------------------------------
# Data source + outputs
# -----------------------------------------------

data "tcg-sandbox_game" "peeker" {
  id = tcg-sandbox_game.terraformed.id
}

output "game_id" {
  value = tcg-sandbox_game.terraformed.id
}

output "game_banner_url" {
  value = tcg-sandbox_game.terraformed.banner_image_public_url
}

output "game_options" {
  value = data.tcg-sandbox_game.peeker.options
}

output "card_plains_id" {
  value = tcg-sandbox_card.plains.id
}

output "card_forest_id" {
  value = tcg-sandbox_card.forest.id
}
