// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCiliumHelmValuesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCiliumHelmValuesDataSourceConfig(),
				Check:  resource.ComposeAggregateTestCheckFunc(
				//resource.TestCheckResourceAttr("cilium.test", "version", "1.14.4"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccCiliumHelmValuesDataSourceConfig(version string) string {
	return `
resource "cilium" "test" {
  version = "1.15.2"
}

data "cilium_helm_values" "test" {
}
`
}
