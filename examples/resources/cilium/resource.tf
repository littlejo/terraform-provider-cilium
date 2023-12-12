resource "cilium" "example" {
  version = "1.14.4"

  helm_set = [
    "cluster.id=1",
    "ipam.operator.clusterPoolIPv4PodCIDRList=10.10.0.0/16",
    "cluster.name=cilium-clustermesh-1",
    "clustermesh.maxConnectedClusters=511",
  ]
}
