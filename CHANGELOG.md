## 0.4.0 (Unreleased)

BREAKING CHANGES:

* Go 1.27 is required to build the provider.
* `github.com/cilium/cilium` is now pinned to the commit used by cilium-cli v0.20.3 (Cilium 1.21 development branch), and Helm v4 replaces Helm v3. Supported Cilium versions follow cilium-cli: Cilium 1.17 and newer.
* resource/cilium: the `repository` and `data_path` attributes were ignored before; they are now passed to the installer.

FEATURES:

* resource/cilium: new `reset_then_reuse` attribute. Existing states are migrated automatically from `reusethenreuse`.

ENHANCEMENTS:

* resource/cilium: new installations use Cilium 1.20.4 by default. When `version` is not set, existing installations keep their current version instead of being upgraded to the new default.
* All resources: Terraform cancellation and timeouts now propagate to Kubernetes and Helm calls.
* Documentation: the schema is generated from the provider code and no longer drifts from it.

BUG FIXES:

* resource/cilium: a Cilium that never became ready is now reported as an error instead of failing without explanation.
* resource/cilium: errors while writing the Helm values file now stop the apply.
* resource/cilium_kubeproxy_free: a missing DaemonSet now stops the apply instead of attempting the patch anyway.
* data-source/cilium_helm_values: Helm errors are now reported instead of returning empty values.
* provider: an invalid `config_content` now returns a diagnostic instead of crashing the provider, and Kubernetes client errors are reported with their cause by the resources that need the cluster.
* provider: the temporary kubeconfig file written from `config_content` is removed when the provider exits.

DEPRECATIONS:

* resource/cilium: `reusethenreuse` is deprecated in favour of `reset_then_reuse`.
