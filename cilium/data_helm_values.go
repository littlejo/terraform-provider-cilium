// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/cilium/cilium-cli/k8s"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &CiliumHelmValuesDataSource{}

func NewCiliumHelmValuesDataSource() datasource.DataSource {
	return &CiliumHelmValuesDataSource{}
}

// ExampleDataSource defines the data source implementation.
type CiliumHelmValuesDataSource struct {
	client *k8s.Client
}

// ExampleDataSourceModel describes the data source data model.
type CiliumHelmValuesDataSourceModel struct {
	Release   types.String `tfsdk:"release"`
	Namespace types.String `tfsdk:"namespace"`
	Yaml      types.String `tfsdk:"yaml"`
}

func (d *CiliumHelmValuesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_helm_values"
}

func (d *CiliumHelmValuesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Helm values of cilium",

		Attributes: map[string]schema.Attribute{
			"release": schema.StringAttribute{
				MarkdownDescription: ConcatDefault("Helm release", "cilium"),
				Optional:            true,
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: ConcatDefault("Namespace of cilium", "kube-system"),
				Optional:            true,
			},
			"yaml": schema.StringAttribute{
				MarkdownDescription: "Yaml output",
				Computed:            true,
			},
		},
	}
}

func (d *CiliumHelmValuesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*k8s.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *CiliumHelmValuesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CiliumHelmValuesDataSourceModel
	k8sClient := d.client
	if k8sClient == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to connect to kubernetes")
		return
	}

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	actionConfig := action.Configuration{}
	logger := func(format string, v ...interface{}) {}
	namespace := data.Namespace.ValueString()
	if namespace == "" {
		namespace = "kube-system"
	}
	release := data.Release.ValueString()
	if release == "" {
		release = "cilium"
	}
	helmDriver := ""
	if err := actionConfig.Init(k8sClient.RESTClientGetter, namespace, helmDriver, logger); err != nil {
		return
	}

	client := action.NewGetValues(&actionConfig)

	vals, err := client.Run(release)

	if err != nil {
		return
	}

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	yaml, err := yaml.Marshal(vals)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed: %s", err))
		return
	}

	data.Yaml = types.StringValue(string(yaml))
	data.Namespace = types.StringValue(namespace)
	data.Release = types.StringValue(release)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
