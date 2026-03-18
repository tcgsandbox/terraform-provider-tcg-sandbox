package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"tcg-sandbox": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("TCGSANDBOX_HOST") == "" {
		t.Fatal("TCGSANDBOX_HOST must be set for acceptance tests")
	}
	if os.Getenv("TCGSANDBOX_API_KEY") == "" {
		t.Fatal("TCGSANDBOX_API_KEY must be set for acceptance tests")
	}
}
