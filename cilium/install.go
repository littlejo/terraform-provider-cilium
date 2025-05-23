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

	"helm.sh/helm/v3/pkg/cli/values"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
				MarkdownDescription: ConcatDefault("When upgrading, reset the values to the ones built into the chart, apply the last release's values and merge in any overrides from the command line via --set and -f. If '--reset-values' or '--reuse-values' is specified, this is ignored", "true"),
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

func (r *CiliumInstallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CiliumInstallResourceModel
	c := r.client
	if c == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to connect to kubernetes")
		return
	}
	k8sClient, namespace, helm_release := c.client, c.namespace, c.helm_release
	var params = install.Parameters{Writer: os.Stdout}
	var options values.Options

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	params.Namespace = namespace
	params.Version = data.Version.ValueString()
	params.HelmReleaseName = helm_release
	wait := data.Wait.ValueBool()

	options.Values = ValueList(ctx, data.HelmSet)

	values := data.Values.ValueString()

	if values != "" {
		f, err := os.CreateTemp("", ".values.*.yaml")
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Cilium installer: %s", err))
		}
		defer os.Remove(f.Name())

		if _, err := f.Write([]byte(values)); err != nil {
			f.Close()
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Cilium installer: %s", err))
		}
		if err := f.Close(); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Cilium installer: %s", err))
		}
		options.ValueFiles = []string{f.Name()}
	}

	params.HelmOpts = options

	installer, err := install.NewK8sInstaller(k8sClient, params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Cilium installer: %s", err))
		return
	}

	if err := installer.InstallWithHelm(context.Background(), k8sClient); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to install Cilium: %s", err))
		return
	}

	if wait {
		if err := c.Wait(); err != nil {
			return
		}
	}
	data.Id = types.StringValue(helm_release)
	helm_values, err := c.GetHelmValues()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to install Cilium: %s", err))
		return
	}
	ca, err := c.GetCA(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve cilium-ca: %s", err))
		return
	}
	data.CA = types.ObjectValueMust(CaAttributeTypes, ca)
	data.HelmValues = types.StringValue(helm_values)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumInstallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CiliumInstallResourceModel
	c := r.client
	if c == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to connect to kubernetes")
		return
	}

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := c.GetCurrentRelease()
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}
	helm_values, err := c.GetHelmValues()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tfstate: %s", err))
		return
	}
	version, err := c.GetMetadata()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tfstate: %s", err))
		return
	}
	ca, err := c.GetCA(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve cilium-ca: %s", err))
		return
	}
	data.CA = types.ObjectValueMust(CaAttributeTypes, ca)
	data.HelmValues = types.StringValue(helm_values)
	data.Version = types.StringValue(version)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumInstallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CiliumInstallResourceModel
	c := r.client
	if c == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to connect to kubernetes")
		return
	}
	k8sClient, namespace, helm_release := c.client, c.namespace, c.helm_release
	var params = install.Parameters{Writer: os.Stdout}
	var options values.Options

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params.Namespace = namespace
	params.Version = data.Version.ValueString()
	params.HelmReleaseName = helm_release
	params.HelmResetValues = data.Reset.ValueBool()
	params.HelmReuseValues = data.Reuse.ValueBool()
	params.HelmResetThenReuseValues = data.ResetThenReuse.ValueBool()
	wait := data.Wait.ValueBool()

	options.Values = ValueList(ctx, data.HelmSet)

	values := data.Values.ValueString()

	if values != "" {
		f, err := os.CreateTemp("", ".values.*.yaml")
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Cilium installer: %s", err))
		}
		defer os.Remove(f.Name())

		if _, err := f.Write([]byte(values)); err != nil {
			f.Close()
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Cilium installer: %s", err))
		}
		if err := f.Close(); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Cilium installer: %s", err))
		}
		options.ValueFiles = []string{f.Name()}
	}
	params.HelmOpts = options

	installer, err := install.NewK8sInstaller(k8sClient, params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to upgrade Cilium: %s", err))
		return
	}
	if err := installer.UpgradeWithHelm(context.Background(), k8sClient); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to upgrade Cilium: %s", err))
		return
	}
	if wait {
		if err := c.Wait(); err != nil {
			return
		}
	}

	helm_values, err := c.GetHelmValues()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to upgrade Cilium: %s", err))
		return
	}
	ca, err := c.GetCA(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve cilium-ca: %s", err))
		return
	}
	data.CA = types.ObjectValueMust(CaAttributeTypes, ca)
	data.HelmValues = types.StringValue(helm_values)
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumInstallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CiliumInstallResourceModel
	c := r.client
	if c == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to connect to kubernetes")
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
	ctxb := context.Background()

	uninstaller := install.NewK8sUninstaller(k8sClient, params)
	uninstaller.DeleteTestNamespace(ctxb)

	if params.Wait {
		fmt.Printf("⌛ Waiting to disable Hubble before uninstalling Cilium\n")
		for {
			// Wait for the test namespace to be terminated. Subsequent connectivity checks would fail
			// if the test namespace is in Terminating state.
			_, err := k8sClient.GetNamespace(ctx, params.TestNamespace, metav1.GetOptions{})
			if err == nil {
				time.Sleep(defaults.WaitRetryInterval)
			} else {
				break
			}
		}
	}
	if err := uninstaller.UninstallWithHelm(ctxb, k8sClient.HelmActionConfig); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("⚠ ️ Unable to uninstall Cilium: %s", err))
		return
	}
}

func (r *CiliumInstallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
