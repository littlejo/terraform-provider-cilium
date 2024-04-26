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
  namespace = "cilium"
}

provider "cilium" {
  alias = "preflight"
  namespace = "cilium"
  helm_release = "cilium-preflight"
}

resource "cilium" "this" {
  version  = "1.15.2"
  set = [
    "hubble.relay.enabled=true",
    "hubble.ui.enabled=true",
  ]
}

resource "cilium" "preflight" {
  version  = "1.15.3"
  set = [
    "preflight.enabled=true",
    "agent=false",
    "operator.enabled=false",
  ]
  provider = cilium.preflight
}

output "cilium_ca" {
  value = nonsensitive(cilium.this.ca)
}
