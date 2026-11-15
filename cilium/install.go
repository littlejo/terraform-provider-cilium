// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cilium/cilium/cilium-cli/defaults"
	"github.com/cilium/cilium/cilium-cli/install"

	"helm.sh/helm/v4/pkg/cli/values"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
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

// CiliumInstallResource defines the resource implementation.
type CiliumInstallResource struct {
	client *CiliumClient
}

// CiliumInstallResourceModel describes the resource data model.
type CiliumInstallResourceModel struct {
	HelmSet        types.List   `tfsdk:"set"`
	Values         types.String `tfsdk:"values"`
	Version        types.String `tfsdk:"version"`
	Repository     types.String `tfsdk:"repository"`
	DataPath       types.String `tfsdk:"data_path"`
	Wait           types.Bool   `tfsdk:"wait"`
	Reuse          types.Bool   `tfsdk:"reuse"`
	Reset          types.Bool   `tfsdk:"reset"`
	ResetThenReuse types.Bool   `tfsdk:"reusethenreuse"`
	Id             types.String `tfsdk:"id"`
	HelmValues     types.String `tfsdk:"helm_values"`
	CA             types.Object `tfsdk:"ca"`
}

func (r *CiliumInstallResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName
}

func (r *CiliumInstallResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resetThenReuseDescription := "When upgrading, reset the values to the ones built into the chart, apply the last release's values and merge in any overrides from the command line via --set and -f. If '--reset-values' or '--reuse-values' is specified, this is ignored"
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Install resource for Cilium. This is equivalent to cilium cli: `cilium install`, `cilium upgrade` and `cilium uninstall`: It manages cilium helm chart",

		Attributes: map[string]schema.Attribute{
			"ca": schema.ObjectAttribute{
				AttributeTypes: CaAttributeTypes,
				Computed:       true,
				Sensitive:      true,
			},
			"set": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: ConcatDefault("Set helm values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2", "[]"),
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListNull(types.StringType)),
			},
			"values": schema.StringAttribute{
				MarkdownDescription: ConcatDefault("values in raw yaml to pass to helm.", "empty"),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"version": schema.StringAttribute{
				MarkdownDescription: ConcatDefault("Version of Cilium", "1.17.3"),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("1.17.3"),
			},
			"repository": schema.StringAttribute{
				MarkdownDescription: ConcatDefault("Helm chart repository to download Cilium charts from", defaults.HelmRepository),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(defaults.HelmRepository),
			},
			"data_path": schema.StringAttribute{
				MarkdownDescription: ConcatDefault("Datapath mode to use { tunnel | native | aws-eni | gke | azure | aks-byocni }", "autodetected"),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"reuse": schema.BoolAttribute{
				MarkdownDescription: ConcatDefault("When upgrading, reuse the helm values from the latest release unless any overrides from are set from other flags. This option takes precedence over HelmResetValues", "false"),
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"reset": schema.BoolAttribute{
				MarkdownDescription: ConcatDefault("When upgrading, reset the helm values to the ones built into the chart", "false"),
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"reusethenreuse": schema.BoolAttribute{
				MarkdownDescription: ConcatDefault(resetThenReuseDescription, "true"),
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"wait": schema.BoolAttribute{
				MarkdownDescription: ConcatDefault("Wait for Cilium status is ok", "true"),
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cilium install identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"helm_values": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Helm values (`helm get values -n kube-system cilium`)",
			},
		},
	}
}

func (r *CiliumInstallResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// installParameters builds the cilium-cli install parameters from the resource model.
// The returned cleanup function removes the temporary values file, if any.
func (c *CiliumClient) installParameters(ctx context.Context, data *CiliumInstallResourceModel) (install.Parameters, func(), diag.Diagnostics) {
	var diags diag.Diagnostics
	var options values.Options
	cleanup := func() {}

	params := install.Parameters{Writer: os.Stdout}
	params.Namespace = c.namespace
	params.HelmReleaseName = c.helm_release
	params.Version = data.Version.ValueString()
	params.HelmRepository = data.Repository.ValueString()
	params.DatapathMode = data.DataPath.ValueString()
	params.HelmResetValues = data.Reset.ValueBool()
	params.HelmReuseValues = data.Reuse.ValueBool()
	params.HelmResetThenReuseValues = data.ResetThenReuse.ValueBool()

	options.Values = ValueList(ctx, data.HelmSet)

	if v := data.Values.ValueString(); v != "" {
		f, err := writeTempFile(".values.*.yaml", []byte(v))
		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to write Helm values to a temporary file: %s", err))
			return params, cleanup, diags
		}
		cleanup = func() { os.Remove(f) }
		options.ValueFiles = []string{f}
	}
	params.HelmOpts = options

	return params, cleanup, diags
}

// refresh updates the computed attributes of the model from the cluster.
func (c *CiliumClient) refresh(ctx context.Context, data *CiliumInstallResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	helm_values, err := c.GetHelmValues()
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read Cilium Helm values: %s", err))
		return diags
	}
	ca, err := c.GetCA(ctx)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to retrieve cilium-ca: %s", err))
		return diags
	}
	data.CA = types.ObjectValueMust(CaAttributeTypes, ca)
	data.HelmValues = types.StringValue(helm_values)
	return diags
}

