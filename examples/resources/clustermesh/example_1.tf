resource "cilium" "example" {
  set = [
    "cluster.name=clustermesh1",
    "cluster.id=1",
    "ipam.mode=kubernetes",
  ]
  version = "1.14.5"
}

resource "cilium_clustermesh" "example" {
  service_type = "LoadBalancer"
  depends_on = [
    cilium.example
  ]
}
