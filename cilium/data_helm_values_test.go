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
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cilium_helm_values.test", "yaml", "cluster:\n    name: kind-chart-testing\nipam:\n    mode: kubernetes\noperator:\n    replicas: 1\nroutingMode: tunnel\ntunnelProtocol: vxlan\n"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccCiliumHelmValuesDataSourceConfig() string {
	return `
resource "cilium" "test" {
  version = "1.16.1"
}

data "cilium_helm_values" "test" {

  depends_on = [
	  cilium.test
  ]
}
`
}
