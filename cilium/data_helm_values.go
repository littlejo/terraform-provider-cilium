// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

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
	client *CiliumClient
}

// ExampleDataSourceModel describes the data source data model.
type CiliumHelmValuesDataSourceModel struct {
	Yaml types.String `tfsdk:"yaml"`
}

func (d *CiliumHelmValuesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_helm_values"
}

func (d *CiliumHelmValuesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Helm values of cilium",

		Attributes: map[string]schema.Attribute{
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

	client, ok := req.ProviderData.(*CiliumClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *CiliumClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *CiliumHelmValuesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CiliumHelmValuesDataSourceModel
	c := d.client
	if err := c.Ready(); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	helm_values, err := c.GetHelmValues()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Helm values of release %s/%s: %s", c.namespace, c.helm_release, err))
		return
	}

	data.Yaml = types.StringValue(helm_values)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
