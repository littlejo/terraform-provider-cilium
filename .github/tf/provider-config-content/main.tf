variable "config_content" {
}

provider "cilium" {
  config_content = var.config_content
}

resource "cilium" "this" {
  version  = "1.15.3"
}
