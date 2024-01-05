// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/cilium/cilium-cli/config"
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
var _ resource.Resource = &CiliumConfigResource{}
var _ resource.ResourceWithImportState = &CiliumConfigResource{}

func NewCiliumConfigResource() resource.Resource {
	return &CiliumConfigResource{}
}

// CiliumConfigResource defines the resource implementation.
type CiliumConfigResource struct {
	client *k8s.Client
}

// CiliumConfigResourceModel describes the resource data model.
type CiliumConfigResourceModel struct {
	Namespace types.String `tfsdk:"namespace"`
	Restart   types.Bool   `tfsdk:"restart"`
	Key       types.String `tfsdk:"key"`
	Value     types.String `tfsdk:"value"`
	Id        types.String `tfsdk:"id"`
}

func (r *CiliumConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config"
}

func (r *CiliumConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Config resource for Cilium. This is equivalent to cilium cli: `cilium config`: It manages the cilium Kubernetes ConfigMap resource",

		Attributes: map[string]schema.Attribute{
			"namespace": schema.StringAttribute{
				MarkdownDescription: ConcatDefault("Namespace in which to install", "kube-system"),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("kube-system"),
			},
			"restart": schema.BoolAttribute{
				MarkdownDescription: ConcatDefault("Restart Cilium pods", "true"),
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "Key of the config",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Value of the key",
				Required:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cilium config identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *CiliumConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CiliumConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CiliumConfigResourceModel
	k8sClient := r.client
	var params = config.Parameters{
		Writer: os.Stdout,
	}

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	namespace := data.Namespace.ValueString()
	key := data.Key.ValueString()
	value := data.Value.ValueString()
	params.Namespace = namespace
	params.Restart = data.Restart.ValueBool()

	check := config.NewK8sConfig(k8sClient, params)
	if err := check.Set(context.Background(), key, value, params); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set config: %s", err))
		return
	}

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	data.Id = types.StringValue("cilium-config-" + key)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CiliumConfigResourceModel
	k8sClient := r.client
	var params = config.Parameters{
		Writer: os.Stdout,
	}

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	namespace := data.Namespace.ValueString()
	params.Namespace = namespace
	key := data.Key.ValueString()
	value := data.Value.ValueString()

	readReq := CiliumConfigResourceModel{
		Id:        data.Id,
		Value:     data.Value,
		Key:       data.Key,
		Restart:   data.Restart,
		Namespace: data.Namespace,
	}

	_, err := json.Marshal(readReq)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Refresh Resource",
			"An unexpected error occurred while creating the resource read request. "+
				"Please report this issue to the provider developers.\n\n"+
				"JSON Error: "+err.Error(),
		)

		return
	}

	check := config.NewK8sConfig(k8sClient, params)
	out, err := check.View(context.Background())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to view config: %s", err))
		return
	}

	m, err := regexp.MatchString(key+".*"+value, out)
	if err != nil {
		fmt.Println("your regex is faulty")
		return
	}
	if m {
		fmt.Println("Ok ttttttttttttt")
	} else {
		resp.State.RemoveResource(ctx)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CiliumConfigResourceModel
	k8sClient := r.client
	var params = config.Parameters{
		Writer: os.Stdout,
	}

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	key := data.Key.ValueString()
	value := data.Value.ValueString()
	params.Namespace = namespace
	params.Restart = data.Restart.ValueBool()

	check := config.NewK8sConfig(k8sClient, params)
	if err := check.Set(context.Background(), key, value, params); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set config: %s", err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CiliumConfigResourceModel
	k8sClient := r.client
	var params = config.Parameters{
		Writer: os.Stdout,
	}

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	params.Namespace = namespace
	key := data.Key.ValueString()
	params.Restart = data.Restart.ValueBool()

	check := config.NewK8sConfig(k8sClient, params)
	if err := check.Delete(context.Background(), key, params); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete config: %s", err))
		return
	}
}

func (r *CiliumConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
