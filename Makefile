# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

BUILD_DATE?=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_VERSION?=$(shell git describe --tags --dirty 2>/dev/null)
GIT_COMMIT?=$(shell git rev-parse HEAD 2>/dev/null)
GIT_BRANCH?=$(shell git symbolic-ref --short HEAD 2>/dev/null)

BIN_DIR = ${PWD}/bin
ifeq (${GIT_VERSION},)
	GIT_VERSION=${GIT_BRANCH}
endif

IMAGE_REGISTRY=harbor.cloudminds.com
IMAGE_TAG=${GIT_VERSION}
ifeq (${IMAGE_TAG},master)
   IMAGE_TAG = latest
endif
# Image URL to use all building/pushing image targets
IMG ?=  ${IMAGE_REGISTRY}/kubegems/kubegems:$(IMAGE_TAG)

GOPACKAGE=github.com/kubegems/gems
ldflags+=-w -s
ldflags+=-X '${GOPACKAGE}/pkg/version.gitVersion=${GIT_VERSION}'
ldflags+=-X '${GOPACKAGE}/pkg/version.gitCommit=${GIT_COMMIT}'
ldflags+=-X '${GOPACKAGE}/pkg/version.buildDate=${BUILD_DATE}'


##@ All

all: container-build ## build all

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

generate: controller-gen ## Generate  WebhookConfiguration, ClusterRole, CustomResourceDefinition objects and code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) paths="./pkg/..." crd  output:crd:artifacts:config=deploy/crd/bases 			# Generate CRDs
	$(CONTROLLER_GEN) paths="./pkg/..." rbac:roleName=manager-role webhook output:dir=deploy/rbac 	# Generate RBAC
	$(CONTROLLER_GEN) paths="./pkg/..." webhook output:dir=deploy/webhook 							# Generate Webhooks
	$(CONTROLLER_GEN) paths="./pkg/..." object:headerFile="hack/boilerplate.go.txt"					# Generate DeepCopy, DeepCopyInto, DeepCopyObject
	$(KUSTOMIZE) build ./deploy/default > deploy/bundle.yaml										# Build bundle.yaml

check: linter ## Static code check.
	${LINTER} run ./...


ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: generate ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

##@ Build

build: ## Build binaries.
	- mkdir -p ${BIN_DIR}
	CGO_ENABLED=0 go build -o ${BIN_DIR}/kubegems -gcflags=all="-N -l" -ldflags="${ldflags}" cmd/main.go

container: build ## Build container image.
	docker build -t ${IMG} .

push: ## Push docker image with the manager.
	docker push ${IMG}

CONTROLLER_GEN = ${BIN_DIR}/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	GOBIN=${BIN_DIR} go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0

KUSTOMIZE = ${BIN_DIR}/kustomize
KUSTOMIZE_VERSION = 4.4.1
kustomize: ## Download kustomize locally if necessary.
	mkdir -p $(BIN_DIR)
	curl -SLf https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv$(KUSTOMIZE_VERSION)/kustomize_v$(KUSTOMIZE_VERSION)_linux_amd64.tar.gz | tar -xz -C $(BIN_DIR)

LINTER = ${BIN_DIR}/golangci-lint
linter: ## Download controller-gen locally if necessary.
	GOBIN=${BIN_DIR} go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.44.0
