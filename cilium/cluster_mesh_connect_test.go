// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCiliumClusterMeshConnectResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCiliumClusterMeshConnectResourceConfig(),
				Check:  resource.ComposeAggregateTestCheckFunc(
				//resource.TestCheckResourceAttr("cilium_clustermesh.test", "enable_external_workloads", "false"),
				//resource.TestCheckResourceAttr("cilium_clustermesh.test", "enable_kv_store_mesh", "false"),
				//resource.TestCheckResourceAttr("cilium_clustermesh.test", "service_type", "NodePort"),
				//resource.TestCheckResourceAttr("cilium_clustermesh.test", "id", "ciliumclustermeshenable"),
				),
			},
			// ImportState testing
			//{
			//	ResourceName:            "cilium_clustermesh.test",
			//	ImportState:             true,
			//	ImportStateVerify:       true,
			//	ImportStateVerifyIgnore: []string{"enable_external_workloads", "enable_kv_store_mesh", "service_type"},
			//},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccCiliumClusterMeshConnectResourceConfig() string {
	return `
resource "cilium" "test" {
  version = "1.15.2"
  set = [
    "cluster.name=test1",
    "cluster.id=1",
    "ipam.mode=kubernetes",
  ]
}
resource "cilium_clustermesh" "test" {
  service_type = "NodePort"
  depends_on   = [ cilium.test ]
}

resource "kind_cluster" "test2" {
  name = mesh2
  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"

    node {
      role = "control-plane"
    }

    node {
      role = "worker"
    }

    networking {
      disable_default_cni = true
      pod_subnet          = "10.245.0.0/16"
      service_subnet      = "10.80.0.0/12"
    }
  }
}

provider "cilium" {
  alias       = "test2"
  config_path = kind_cluster.test2.kubeconfig_path
}

resource "cilium" "test2" {
  version = "1.15.2"
  set = [
    "cluster.name=test2",
    "cluster.id=2",
    "ipam.mode=kubernetes",
  ]

  provider = cilium.test2
}
resource "cilium_clustermesh" "test2" {
  service_type = "NodePort"
  depends_on   = [ cilium.test2 ]
  provider = cilium.test2
}
`
}
