---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cilium Provider"
subcategory: ""
description: |-
  
---

# cilium Provider

## Example Usage

```terraform
provider "cilium" {
  config_path = "${path.module}/kubeconfig"
}
```

<!-- schema generated by tfplugindocs -->

## Schema

### Optional

- `config_path` (String) A path to a kube config file (Default: `~/.kube/config`).
- `context` (String) Context of kubeconfig file (Default: `default context`).
- `namespace` (String) Namespace to install cilium (Default: `kube-system`).