func (r *CiliumInstallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CiliumInstallResourceModel
	c := r.client
	if err := c.Ready(); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	k8sClient := c.client

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params, cleanup, diags := c.installParameters(ctx, &data)
	defer cleanup()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	installer, err := install.NewK8sInstaller(k8sClient, params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Cilium installer: %s", err))
		return
	}

	if err := installer.InstallWithHelm(ctx, k8sClient); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to install Cilium: %s", err))
		return
	}

	if data.Wait.ValueBool() {
		if err := c.Wait(ctx); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Cilium was installed but did not become ready: %s", err))
			return
		}
	}
	data.Id = types.StringValue(c.helm_release)

	resp.Diagnostics.Append(c.refresh(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumInstallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CiliumInstallResourceModel
	c := r.client
	if err := c.Ready(); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := c.ReleaseExists(); err != nil {
		resp.State.RemoveResource(ctx)
		return
	}
	version, err := c.GetMetadata()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Cilium Helm metadata: %s", err))
		return
	}
	data.Version = types.StringValue(version)

	resp.Diagnostics.Append(c.refresh(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumInstallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CiliumInstallResourceModel
	c := r.client
	if err := c.Ready(); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	k8sClient := c.client

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params, cleanup, diags := c.installParameters(ctx, &data)
	defer cleanup()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	installer, err := install.NewK8sInstaller(k8sClient, params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to upgrade Cilium: %s", err))
		return
	}
	if err := installer.UpgradeWithHelm(ctx, k8sClient); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to upgrade Cilium: %s", err))
		return
	}
	if data.Wait.ValueBool() {
		if err := c.Wait(ctx); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Cilium was upgraded but did not become ready: %s", err))
			return
		}
	}

	resp.Diagnostics.Append(c.refresh(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumInstallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CiliumInstallResourceModel
	c := r.client
	if err := c.Ready(); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	k8sClient, namespace, helm_release := c.client, c.namespace, c.helm_release
	var params = install.UninstallParameters{Writer: os.Stdout}

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params.Namespace = namespace
	params.HelmReleaseName = helm_release
	params.TestNamespace = defaults.ConnectivityCheckNamespace
	params.Wait = data.Wait.ValueBool()

	params.Timeout = defaults.UninstallTimeout

	uninstaller := install.NewK8sUninstaller(k8sClient, params)
	uninstaller.DeleteTestNamespace(ctx)

	if params.Wait {
		tflog.Info(ctx, "Waiting for the connectivity test namespace to be deleted before uninstalling Cilium")
		for {
			// Wait for the test namespace to be terminated. Subsequent connectivity checks would fail
			// if the test namespace is in Terminating state.
			_, err := k8sClient.GetNamespace(ctx, params.TestNamespace, metav1.GetOptions{})
			if err != nil {
				break
			}
			select {
			case <-ctx.Done():
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Timed out waiting for namespace %s to be deleted: %s", params.TestNamespace, ctx.Err()))
				return
			case <-time.After(defaults.WaitRetryInterval):
			}
		}
	}
	if err := uninstaller.UninstallWithHelm(ctx, k8sClient.HelmActionConfig); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to uninstall Cilium: %s", err))
		return
	}
}

func (r *CiliumInstallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
