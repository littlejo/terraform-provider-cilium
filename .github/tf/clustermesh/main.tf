terraform {
  required_providers {
    cilium = {
      source  = "terraform.local/local/cilium"
      version = "0.0.1"
    }
  }
  required_version = ">= 1.3"
}

provider "cilium" {
  alias   = "mesh1"
  context = "kind-test1"
}

provider "cilium" {
  alias   = "mesh2"
  context = "kind-test2"
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
  ]
  version  = "1.15.2"
  provider = cilium.mesh2
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
