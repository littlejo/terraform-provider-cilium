provider "cilium" {
  namespace = "cilium"
}

provider "cilium" {
  alias = "preflight"
  namespace = "cilium"
  helm_release = "cilium-preflight"
}

resource "cilium" "this" {
  version  = "1.16.0"
  set = [
    "hubble.relay.enabled=true",
    "hubble.ui.enabled=true",
  ]
}

resource "cilium" "preflight" {
  version  = "1.16.1"
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
