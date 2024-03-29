// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCiliumHubbleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCiliumHubbleResourceConfig("true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cilium_hubble.test", "ui", "true"),
					resource.TestCheckResourceAttr("cilium_hubble.test", "relay", "true"),
					resource.TestCheckResourceAttr("cilium_hubble.test", "id", "cilium-hubble"),
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

func testAccCiliumHubbleResourceConfig(ui string) string {
	return fmt.Sprintf(`
resource "cilium" "test" {
  version = "1.15.2"
}
resource "cilium_hubble" "test" {
  ui         = %s
  depends_on = [ cilium.test ]
}
`, ui)
}
