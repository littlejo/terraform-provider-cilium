TEST?=$$(go list ./... | grep -v 'vendor')
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
HOSTNAME=terraform.local
# HOSTNAME=registry.terraform.io
NAMESPACE=local
# NAMESPACE=hashicorp
NAME=cilium
BINARY=terraform-provider-${NAME}
VERSION=0.0.1
OS_ARCH=linux_amd64

default: install

build:
	go build -o ${BINARY}

build-linux:
	env GOOS=linux GOARCH=amd64 go build -o ${BINARY}

fmt:
	gofmt -w $(GOFMT_FILES)

doc:
	go generate

release:
	goreleaser release --clean --snapshot --skip=publish,sign

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

test:
	echo $(TEST) | xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

# Align github.com/cilium/cilium with a cilium-cli release (check its go.mod replace directives too).
# Since Cilium 1.18, cilium-cli is no longer shipped in Cilium release tags: like the
# upstream cilium-cli, the provider must use the Cilium commit pinned by cilium-cli.
# Usage: make update-cilium-cli CILIUM_CLI_VERSION=v0.20.3
CILIUM_CLI_VERSION?=v0.20.3
.PHONY: update-cilium-cli
update-cilium-cli:
	@CILIUM=$$(curl -sSfL https://raw.githubusercontent.com/cilium/cilium-cli/$(CILIUM_CLI_VERSION)/go.mod | awk '$$1 == "require" && $$2 == "github.com/cilium/cilium" {print $$3}'); \
	test -n "$$CILIUM" || { echo "unable to find github.com/cilium/cilium in cilium-cli $(CILIUM_CLI_VERSION) go.mod"; exit 1; }; \
	echo "cilium-cli $(CILIUM_CLI_VERSION) uses github.com/cilium/cilium $$CILIUM"; \
	go get github.com/cilium/cilium@$$CILIUM && go mod tidy
