# Terraform Provider: TCG Sandbox

A Terraform provider for managing resources in the [TCG Sandbox](https://registry.terraform.io/providers/bradlet/tcg-sandbox) API — a platform for building and managing trading card games.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21
- [golangci-lint](https://golangci-lint.run/usage/install/) (for linting)

## Usage

```hcl
terraform {
  required_providers {
    tcg-sandbox = {
      source = "registry.terraform.io/bradlet/tcg-sandbox"
    }
  }
}

provider "tcg-sandbox" {
  host    = "https://api.tcgsandbox.com"
  api_key = var.tcg_sandbox_api_key
}
```

Provider configuration can also be supplied via environment variables:

| Variable               | Description        |
|------------------------|--------------------|
| `TCGSANDBOX_HOST`      | API base URL       |
| `TCGSANDBOX_API_KEY`   | API key            |

## Development

### Build & Install

```bash
make install   # build and install provider locally
make fmt       # format Go code
make lint      # run golangci-lint
```

### Code Generation

The API client ([internal/provider/client_generated.go](internal/provider/client_generated.go)) is auto-generated from the OpenAPI spec. Do not edit it directly.

```bash
make generate  # sync API spec, regenerate client, format examples, generate docs
```

### Testing

```bash
# Unit tests
make test

# Acceptance tests (requires live API credentials for local dev environment)
make testacc
```

## License

[MIT](LICENSE-MIT)
