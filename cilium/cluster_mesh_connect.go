// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cilium/cilium/cilium-cli/clustermesh"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
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

// CiliumClusterMeshConnectResource defines the resource implementation.
type CiliumClusterMeshConnectResource struct {
	client *CiliumClient
}

// CiliumClusterMeshConnectResourceModel describes the resource data model.
type CiliumClusterMeshConnectResourceModel struct {
	//SourceEndpoints      types.List `tfsdk:"source_endpoint"`
	//DestinationEndpoints types.List `tfsdk:"destination_endpoint"`
	DestinationContexts types.List   `tfsdk:"destination_contexts"`
	Parallel            types.Int32  `tfsdk:"parallel"`
	ConnectionMode      types.String `tfsdk:"connection_mode"`
	Id                  types.String `tfsdk:"id"`
}

func (r *CiliumClusterMeshConnectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clustermesh_connection"
}

func (r *CiliumClusterMeshConnectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Cluster Mesh connection resource. This is equivalent to cilium cli: `cilium clustermesh connect` and `cilium clustermesh disconnect`: It manages the connections between two Kubernetes clusters.",

		Attributes: map[string]schema.Attribute{
			"destination_contexts": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Kubernetes configuration contexts of destination clusters",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListNull(types.StringType)),
			},
			"parallel": schema.Int32Attribute{
				MarkdownDescription: ConcatDefault("Number of parallel connections of destination clusters", "1"),
				Optional:            true,
				Computed:            true,
				Default:             int32default.StaticInt32(1),
			},
			"connection_mode": schema.StringAttribute{
				MarkdownDescription: ConcatDefault("Connection mode. unicast, bidirectional and mesh", "bidirectional"),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("bidirectional"),
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
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cilium ClusterMesh Connection identifier",
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

	client, ok := req.ProviderData.(*CiliumClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *CiliumClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *CiliumClusterMeshConnectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CiliumClusterMeshConnectResourceModel
	c := r.client
	if c == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to connect to kubernetes")
		return
	}
	k8sClient, namespace, helm_release := c.client, c.namespace, c.helm_release
	var params = clustermesh.Parameters{
		Writer: os.Stdout,
	}

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	params.Namespace = namespace
	params.HelmReleaseName = helm_release

	params.DestinationContext = ValueList(ctx, data.DestinationContexts)
	params.ConnectionMode = data.ConnectionMode.ValueString()
	params.Parallel = int(data.Parallel.ValueInt32())

	cm := clustermesh.NewK8sClusterMesh(k8sClient, params)
	if err := cm.ConnectWithHelm(context.Background()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to connect cluster: %s", err))
		return
	}

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	data.Id = types.StringValue("ciliumclustermeshconnect-" + strings.Join(params.DestinationContext, "-"))

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumClusterMeshConnectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CiliumClusterMeshConnectResourceModel
	c := r.client
	if c == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to connect to kubernetes")
		return
	}
	k8sClient, namespace, helm_release := c.client, c.namespace, c.helm_release
	var params = clustermesh.Parameters{
		Writer: os.Stdout,
	}

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params.Namespace = namespace
	params.HelmReleaseName = helm_release
	params.Wait = true
	params.WaitDuration = 20 * time.Second

	cm := clustermesh.NewK8sClusterMesh(k8sClient, params)
	if _, err := cm.Status(context.Background()); err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumClusterMeshConnectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CiliumClusterMeshConnectResourceModel
	c := r.client
	if c == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to connect to kubernetes")
		return
	}
	k8sClient, namespace, helm_release := c.client, c.namespace, c.helm_release
	var params = clustermesh.Parameters{
		Writer: os.Stdout,
	}

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	params.Namespace = namespace
	params.DestinationContext = ValueList(ctx, data.DestinationContexts)
	params.ConnectionMode = data.ConnectionMode.ValueString()
	params.HelmReleaseName = helm_release

	cm := clustermesh.NewK8sClusterMesh(k8sClient, params)
	if err := cm.ConnectWithHelm(context.Background()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to connect clusters: %s", err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumClusterMeshConnectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CiliumClusterMeshConnectResourceModel
	c := r.client
	if c == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to connect to kubernetes")
		return
	}
	k8sClient, namespace, helm_release := c.client, c.namespace, c.helm_release
	var params = clustermesh.Parameters{
		Writer: os.Stdout,
	}

	//// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params.Namespace = namespace
	params.HelmReleaseName = helm_release
	params.ConnectionMode = data.ConnectionMode.ValueString()
	params.DestinationContext = ValueList(ctx, data.DestinationContexts)

	cm := clustermesh.NewK8sClusterMesh(k8sClient, params)
	if err := cm.DisconnectWithHelm(context.Background()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to disconnect clusters: %s", err))
		return
	}
}

func (r *CiliumClusterMeshConnectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
