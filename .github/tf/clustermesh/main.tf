locals {
  cert = cilium.this.ca["crt"]
  key  = cilium.this.ca["key"]
}

provider "cilium" {
  alias   = "mesh1"
  context = "kind-test1"
}

provider "cilium" {
  alias   = "mesh2"
  context = "kind-test2"
}

provider "cilium" {
  alias   = "mesh3"
  context = "kind-test3"
}

provider "cilium" {
  alias   = "mesh4"
  context = "kind-test4"
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
    "tls.ca.cert=${local.cert}",
    "tls.ca.key=${local.key}",
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

resource "cilium" "this3" {
  set = [
    "cluster.name=mesh3",
    "cluster.id=3",
    "ipam.mode=kubernetes",
    "tls.ca.cert=${local.cert}",
    "tls.ca.key=${local.key}",
  ]
  version  = "1.15.2"
  provider = cilium.mesh3
}

resource "cilium_clustermesh" "this3" {
  service_type = "NodePort"
  depends_on = [
    cilium.this3
  ]
  provider = cilium.mesh3
}

resource "cilium" "this4" {
  set = [
    "cluster.name=mesh4",
    "cluster.id=4",
    "ipam.mode=kubernetes",
    "tls.ca.cert=${local.cert}",
    "tls.ca.key=${local.key}",
  ]
  version  = "1.15.2"
  provider = cilium.mesh4
}

resource "cilium_clustermesh" "this4" {
  service_type = "NodePort"
  depends_on = [
    cilium.this4
  ]
  provider = cilium.mesh4
}

#resource "cilium_clustermesh_connection" "this" {
#  destination_context = "kind-test2"
#  provider            = cilium.mesh1
#  depends_on = [
#    cilium_clustermesh.this,
#    cilium_clustermesh.this2,
#  ]
#}
