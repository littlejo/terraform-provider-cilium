// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCiliumInstallResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCiliumInstallResourceConfig("1.15.7"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cilium.test", "version", "1.15.7"),
					resource.TestCheckResourceAttr("cilium.test", "id", "cilium"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "cilium.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"data_path", "repository", "reset", "reuse", "values", "wait"},
			},
			// Update and Read testing
			{
				Config: testAccCiliumInstallResourceConfig("1.15.8"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cilium.test", "version", "1.15.8"),
					resource.TestCheckResourceAttr("cilium.test", "id", "cilium"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccCiliumInstallResourceConfig(version string) string {
	return fmt.Sprintf(`
resource "cilium" "test" {
  version = %[1]q
}
`, version)
}
