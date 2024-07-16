KIND_CLUSTER_NAME ?= "podsql-cluster"
K8S_NODE_IMAGE ?= v1.21.10
ENVTEST_K8S_VERSION ?= 1.30.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

KIND_CLUSTER_CONFIG_DIR=$(shell pwd)/sample/kind
KUBECONFIG_BACKUP_DIR=$(shell pwd)/.kube

all: build

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

kind-create: ## Create a kind cluster named ${KIND_CLUSTER_NAME} locally if necessary and save the kubectl config.
ifeq (1, $(shell kind get clusters | grep ${KIND_CLUSTER_NAME} | wc -l | tr -d ' '))
	@echo "Cluster already exists"
else
	@echo "Creating Cluster"
	kind create cluster --name ${KIND_CLUSTER_NAME} --image=kindest/node:${K8S_NODE_IMAGE} --config ${KIND_CLUSTER_CONFIG_DIR}/cluster.yaml
ifeq ($(IN_DEV_CONTAINER), true)
	@echo "kubeconfig backup =>"
	mkdir -p ${KUBECONFIG_BACKUP_DIR} && kind get kubeconfig --name ${KIND_CLUSTER_NAME} > ${KUBECONFIG_BACKUP_DIR}/kind-conifg.yaml
endif
endif

##@ Development

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

##@ Build

build: fmt vet ## Build podsql binary.
	go build -o bin/podsql .

run: fmt vet ## Run a controller from your host.
	go run ./main.go

##@ Deployment

deploy-mysql: ## Deploy MySQL in the kind cluster.
	kubectl apply -f ${KIND_CLUSTER_CONFIG_DIR}/mysql.yaml

deploy-sqlserver: ## Deploy SQL Server in the kind cluster.
	kubectl apply -f ${KIND_CLUSTER_CONFIG_DIR}/sqlserver.yaml
