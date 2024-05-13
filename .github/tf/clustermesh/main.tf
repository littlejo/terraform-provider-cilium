locals {
  cert = cilium.this.ca["crt"]
  key  = cilium.this.ca["key"]
}

variable "config_content" {
}

provider "cilium" {
  alias   = "mesh1"
  context = "kind-test1"
  config_content = var.config_content
}

provider "cilium" {
  alias   = "mesh2"
  context = "kind-test2"
  config_content = var.config_content
}

resource "cilium" "this" {
  set = [
    "cluster.name=mesh1",
    "cluster.id=1",
    "ipam.mode=kubernetes",
  ]
  version  = "1.15.2"
  provider = cilium.mesh1
}

output "cilium_ca1" {
  value = nonsensitive(cilium.this.ca)
}

resource "cilium_clustermesh" "this" {
  service_type = "NodePort"
  depends_on = [
    cilium.this
  ]
  provider = cilium.mesh1
}

resource "cilium" "this2" {
  set = [
    "cluster.name=mesh2",
    "cluster.id=2",
    "ipam.mode=kubernetes",
    "tls.ca.cert=${local.cert}",
    "tls.ca.key=${local.key}",
  ]
  version  = "1.15.2"
  provider = cilium.mesh2
}

output "cilium_ca2" {
  value = nonsensitive(cilium.this2.ca)
}

resource "cilium_clustermesh" "this2" {
  service_type = "NodePort"
  depends_on = [
    cilium.this2
  ]
  provider = cilium.mesh2
}

resource "cilium_clustermesh_connection" "this" {
  destination_context = "kind-test2"
  provider            = cilium.mesh1
  depends_on = [
    cilium_clustermesh.this,
    cilium_clustermesh.this2,
  ]
}
