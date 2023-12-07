// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	//"net/http"

	"github.com/cilium/cilium-cli/k8s"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &CiliumProvider{}

// ScaffoldingProvider defines the provider implementation.
type CiliumProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ScaffoldingProviderModel describes the provider data model.
type CiliumProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
}

func (p *CiliumProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cilium"
	resp.Version = p.version
}

func (p *CiliumProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Example provider attribute",
				Optional:            true,
			},
		},
	}
}

func (p *CiliumProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data CiliumProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := k8s.NewClient("", "", "kube-system")
	if err != nil {
		fmt.Printf("unable to create Kubernetes client: %v\n", err)
		return
	}

	// Configuration values are now available.
	// if data.Endpoint.IsNull() { /* ... */ }

	// Example client configuration for data sources and resources
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *CiliumProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCiliumInstallResource,
		NewCiliumClusterMeshEnableResource,
		NewCiliumClusterMeshConnectResource,
	}
}

func (p *CiliumProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		//NewExampleDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CiliumProvider{
			version: version,
		}
	}
}
