resource "cilium_clustermesh_connection" "example" {
  destination_contexts = ["context-2"]
}

provider "cilium" {
  config_path = "${path.module}/kubeconfig"
  context     = "context-1"
}
