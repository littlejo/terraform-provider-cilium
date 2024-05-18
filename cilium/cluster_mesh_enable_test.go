// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCiliumClusterMeshEnableResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCiliumClusterMeshEnableResourceConfig("NodePort"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cilium_clustermesh.test", "enable_external_workloads", "false"),
					resource.TestCheckResourceAttr("cilium_clustermesh.test", "enable_kv_store_mesh", "false"),
					resource.TestCheckResourceAttr("cilium_clustermesh.test", "service_type", "NodePort"),
					resource.TestCheckResourceAttr("cilium_clustermesh.test", "id", "ciliumclustermeshenable"),
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

func testAccCiliumClusterMeshEnableResourceConfig(service_type string) string {
	return fmt.Sprintf(ProviderConfig+`
resource "cilium" "test" {
  version = "1.15.2"
}
resource "cilium_clustermesh" "test" {
  service_type = %[1]q
  depends_on   = [ cilium.test ]
}
resource "cilium" "test2" {
  version = "1.15.2"
  provider = cilium.test
}
resource "cilium_clustermesh" "test2" {
  service_type = %[1]q
  depends_on   = [ cilium.test2 ]
  provider = cilium.test
}
`, service_type)
}
