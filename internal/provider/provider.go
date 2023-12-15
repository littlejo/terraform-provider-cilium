// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/cilium/cilium-cli/k8s"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure CiliumProvider satisfies various provider interfaces.
var _ provider.Provider = &CiliumProvider{}

// CiliumProvider defines the provider implementation.
type CiliumProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// CiliumProviderModel describes the provider data model.
type CiliumProviderModel struct {
	Context    types.String `tfsdk:"context"`
	ConfigPath types.String `tfsdk:"config_path"`
	Namespace  types.String `tfsdk:"namespace"`
}

func (p *CiliumProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cilium"
	resp.Version = p.version
}

func (p *CiliumProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"context": schema.StringAttribute{
				MarkdownDescription: "Context of kubeconfig file.",
				Optional:            true,
			},
			"config_path": schema.StringAttribute{
				MarkdownDescription: "A path to a kube config file.",
				Optional:            true,
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Namespace to install cilium (kube-system by default).",
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

	namespace := data.Namespace.ValueString()
	if namespace == "" {
		namespace = "kube-system"
	}
	config_path := data.ConfigPath.ValueString()
	context := data.Context.ValueString()

	if config_path != "" {
		os.Setenv("KUBECONFIG", config_path)
	}

	client, err := k8s.NewClient(context, config_path, namespace)
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
		NewCiliumConfigResource,
		NewCiliumClusterMeshEnableResource,
		NewCiliumClusterMeshConnectResource,
		NewCiliumHubbleResource,
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
