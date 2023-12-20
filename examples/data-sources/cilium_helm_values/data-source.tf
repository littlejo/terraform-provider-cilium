data "cilium_helm_values" "example" {}

resource "local_file" "example" {
  content  = data.cilium_helm_values.example.yaml
  filename = "${path.module}/values.yaml"
}
