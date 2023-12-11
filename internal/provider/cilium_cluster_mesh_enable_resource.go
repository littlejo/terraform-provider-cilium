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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CiliumClusterMeshEnableResource{}
var _ resource.ResourceWithImportState = &CiliumClusterMeshEnableResource{}

func NewCiliumClusterMeshEnableResource() resource.Resource {
	return &CiliumClusterMeshEnableResource{}
}

// ExampleResource defines the resource implementation.
type CiliumClusterMeshEnableResource struct {
	client *k8s.Client
}

// ExampleResourceModel describes the resource data model.
type CiliumClusterMeshEnableResourceModel struct {
	EnableExternalWorkloads types.Bool   `tfsdk:"enable_external_workloads"`
	EnableKVStoreMesh       types.Bool   `tfsdk:"enable_kv_store_mesh"`
	ServiceType             types.String `tfsdk:"service_type"`
	Namespace               types.String `tfsdk:"namespace"`
	Id                      types.String `tfsdk:"id"`
}

func (r *CiliumClusterMeshEnableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clustermesh"
}

func (r *CiliumClusterMeshEnableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Install resource",

		Attributes: map[string]schema.Attribute{
			"enable_external_workloads": schema.BoolAttribute{
				MarkdownDescription: "Enable support for external workloads, such as VMs",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"enable_kv_store_mesh": schema.BoolAttribute{
				MarkdownDescription: "Enable kvstoremesh, an extension which caches remote cluster information in the local kvstore (Cilium >=1.14 only)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"service_type": schema.StringAttribute{
				MarkdownDescription: "Type of Kubernetes service to expose control plane { LoadBalancer | NodePort | ClusterIP }",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
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

func (r *CiliumClusterMeshEnableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CiliumClusterMeshEnableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CiliumClusterMeshEnableResourceModel
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
	params.ServiceType = data.ServiceType.ValueString()
	params.EnableKVStoreMesh = data.EnableKVStoreMesh.ValueBool() //
	params.EnableExternalWorkloads = data.EnableExternalWorkloads.ValueBool()

	ctxb := context.Background()
	if err := clustermesh.EnableWithHelm(ctxb, k8sClient, params); err != nil {
		fmt.Printf("unable to create Cilium installer: %v\n", err)
		return
	}

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	data.Id = types.StringValue("ciliumclustermeshenable")

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumClusterMeshEnableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CiliumClusterMeshEnableResourceModel
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
		resp.State.RemoveResource(ctx)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumClusterMeshEnableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CiliumClusterMeshEnableResourceModel
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
	params.ServiceType = data.ServiceType.ValueString()
	params.EnableKVStoreMesh = data.EnableKVStoreMesh.ValueBool() //
	params.EnableExternalWorkloads = data.EnableExternalWorkloads.ValueBool()

	ctxb := context.Background()
	if err := clustermesh.EnableWithHelm(ctxb, k8sClient, params); err != nil {
		fmt.Printf("unable to create Cilium installer: %v\n", err)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumClusterMeshEnableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CiliumClusterMeshEnableResourceModel
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
	ctxb := context.Background()

	if err := clustermesh.DisableWithHelm(ctxb, k8sClient, params); err != nil {
		fmt.Printf("Unable to determine status: %s\n", err)
		return
	}
}

func (r *CiliumClusterMeshEnableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
