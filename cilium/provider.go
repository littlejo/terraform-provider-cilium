// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/cilium/cilium/cilium-cli/k8s"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	Context       types.String `tfsdk:"context"`
	ConfigPath    types.String `tfsdk:"config_path"`
	ConfigContent types.String `tfsdk:"config_content"`
	Namespace     types.String `tfsdk:"namespace"`
	HelmRelease   types.String `tfsdk:"helm_release"`
}

func (p *CiliumProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cilium"
	resp.Version = p.version
}

func (p *CiliumProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"context": schema.StringAttribute{
				MarkdownDescription: ConcatDefault("Context of kubeconfig file", "default context"),
				Optional:            true,
			},
			"config_path": schema.StringAttribute{
				MarkdownDescription: ConcatDefault("A path to a kube config file", "~/.kube/config"),
				Optional:            true,
			},
			"config_content": schema.StringAttribute{
				MarkdownDescription: ConcatDefault("The content of kube config file", ""),
				Optional:            true,
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: ConcatDefault("Namespace to install cilium", "kube-system"),
				Optional:            true,
			},
			"helm_release": schema.StringAttribute{
				MarkdownDescription: ConcatDefault("Helm Release to install cilium", "cilium"),
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

	// Provider configuration may depend on resources that are not created yet
	// (e.g. the Kubernetes cluster): Terraform configures the provider again
	// once the values are known.
	if data.Context.IsUnknown() || data.ConfigPath.IsUnknown() || data.ConfigContent.IsUnknown() || data.Namespace.IsUnknown() || data.HelmRelease.IsUnknown() {
		return
	}

	namespace := data.Namespace.ValueString()
	config_path := data.ConfigPath.ValueString()
	context := data.Context.ValueString()
	helm_release := data.HelmRelease.ValueString()
	config_content := data.ConfigContent.ValueString()

	if helm_release == "" {
		helm_release = "cilium"
	}
	if namespace == "" {
		namespace = "kube-system"
	}
	if config_content != "" {
		config_content_str, err := base64.StdEncoding.DecodeString(config_content)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("config_content"), "Invalid config_content", fmt.Sprintf("config_content must be a base64-encoded kubeconfig: %s", err))
			return
		}
		f, err := writeTempFile("kubeconfig", config_content_str)
		if err != nil {
			resp.Diagnostics.AddError("Provider Configuration Error", fmt.Sprintf("Unable to write kubeconfig to a temporary file: %s", err))
			return
		}
		tempFilesMu.Lock()
		tempFiles = append(tempFiles, f)
		tempFilesMu.Unlock()
		config_path = f
	}
	if config_path != "" {
		os.Setenv("KUBECONFIG", config_path)
	}

	impersonate_as := ""
	impersonate_groups := []string{}

	client, err := k8s.NewClient(context, config_path, namespace, impersonate_as, impersonate_groups)
	if err != nil {
		tflog.Warn(ctx, "Unable to create Kubernetes client", map[string]any{"error": err.Error()})
	}

	ciliumClient := &CiliumClient{client: client, namespace: namespace, helm_release: helm_release, err: err}
	resp.DataSourceData = ciliumClient
	resp.ResourceData = ciliumClient
}

func (p *CiliumProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCiliumInstallResource,
		NewCiliumConfigResource,
		NewCiliumClusterMeshEnableResource,
		NewCiliumClusterMeshConnectResource,
		NewCiliumHubbleResource,
		NewCiliumKubeProxyDisabledResource,
	}
}

func (p *CiliumProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCiliumHelmValuesDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CiliumProvider{
			version: version,
		}
	}
}
