# Set the shell to bash always
SHELL := /bin/bash

# Look for a .env file, and if present, set make variables from it.
ifneq (,$(wildcard ./.env))
	include .env
	export $(shell sed 's/=.*//' .env)
endif

KIND_CLUSTER_NAME ?= local-dev
KUBECONFIG ?= $(HOME)/.kube/config

VERSION := $(shell git describe --always --tags | sed 's/-/./2' | sed 's/-/./2')
ifndef VERSION
VERSION := 0.0.0
endif

# Tools
KIND=$(shell which kind)
LINT=$(shell which golangci-lint)
KUBECTL=$(shell which kubectl)
SED=$(shell which sed)

.DEFAULT_GOAL := help

 
.PHONY: dev
dev: generate ## run the controller in debug mode
	$(KUBECTL) apply -f crds/ -R
	go run cmd/main.go -d

.PHONY: generate
generate: tidy ## generate all CRDs
	go generate ./...

.PHONY: tidy
tidy: ## go mod tidy
	kubectl delete -f tmp/gen-crds/crds/ -R || true
	rm -rf ./tmp
	go mod tidy

.PHONY: test
test: ## go test
	go test -v ./...

.PHONY: lint
lint: ## go lint
	$(LINT) run

.PHONY: kind-up
kind-up: ## starts a KinD cluster for local development
	@$(KIND) get kubeconfig --name $(KIND_CLUSTER_NAME) >/dev/null 2>&1 || $(KIND) create cluster --name=$(KIND_CLUSTER_NAME)

.PHONY: kind-down
kind-down: ## shuts down the KinD cluster
	@$(KIND) delete cluster --name=$(KIND_CLUSTER_NAME)


.PHONY: demo
demo: ## Run the demo examples
	@$(KUBECTL) create secret generic azuredevops-endpoint --from-literal=token=$(TOKEN) || true
	@$(KUBECTL) apply -f samples/claim.yaml


.PHONY: help
help: ## print this help
	@grep -E '^[a-zA-Z\._-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
