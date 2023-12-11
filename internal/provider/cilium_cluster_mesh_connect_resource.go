// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/cilium/cilium-cli/clustermesh"
	"github.com/cilium/cilium-cli/k8s"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	//	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	//	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CiliumClusterMeshConnectResource{}
var _ resource.ResourceWithImportState = &CiliumClusterMeshConnectResource{}

func NewCiliumClusterMeshConnectResource() resource.Resource {
	return &CiliumClusterMeshConnectResource{}
}

// ExampleResource defines the resource implementation.
type CiliumClusterMeshConnectResource struct {
	client *k8s.Client
}

// ExampleResourceModel describes the resource data model.
type CiliumClusterMeshConnectResourceModel struct {
	//SourceEndpoints      types.List `tfsdk:"source_endpoint"`
	//DestinationEndpoints types.List `tfsdk:"destination_endpoint"`
	DestinationContext types.String `tfsdk:"destination_context"`
	Namespace          types.String `tfsdk:"namespace"`
	Id                 types.String `tfsdk:"id"`
}

func (r *CiliumClusterMeshConnectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clustermesh_connection"
}

func (r *CiliumClusterMeshConnectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Install resource",

		Attributes: map[string]schema.Attribute{
			"destination_context": schema.StringAttribute{
				MarkdownDescription: "Kubernetes configuration context of destination cluster",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			//"destination_endpoint": schema.ListAttribute{
			//	ElementType:         types.StringType,
			//	MarkdownDescription: "IP of ClusterMesh service of destination cluster",
			//	Optional:            true,
			//	Computed:            true,
			//},
			//"source_endpoint": schema.ListAttribute{
			//	ElementType:         types.StringType,
			//	MarkdownDescription: "IP of ClusterMesh service of source cluster",
			//	Optional:            true,
			//	Computed:            true,
			//	Default:             listdefault.StaticValue([]),
			//},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Namespace in which to install",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("kube-system"),
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

func (r *CiliumClusterMeshConnectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CiliumClusterMeshConnectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CiliumClusterMeshConnectResourceModel
	k8sClient := r.client
	var params = clustermesh.Parameters{
		Writer: os.Stdout,
	}

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	namespace := data.Namespace.ValueString()
	params.Namespace = namespace
	params.DestinationContext = data.DestinationContext.ValueString()

	cm := clustermesh.NewK8sClusterMesh(k8sClient, params)
	if err := cm.ConnectWithHelm(context.Background()); err != nil {
		fmt.Printf("Unable to connect cluster: %v\n", err)
	}

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	data.Id = types.StringValue("ciliumclustermeshconnect-" + params.DestinationContext)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumClusterMeshConnectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CiliumClusterMeshConnectResourceModel
	var params = clustermesh.Parameters{
		Writer: os.Stdout,
	}
	k8sClient := r.client

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	params.Namespace = namespace

	cm := clustermesh.NewK8sClusterMesh(k8sClient, params)
	if _, err := cm.Status(context.Background()); err != nil {
		fmt.Printf("Unable to determine status: %s\n", err)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumClusterMeshConnectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CiliumClusterMeshConnectResourceModel
	k8sClient := r.client
	var params = clustermesh.Parameters{
		Writer: os.Stdout,
	}

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	namespace := data.Namespace.ValueString()
	params.Namespace = namespace
	params.DestinationContext = data.DestinationContext.ValueString()

	cm := clustermesh.NewK8sClusterMesh(k8sClient, params)
	if err := cm.ConnectWithHelm(context.Background()); err != nil {
		fmt.Printf("Unable to connect cluster: %v\n", err)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumClusterMeshConnectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CiliumClusterMeshConnectResourceModel
	k8sClient := r.client
	var params = clustermesh.Parameters{
		Writer: os.Stdout,
	}

	//// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	params.Namespace = namespace

	cm := clustermesh.NewK8sClusterMesh(k8sClient, params)
	if err := cm.DisconnectWithHelm(context.Background()); err != nil {
		fmt.Printf("Unable to disconnect clusters: %s\n", err)
		return
	}
}

func (r *CiliumClusterMeshConnectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
