terraform {
  required_providers {
    cilium = {
      source = "littlejo/cilium"
      version = ">= 0.2.0"
    }
  }
  required_version = ">= 1.3"
}
