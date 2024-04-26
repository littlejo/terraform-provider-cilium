Terraform Cilium Provider (Experimental)
==================

The Cilium Provider allows Terraform to manage [Cilium](https://cilium.io/) resources.

- Website: [registry.terraform.io](https://registry.terraform.io/providers/littlejo/cilium/latest/docs)

Requirements
------------

- [Terraform](https://www.terraform.io/downloads.html) > 1.5 or [OpenTofu](https://opentofu.org/docs/intro/install/) > 1.6
- [Go](https://golang.org/doc/install) 1.22 (to build the provider plugin)

Building The Provider
---------------------

```sh
$ git clone https://github.com/littlejo/terraform-provider-cilium.git
$ cd terraform-provider-cilium
```

Enter the provider directory and build the provider

```sh
$ make build
```

Using the provider
----------------------

Please see the documentation in the [Terraform registry](https://registry.terraform.io/providers/littlejo/cilium/latest/docs).

Or you can browse the documentation within this repo [here](https://github.com/littlejo/terraform-provider-cilium/tree/main/docs).

Using the locally built provider
----------------------

If you wish to test the provider from the local version you just built, you can try the following method.

First install the Terraform Provider binary into your local plugin repository:

```sh
# Set your target environment (OS_architecture): linux_amd64, darwin_amd64...
$ vim GNUmakefile
$ make install
```

Then create a Terraform configuration using this exact provider:

```sh
$ mkdir ~/test-terraform-provider-cilium
$ cd ~/test-terraform-provider-cilium
$ cat > main.tf <<EOF
# Configure the cilium Provider
terraform {
  required_providers {
    cilium = {
      source = "terraform.local/local/cilium"
      version = "0.0.1"
    }
  }
}

provider "cilium" {
}
EOF

# Initialize your project and remove existing dependencies lock file
$ rm .terraform.lock.hcl && terraform init
...

# Apply your resources & datasources
$ terraform apply
...
```


Developing the Provider
---------------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.22+ is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

```sh
$ make build
```

Testing the Provider
--------------------

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

In order to run the full suite of Acceptance tests you will need to create a kubernetes cluster like kind. You can also test on some examples:
* Examples of some cloud providers (AWS, Azure and GCP): https://github.com/orgs/tf-cilium/repositories
* Examples from the CICD: https://github.com/littlejo/terraform-provider-cilium/tree/main/.github/tf

# Contributing

Please read the [contributing guide](./CONTRIBUTING.md) to learn about how you can contribute to the Cilium Terraform provider.<br/>
There is no small contribution, don't hesitate!

Our awesome contributors:

<a href="https://github.com/littlejo/terraform-provider-cilium/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=littlejo/terraform-provider-cilium" />
</a>
