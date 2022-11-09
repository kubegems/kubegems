# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

BUILD_DATE?=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_VERSION?=$(shell git describe --tags --dirty --abbrev=0 2>/dev/null || git symbolic-ref --short HEAD)
GIT_COMMIT?=$(shell git rev-parse HEAD 2>/dev/null)
GIT_BRANCH?=$(shell git symbolic-ref --short HEAD 2>/dev/null)
# semver version
VERSION?=$(shell echo "${GIT_VERSION}" | sed -e 's/^v//')

OS?=linux
ARCH?=amd64
BIN_DIR?=$(shell pwd)/bin
PLATFORM?=linux/amd64,linux/arm64

IMAGE_REGISTRY?=docker.io
IMAGE_TAG=${GIT_VERSION}

# Image URL to use all building/pushing image targets
IMG ?=  ${IMAGE_REGISTRY}/kubegems/kubegems:$(IMAGE_TAG)

GOPACKAGE=$(shell go list -m)
ldflags+=-w -s
ldflags+=-X '${GOPACKAGE}/pkg/version.gitVersion=${GIT_VERSION}'
ldflags+=-X '${GOPACKAGE}/pkg/version.gitCommit=${GIT_COMMIT}'
ldflags+=-X '${GOPACKAGE}/pkg/version.buildDate=${BUILD_DATE}'


# HELM BUILD
HELM_USER?=kubegems
HELM_PASSWORD?=
HELM_ADDR?=https://charts.kubegems.io/kubegems

##@ All

all: generate build docker helm-push## build all

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

generate: add-license gen-i18n ## Generate  WebhookConfiguration, ClusterRole, CustomResourceDefinition objects and code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) paths="./pkg/apis/plugins/..." crd  output:crd:artifacts:config=deploy/plugins/kubegems-installer/crds
	$(CONTROLLER_GEN) paths="./pkg/apis/gems/..."    crd  output:crd:artifacts:config=deploy/plugins/kubegems-local/crds
	$(CONTROLLER_GEN) paths="./pkg/apis/models/..."  crd  output:crd:artifacts:config=deploy/plugins/kubegems-models/crds
	$(CONTROLLER_GEN) paths="./pkg/..." object:headerFile="hack/boilerplate.go.txt"

	sed -i 's/^version:.*/version: $(GIT_VERSION)/g' \
		deploy/plugins/kubegems-installer/Chart.yaml \
		deploy/plugins/kubegems-models/Chart.yaml \
		deploy/plugins/kubegems-local/Chart.yaml
	sed -i 's/^appVersion:.*/appVersion: $(GIT_VERSION)/g' \
		deploy/plugins/kubegems-installer/Chart.yaml \
		deploy/plugins/kubegems-models/Chart.yaml \
		deploy/plugins/kubegems-local/Chart.yaml

	sed -i 's/kubegemsVersion:.*/kubegemsVersion: $(GIT_VERSION)/g' deploy/kubegems.yaml
	sed -i 's/export KUBEGEMS_VERSION.*#/export KUBEGEMS_VERSION=$(GIT_VERSION)  #/g' README.md README_zh.md

	helm template --namespace kubegems-installer --include-crds \
	--set global.kubegemsVersion=$(GIT_VERSION) \
	kubegems-installer deploy/plugins/kubegems-installer \
	| kubectl annotate -f -  --local  -oyaml \
	meta.helm.sh/release-name=kubegems-installer meta.helm.sh/release-namespace=kubegems-installer \
	> deploy/installer.yaml

	# go run scripts/generate-system-alert/main.go

swagger:
	go install github.com/swaggo/swag/cmd/swag@v1.8.4
	swag f -g cmd/main.go
	swag i --parseDependency --parseInternal -g cmd/main.go -o docs/swagger

check: linter ## Static code check.
	${LINTER} run ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: generate ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

gen-i18n:
	go run internal/cmd/i18n/main.go gen

collect-i18n:
	go run internal/cmd/i18n/main.go collect

