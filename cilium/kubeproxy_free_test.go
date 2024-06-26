// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCiliumKubeProxyDisabledResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCiliumKubeProxyDisabledResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cilium_kubeproxy_free.test", "name", "kube-proxy"),
					resource.TestCheckResourceAttr("cilium_kubeproxy_free.test", "namespace", "kube-system"),
					resource.TestCheckResourceAttr("cilium_kubeproxy_free.test", "id", "cilium-kubeproxy-less"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "cilium_kubeproxy_free.test",
				ImportState:       true,
				ImportStateVerify: true,
				// This is not normally necessary, but is here because this
				// example code does not have an actual upstream service.
				// Once the Read method is able to refresh information from
				// the upstream service, this can be removed.
				ImportStateVerifyIgnore: []string{"name", "namespace"},
			},
			// Update and Read testing
			//{
			//	Config: testAccCiliumKubeProxyDisabledResourceConfig("two"),
			//	Check: resource.ComposeAggregateTestCheckFunc(
			//		resource.TestCheckResourceAttr("cilium_kubeproxy_free.test", "name", "two"),
			//	),
			//},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccCiliumKubeProxyDisabledResourceConfig() string {
	return `
resource "cilium_kubeproxy_free" "test" {
}
`
}
