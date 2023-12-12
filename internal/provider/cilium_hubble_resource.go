// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/cilium/cilium-cli/hubble"
	"github.com/cilium/cilium-cli/k8s"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CiliumHubbleResource{}
var _ resource.ResourceWithImportState = &CiliumHubbleResource{}

func NewCiliumHubbleResource() resource.Resource {
	return &CiliumHubbleResource{}
}

// CiliumHubbleResource defines the resource implementation.
type CiliumHubbleResource struct {
	client *k8s.Client
}

// CiliumHubbleResourceModel describes the resource data model.
type CiliumHubbleResourceModel struct {
	Relay      types.Bool   `tfsdk:"relay"`
	UI         types.Bool   `tfsdk:"ui"`
	Namespace  types.String `tfsdk:"namespace"`
	Id         types.String `tfsdk:"id"`
}

func (r *CiliumHubbleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hubble"
}

func (r *CiliumHubbleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Hubble resource for Cilium. This is equivalent to cilium cli: `cilium hubble`: It manages cilium hubble",

		Attributes: map[string]schema.Attribute{
			"ui": schema.BoolAttribute{
				MarkdownDescription: "Enable Hubble UI (Default: false)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"relay": schema.BoolAttribute{
				MarkdownDescription: "Deploy Hubble Relay (Default: true)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Namespace in which to install",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("kube-system"),
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cilium hubble identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *CiliumHubbleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CiliumHubbleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CiliumHubbleResourceModel
	k8sClient := r.client
	var params = hubble.Parameters{Writer: os.Stdout}

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	namespace := data.Namespace.ValueString()
	params.Namespace = namespace
	params.UI = data.UI.ValueBool()
	params.Relay = data.Relay.ValueBool()

	if err := hubble.EnableWithHelm(context.Background(), k8sClient, params); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to enable Hubble: %s", err))
		return
	}
	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	data.Id = types.StringValue("cilium-hubble")

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumHubbleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CiliumInstallResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumHubbleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CiliumHubbleResourceModel
	k8sClient := r.client
	var params = hubble.Parameters{Writer: os.Stdout}

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	params.Namespace = namespace
	params.UI = data.UI.ValueBool()
	params.Relay = data.Relay.ValueBool()

	if err := hubble.EnableWithHelm(context.Background(), k8sClient, params); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Hubble: %s", err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumHubbleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CiliumHubbleResourceModel
	k8sClient := r.client
	var params = hubble.Parameters{Writer: os.Stdout}

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	params.Namespace = namespace

	if err := hubble.DisableWithHelm(context.Background(), k8sClient, params); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to disable Hubble: %s", err))
		return
	}
}

func (r *CiliumHubbleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
