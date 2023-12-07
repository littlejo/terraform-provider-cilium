// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/cilium/cilium-cli/connectivity/check"
	"github.com/cilium/cilium-cli/defaults"
	"github.com/cilium/cilium-cli/install"
	"github.com/cilium/cilium-cli/k8s"
	"github.com/cilium/cilium-cli/status"
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
	AzureResourceGroupName     types.String `tfsdk:"azure_resource_group_name"`
	ClusterId                  types.String `tfsdk:"cluster_id"`
	ClusterName                types.String `tfsdk:"cluster_name"`
	ClusterPoolIpv4PodCidrList types.List   `tfsdk:"cluster_pool_ipv4_pod_cidr_list"`
	Version                    types.String `tfsdk:"version"`
	Namespace                  types.String `tfsdk:"namespace"`
	Repository                 types.String `tfsdk:"repository"`
	DataPath                   types.String `tfsdk:"data_path"`
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
			"version": schema.StringAttribute{
				MarkdownDescription: "Version of Cilium",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(defaults.Version),
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Namespace in which to install",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("kube-system"),
			},
			"repository": schema.StringAttribute{
				MarkdownDescription: "Helm repository",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(defaults.HelmRepository),
			},
			"data_path": schema.StringAttribute{
				MarkdownDescription: "Datapath mode to use { tunnel | native | aws-eni | gke | azure | aks-byocni } (default: autodetected).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
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
	k8sClient := r.client
	var params = install.Parameters{Writer: os.Stdout}
	var options values.Options
	params.APIVersions = []string{"v1"}
	params.HelmValuesSecretName = "cilium"

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	namespace := data.Namespace.ValueString()
	params.Namespace = namespace
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

	installer, err := install.NewK8sInstaller(k8sClient, params)
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
	k8sClient := r.client
	var params = status.K8sStatusParameters{}

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	params.Namespace = namespace

	collector, err := status.NewK8sStatusCollector(k8sClient, params)
	if err != nil {
		return
	}

	s, err := collector.Status(context.Background())
	if err != nil {
		// Report the most recent status even if an error occurred.
		fmt.Fprint(os.Stderr, s.Format())
		fmt.Printf("Unable to determine status: %s\n", err)
		return
	}
	if params.Output == status.OutputJSON {
		jsonStatus, err := json.MarshalIndent(s, "", " ")
		if err != nil {
			// Report the most recent status even if an error occurred.
			fmt.Fprint(os.Stderr, s.Format())
			fmt.Printf("Unable to marshal status to JSON: %s\n", err)
			return
		}
		fmt.Println(string(jsonStatus))
	} else {
		fmt.Print(s.Format())
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumInstallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CiliumInstallResourceModel
	k8sClient := r.client
	var params = install.Parameters{Writer: os.Stdout}
	var options values.Options

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	params.Namespace = namespace
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

	installer, err := install.NewK8sInstaller(k8sClient, params)
	if err != nil {
		fmt.Printf("Unable to upgrade Cilium: %s\n", err)
		return
	}
	if err := installer.UpgradeWithHelm(context.Background(), k8sClient); err != nil {
		fmt.Printf("Unable to upgrade Cilium: %s\n", err)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CiliumInstallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CiliumInstallResourceModel
	k8sClient := r.client
	var params = install.UninstallParameters{Writer: os.Stdout}

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	params.Namespace = namespace
	params.TestNamespace = defaults.ConnectivityCheckNamespace
	params.Wait = true
	ctxb := context.Background()
	version := data.Version.ValueString()

	cc, err := check.NewConnectivityTest(k8sClient, check.Parameters{
		CiliumNamespace: namespace,
		TestNamespace:   params.TestNamespace,
		FlowValidation:  check.FlowValidationModeDisabled,
		Writer:          os.Stdout,
	}, version)
	if err != nil {
		fmt.Printf("⚠ ️ Failed to initialize connectivity test uninstaller: %s\n", err)
	} else {
		cc.UninstallResources(ctxb, params.Wait)
	}
	uninstaller := install.NewK8sUninstaller(k8sClient, params)
	if err := uninstaller.UninstallWithHelm(ctxb, k8sClient.HelmActionConfig); err != nil {
		fmt.Printf("⚠ ️ Unable to uninstall Cilium: %s\n", err)
		return
	}
}

func (r *CiliumInstallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
