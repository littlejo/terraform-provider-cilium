// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/cilium/cilium-cli/install"
	"github.com/cilium/cilium-cli/k8s"
	"helm.sh/helm/v3/pkg/cli/values"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CiliumInstallResource{}
var _ resource.ResourceWithImportState = &CiliumInstallResource{}

func NewCiliumInstallResource() resource.Resource {
	return &CiliumInstallResource{}
}

// ExampleResource defines the resource implementation.
type CiliumInstallResource struct {
	client *k8s.Client
}

// ExampleResourceModel describes the resource data model.
type CiliumInstallResourceModel struct {
	Version                    types.String `tfsdk:"version"`
	AzureResourceGroupName     types.String `tfsdk:"azure_resource_group_name"`
	ClusterId                  types.String `tfsdk:"cluster_id"`
	ClusterName                types.String `tfsdk:"cluster_name"`
	ClusterPoolIpv4PodCidrList types.List   `tfsdk:"cluster_pool_ipv4_pod_cidr_list"`
	Id                         types.String `tfsdk:"id"`
}

func (r *CiliumInstallResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_install"
}

func (r *CiliumInstallResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Install resource",

		Attributes: map[string]schema.Attribute{
			"version": schema.StringAttribute{
				MarkdownDescription: "Version of Cilium",
				Required:            true,
			},
			"azure_resource_group_name": schema.StringAttribute{
				MarkdownDescription: "Azure Resource Group Name",
				Optional:            true,
				Computed:            true,
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "Cluster Id (useful to modify for cluster mesh)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("0"),
			},
			"cluster_name": schema.StringAttribute{
				MarkdownDescription: "Cluster Name (useful to modify for Cluster Mesh)",
				Optional:            true,
				Computed:            true,
			},
			"cluster_pool_ipv4_pod_cidr_list": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of CIDR for pod",
				Optional:            true,
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Example identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *CiliumInstallResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*k8s.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *k8s.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *CiliumInstallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CiliumInstallResourceModel
	var params = install.Parameters{Writer: os.Stdout}
	var options values.Options
	params.Namespace = "kube-system"
	params.APIVersions = []string{"v1"}
	params.HelmValuesSecretName = "cilium"

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	params.Version = data.Version.ValueString()
	params.Azure.ResourceGroupName = data.AzureResourceGroupName.ValueString()
	clusterId := data.ClusterId.ValueString()
	clusterName := data.ClusterName.ValueString()
	clusterPoolIpv4PodCidrList := data.ClusterPoolIpv4PodCidrList.Elements()
	a := "{"
	for i, element := range clusterPoolIpv4PodCidrList {
		if i > 0 {
			a += ","
		}
		a += element.String()
	}
	a += "}"
	fmt.Printf("clusterPoolIpv4PodCidrList: %v\n", a)

	options.Values = []string{"cluster.id=" + clusterId, "cluster.name=" + clusterName, "ipam.operator.clusterPoolIPv4PodCIDRList=" + a}
	params.HelmOpts = options

	installer, err := install.NewK8sInstaller(r.client, params)
	if err != nil {
		fmt.Printf("unable to create Cilium installer: %v\n", err)
		return
	}

	if err := installer.InstallWithHelm(context.Background(), r.client); err != nil {
		fmt.Printf("Unable to install Cilium: %v\n", err)
		return
	}
	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	data.Id = types.StringValue("cilium")

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumInstallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CiliumInstallResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumInstallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CiliumInstallResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumInstallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CiliumInstallResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }
}

func (r *CiliumInstallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
