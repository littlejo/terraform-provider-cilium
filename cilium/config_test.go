// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCiliumConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCiliumConfigResourceConfig("debug", "true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cilium_config.test", "key", "debug"),
					resource.TestCheckResourceAttr("cilium_config.test", "value", "true"),
					resource.TestCheckResourceAttr("cilium_config.test", "restart", "true"),
					resource.TestCheckResourceAttr("cilium_config.test", "id", "cilium-config-debug"),
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

func testAccCiliumConfigResourceConfig(key string, value string) string {
	return fmt.Sprintf(`
resource "cilium" "test" {
  version = "1.16.1"
}
resource "cilium_config" "test" {
  key        = %[1]q
  value      = %[2]q
  depends_on = [ cilium.test ]
}
`, key, value)
}
