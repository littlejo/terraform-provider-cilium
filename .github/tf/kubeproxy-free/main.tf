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
}

resource "cilium_kubeproxy_free" "this" {
}
