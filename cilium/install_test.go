// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccCiliumInstallResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCiliumInstallResourceConfig("1.20.3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cilium.test", "version", "1.20.3"),
					resource.TestCheckResourceAttr("cilium.test", "id", "cilium"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "cilium.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"data_path", "repository", "reset", "reuse", "reset_then_reuse", "values", "wait"},
			},
			// Update and Read testing
			{
				Config: testAccCiliumInstallResourceConfig("1.20.4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cilium.test", "version", "1.20.4"),
					resource.TestCheckResourceAttr("cilium.test", "id", "cilium"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// TestAccCiliumInstallResource_upgradeFromV0 checks that a state created by the
// last 0.3.x release is migrated without any planned change.
func TestAccCiliumInstallResource_upgradeFromV0(t *testing.T) {
	// The released provider and the provider under test must share the same
	// address so that the state written by the former is read by the latter.
	t.Setenv("TF_ACC_PROVIDER_NAMESPACE", "littlejo")
	config := testAccCiliumInstallResourceConfig("1.20.4")
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"cilium": {
						Source:            "littlejo/cilium",
						VersionConstraint: "0.3.2",
					},
				},
				Config: config,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cilium.test", "reset_then_reuse", "true"),
					resource.TestCheckNoResourceAttr("cilium.test", "reusethenreuse"),
				),
			},
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