define go-build
	@echo "Building ${1}/${2}"
	@CGO_ENABLED=0 GOOS=${1} GOARCH=$(2) go build -gcflags=all="-N -l" -ldflags="${ldflags}" -o ${BIN_DIR}/kubegems-$(1)-$(2) cmd/main.go
endef

##@ Build
build: build-files build-binaries-all

build-binaries-all: ## Build binaries.
	- mkdir -p ${BIN_DIR}
	$(call go-build,linux,amd64)
	$(call go-build,linux,arm64)

build-binaries:
	$(call go-build,${OS},${ARCH})
	- mkdir -p ${BIN_DIR}
	@cp ${BIN_DIR}/kubegems-${OS}-${ARCH} ${BIN_DIR}/kubegems

build-files: build-binaries ## Build around files
	${BIN_DIR}/kubegems plugins -c ${BIN_DIR}/plugins -s deploy/plugins download deploy/plugins/common
	${BIN_DIR}/kubegems plugins -c ${BIN_DIR}/plugins -s deploy/plugins download deploy/plugins/*
	${BIN_DIR}/kubegems plugins -c ${BIN_DIR}/plugins -s deploy/plugins template deploy/plugins/* | \
		${BIN_DIR}/kubegems plugins -c ${BIN_DIR}/plugins -s deploy/plugins download -
	cp -rf deploy/plugins ${BIN_DIR}/
	cp -rf deploy/*.yaml ${BIN_DIR}/plugins/
	mkdir -p ${BIN_DIR}/config
	cp -rf config/promql_tpl.yaml ${BIN_DIR}/config/
	cp -rf config/dashboards/ ${BIN_DIR}/config/dashboards/

CHARTS = kubegems kubegems-local kubegems-installer kubegems-models
helm-readme: readme-generator
	$(foreach var,$(CHARTS),readme-generator -v deploy/plugins/$(var)/values.yaml -r deploy/plugins/$(var)/README.md;)

helm-package: ## Build helm chart
	$(foreach var,$(CHARTS),helm package -u -d ${BIN_DIR}/plugins --version=${VERSION} --app-version=${VERSION} deploy/plugins/$(var);)

helm-push:
	$(foreach var,$(CHARTS),curl -u ${HELM_USER}:${HELM_PASSWORD} --data-binary "@${BIN_DIR}/plugins/$(var)-${VERSION}.tgz" ${HELM_ADDR};)

docker: ## Build container image.
	docker buildx build --platform=${PLATFORM} --push -t ${IMG}  .

KUBECTL_IMG ?=  ${IMAGE_REGISTRY}/kubegems/kubectl:latest
kubectl-image:
	docker buildx build --platform=${PLATFORM} --push -t ${KUBECTL_IMG} -f Dockerfile.kubectl .

clean:
	- rm -rf ${BIN_DIR}

CONTROLLER_GEN = ${BIN_DIR}/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	GOBIN=${BIN_DIR} go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0

KUSTOMIZE = ${BIN_DIR}/kustomize
KUSTOMIZE_VERSION = 4.4.1
kustomize: ## Download kustomize locally if necessary.
	mkdir -p $(BIN_DIR)
	curl -SLf https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv$(KUSTOMIZE_VERSION)/kustomize_v$(KUSTOMIZE_VERSION)_linux_amd64.tar.gz | tar -xz -C $(BIN_DIR)

LINTER = ${BIN_DIR}/golangci-lint
linter: ## Download controller-gen locally if necessary.
	GOBIN=${BIN_DIR} go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.44.0

K8S_VERSION = 1.20.0
setup-envtest: ## setup operator test environment
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	setup-envtest use ${K8S_VERSION}

.PHONY: readme-generator
readme-generator:
ifeq (, $(shell which readme-generator))
	@{ \
	set -e ;\
	echo 'installing readme-generator-for-helm' ;\
	npm install -g readme-generator-for-helm ;\
	}
else
	echo 'readme-generator-for-helm is already installed'
endif

.PHONY: add-license
add-license:
	./scripts/add_license.sh
