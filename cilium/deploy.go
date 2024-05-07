// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/cilium/cilium-cli/defaults"
	"github.com/cilium/cilium-cli/hubble"
	"github.com/cilium/cilium-cli/install"
	"github.com/cilium/cilium/pkg/inctimer"

	"helm.sh/helm/v3/pkg/cli/values"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
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
var _ resource.Resource = &CiliumDeployResource{}
var _ resource.ResourceWithImportState = &CiliumDeployResource{}

func NewCiliumDeployResource() resource.Resource {
	return &CiliumDeployResource{}
}

// CiliumDeployResource defines the resource implementation.
type CiliumDeployResource struct {
	client *CiliumClient
}

// CiliumDeployResourceModel describes the resource data model.
type CiliumDeployResourceModel struct {
	HelmSet    types.List   `tfsdk:"set"`
	Values     types.String `tfsdk:"values"`
	Version    types.String `tfsdk:"version"`
	Repository types.String `tfsdk:"repository"`
	DataPath   types.String `tfsdk:"data_path"`
	Wait       types.Bool   `tfsdk:"wait"`
	Reuse      types.Bool   `tfsdk:"reuse"`
	Reset      types.Bool   `tfsdk:"reset"`
	Id         types.String `tfsdk:"id"`
	HelmValues types.String `tfsdk:"helm_values"`
	CA         types.Object `tfsdk:"ca"`
}

func (r *CiliumDeployResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deploy"
}

func (r *CiliumDeployResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				MarkdownDescription: ConcatDefault("Version of Cilium", defaults.Version),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(defaults.Version),
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
				MarkdownDescription: ConcatDefault("When upgrading, reuse the helm values from the latest release unless any overrides from are set from other flags. This option takes precedence over HelmResetValues", "true"),
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"reset": schema.BoolAttribute{
				MarkdownDescription: ConcatDefault("When upgrading, reset the helm values to the ones built into the chart", "false"),
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
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

func (r *CiliumDeployResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CiliumDeployResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CiliumDeployResourceModel
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

	helmSet := make([]types.String, 0, len(data.HelmSet.Elements()))
	data.HelmSet.ElementsAs(ctx, &helmSet, false)

	h := []string{}
	for _, e := range helmSet {
		h = append(h, e.ValueString())
	}

	options.Values = h

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

func (r *CiliumDeployResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CiliumDeployResourceModel
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

func (r *CiliumDeployResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CiliumDeployResourceModel
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
	wait := data.Wait.ValueBool()
	helmSet := make([]types.String, 0, len(data.HelmSet.Elements()))
	data.HelmSet.ElementsAs(ctx, &helmSet, false)

	h := []string{}
	for _, e := range helmSet {
		h = append(h, e.ValueString())
	}

	options.Values = h

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

func (r *CiliumDeployResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CiliumDeployResourceModel
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

	var hubbleParams = hubble.Parameters{
		Writer:          os.Stdout,
		Wait:            true,
		Namespace:       namespace,
		HelmReleaseName: helm_release,
	}

	if params.Wait {
		// Disable Hubble, then wait for Pods to terminate before uninstalling Cilium.
		// This guarantees that relay Pods are terminated fully via Cilium (rather than
		// being queued for deletion) before uninstalling Cilium.
		fmt.Printf("⌛ Waiting to disable Hubble before uninstalling Cilium\n")
		if err := hubble.DisableWithHelm(ctx, k8sClient, hubbleParams); err != nil {
			fmt.Printf("⚠ ️ Failed to disable Hubble prior to uninstalling Cilium: %s\n", err)
		}
		for {
			ps, err := k8sClient.ListPods(ctx, hubbleParams.Namespace, metav1.ListOptions{
				LabelSelector: "k8s-app=hubble-relay",
			})
			if err != nil {
				if k8sErrors.IsNotFound(err) {
					break
				}
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list pods waiting for hubble-relay to stop: %s", err))
			}
			if len(ps.Items) == 0 {
				break
			}
			select {
			case <-inctimer.After(defaults.WaitRetryInterval):
			case <-ctx.Done():
			}
		}
	}
	if err := uninstaller.UninstallWithHelm(ctxb, k8sClient.HelmActionConfig); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("⚠ ️ Unable to uninstall Cilium: %s", err))
		return
	}
}

func (r *CiliumDeployResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
