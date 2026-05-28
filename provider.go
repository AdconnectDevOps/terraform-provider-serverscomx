package main

import (
	"context"
	"os"

	"github.com/AdconnectDevOps/terraform-provider-serverscom-extras/serverscom"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ provider.Provider = &ServersComExtrasProvider{}
)

type ServersComExtrasProvider struct {
	version string
}

type ServersComExtrasProviderModel struct {
	Token           types.String `tfsdk:"token"`
	Endpoint        types.String `tfsdk:"endpoint"`
	RequestInterval types.Int64  `tfsdk:"request_interval"`
}

func (p *ServersComExtrasProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "serverscom"
	resp.Version = p.version
}

func (p *ServersComExtrasProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Gap-fill Terraform provider for Servers.com Public API endpoints missing from the official serverscom/serverscom provider. Currently exposes PTR record management on dedicated servers; resource set will grow as other gaps are identified.",
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Description: "Servers.com API token. Falls back to SERVERSCOM_TOKEN environment variable when not set.",
				Optional:    true,
				Sensitive:   true,
			},
			"endpoint": schema.StringAttribute{
				Description: "Servers.com API endpoint. Defaults to https://api.servers.com/v1.",
				Optional:    true,
			},
			"request_interval": schema.Int64Attribute{
				Description: "Minimum interval between API requests in seconds. Defaults to 1.",
				Optional:    true,
			},
		},
	}
}

func (p *ServersComExtrasProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ServersComExtrasProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token := config.Token.ValueString()
	if token == "" {
		token = os.Getenv("SERVERSCOM_TOKEN")
	}
	if token == "" {
		resp.Diagnostics.AddError(
			"Missing Servers.com API token",
			"Provider config has no token set and SERVERSCOM_TOKEN environment variable is empty.",
		)
		return
	}

	endpoint := config.Endpoint.ValueString()
	if endpoint == "" {
		endpoint = "https://api.servers.com/v1"
	}

	requestInterval := int64(1)
	if !config.RequestInterval.IsNull() {
		requestInterval = config.RequestInterval.ValueInt64()
		if requestInterval < 1 {
			requestInterval = 1
		}
	}

	client := serverscom.NewClient(token, endpoint, requestInterval)

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ServersComExtrasProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		serverscom.NewPtrRecordResource,
	}
}

func (p *ServersComExtrasProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ServersComExtrasProvider{version: version}
	}
}
