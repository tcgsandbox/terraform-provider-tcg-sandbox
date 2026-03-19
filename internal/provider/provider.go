package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure TcgSandboxProvider satisfies various provider interfaces.
var _ provider.Provider = &TcgSandboxProvider{}

type TcgSandboxProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// TcgSandboxProviderModel describes the provider data model.
type TcgSandboxProviderModel struct {
	Host   types.String `tfsdk:"host"`
	ApiKey types.String `tfsdk:"api_key"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TcgSandboxProvider{
			version: version,
		}
	}
}

func (p *TcgSandboxProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "tcg-sandbox"
	resp.Version = p.version
}

func (p *TcgSandboxProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional: true,
			},
			"api_key": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (p *TcgSandboxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config TcgSandboxProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown configuration value",
			"Unknown configuration value for the API host. Set the value statically in the configuration, or use the TCGSANDBOX_HOST environment variable.",
		)
	}

	if config.ApiKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown configuration value",
			"Unknown configuration value for the API key. Set the value statically in the configuration, or use the TCGSANDBOX_API_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default to env vars, but override with config values
	host := os.Getenv("TCGSANDBOX_HOST")
	apiKey := os.Getenv("TCGSANDBOX_API_KEY")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.ApiKey.IsNull() {
		apiKey = config.ApiKey.ValueString()
	}

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing tcg-sandbox API host",
			"Missing or empty value for the API host. Set the value statically in the configuration, or use the TCGSANDBOX_HOST environment variable.",
		)
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing tcg-sandbox API key",
			"Missing or empty value for the API key. Set the value statically in the configuration, or use the TCGSANDBOX_HOST environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create the API client from the generated client
	// The generated client handles all API communication with proper types and validation
	client, err := NewClient(host)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create API client",
			fmt.Sprintf("Unable to create API client: %s", err.Error()),
		)
		return
	}

	// Configure the client with authentication via request editor
	// This adds the bearer token to each request
	client.RequestEditors = append(client.RequestEditors, func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
		return nil
	})

	// Make the client available to data sources and resources
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *TcgSandboxProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewGameDataSource,
		NewCardDataSource,
		NewLorePostDataSource,
	}
}

func (p *TcgSandboxProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewGameResource,
		NewCardResource,
		NewLorePostResource,
	}
}
