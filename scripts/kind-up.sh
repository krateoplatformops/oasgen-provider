#!/bin/bash

kind get kubeconfig >/dev/null 2>&1 || kind create cluster

# kind get kubeconfig >/dev/null 2>&1 || \
#     cat <<EOF | kind create cluster --config=-
# kind: Cluster
# apiVersion: kind.x-k8s.io/v1alpha4
# nodes:
# - role: control-plane
#   extraPortMappings:
#   - containerPort: 30081
#     hostPort: 30081
#     listenAddress: "127.0.0.1"
#     protocol: TCP
# EOF