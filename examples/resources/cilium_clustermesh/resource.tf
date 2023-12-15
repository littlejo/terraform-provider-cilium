resource "cilium" "example" {
  helm_set = [
    "cluster.name=clustermesh1",
    "cluster.id=1",
    "ipam.mode=kubernetes",
  ]
  version = "1.14.4"
}

resource "cilium_clustermesh" "example" {
  service_type = "LoadBalancer"
  depends_on = [
    cilium.example
  ]
}

# Complete example: https://github.com/littlejo/terraform-kind-cilium-clustermesh
