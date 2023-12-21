// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/cilium/cilium-cli/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"

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
var _ resource.Resource = &CiliumKubeProxyDisabledResource{}
var _ resource.ResourceWithImportState = &CiliumKubeProxyDisabledResource{}

func NewCiliumKubeProxyDisabledResource() resource.Resource {
	return &CiliumKubeProxyDisabledResource{}
}

// CiliumKubeProxyDisabledResource defines the resource implementation.
type CiliumKubeProxyDisabledResource struct {
	client *k8s.Client
}

// CiliumInstallResourceModel describes the resource data model.
type CiliumKubeProxyDisabledResourceModel struct {
	Name      types.String `tfsdk:"name"`
	Namespace types.String `tfsdk:"namespace"`
	Id        types.String `tfsdk:"id"`
}

func (r *CiliumKubeProxyDisabledResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubeproxy_free"
}

func (r *CiliumKubeProxyDisabledResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Disable Kube-Proxy DaemonSet, equivalent to: kubectl -n kube-system patch daemonset kube-proxy -p '\"spec\": {\"template\": {\"spec\": {\"nodeSelector\": {\"non-existing\": \"true\"}}}}'",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of DaemonSet (Default: kube-proxy)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("kube-proxy"),
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Namespace in which to install (Default: kube-system)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("kube-system"),
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cilium install identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *CiliumKubeProxyDisabledResource) CheckDaemonsetStatus(ctx context.Context, namespace, daemonset string) error {
	k8sClient := r.client
	d, _ := k8sClient.GetDaemonSet(ctx, namespace, daemonset, metav1.GetOptions{})
	if d == nil {
		return nil
	}

	if d.Status.NumberReady != 0 {
		return fmt.Errorf("replicas count is not zero")
	}

	return nil
}

func (r *CiliumKubeProxyDisabledResource) CheckDaemonsetAvailability(ctx context.Context, namespace, daemonset string) error {
	k8sClient := r.client
	d, err := k8sClient.GetDaemonSet(ctx, namespace, daemonset, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if d == nil {
		return fmt.Errorf("daemonset is not available")
	}

	return nil
}

func (r *CiliumKubeProxyDisabledResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CiliumKubeProxyDisabledResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CiliumKubeProxyDisabledResourceModel
	k8sClient := r.client

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	namespace := data.Namespace.ValueString()
	name := data.Name.ValueString()
	nodeSelectorKey := "non-existing"
	nodeSelectorValue := "true"
	patch := []byte(fmt.Sprintf(`{"spec":{"template":{"spec":{"nodeSelector":{"%s":"%s"}}}}}`, nodeSelectorKey, nodeSelectorValue))

	if err := r.CheckDaemonsetAvailability(ctx, namespace, name); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("%s", err))
	}

	if _, err := k8sClient.PatchDaemonSet(ctx, namespace, name, ktypes.StrategicMergePatchType, patch, metav1.PatchOptions{FieldManager: "Terraform"}); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("%s", err))
		return
	}

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	data.Id = types.StringValue("cilium-kubeproxy-less")

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumKubeProxyDisabledResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CiliumKubeProxyDisabledResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	namespace := data.Namespace.ValueString()
	name := data.Name.ValueString()
	if err := r.CheckDaemonsetStatus(ctx, namespace, name); err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumKubeProxyDisabledResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CiliumKubeProxyDisabledResourceModel
	k8sClient := r.client

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	namespace := data.Namespace.ValueString()
	name := data.Name.ValueString()
	nodeSelectorKey := "non-existing"
	nodeSelectorValue := "true"
	patch := []byte(fmt.Sprintf(`{"spec":{"template":{"spec":{"nodeSelector":{"%s":"%s"}}}}}`, nodeSelectorKey, nodeSelectorValue))

	if err := r.CheckDaemonsetAvailability(ctx, namespace, name); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("%s", err))
	}

	if _, err := k8sClient.PatchDaemonSet(ctx, namespace, name, ktypes.StrategicMergePatchType, patch, metav1.PatchOptions{FieldManager: "Terraform"}); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("%s", err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumKubeProxyDisabledResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CiliumKubeProxyDisabledResourceModel
	k8sClient := r.client

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	name := data.Name.ValueString()
	nodeSelectorKey := "non-existing"
	patch := []byte(fmt.Sprintf(`[{"op":"remove","path":"/spec/template/spec/nodeSelector/%s"}]`, nodeSelectorKey))

	if err := r.CheckDaemonsetAvailability(ctx, namespace, name); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("%s", err))
	}

	if _, err := k8sClient.PatchDaemonSet(ctx, namespace, name, ktypes.JSONPatchType, patch, metav1.PatchOptions{FieldManager: "Terraform"}); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("%s", err))
		return
	}
}

func (r *CiliumKubeProxyDisabledResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
