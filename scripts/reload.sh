#!/bin/bash

./scripts/build.sh

# Uninstall

kubectl delete -f manifests/
# kubectl delete -f crds/

# Install

kubectl apply -f crds/
kubectl apply -f manifests/


# Deploy 
# kubectl apply -f manifests/ns.yaml
# kubectl apply -f manifests/sa.yaml
# kubectl apply -f manifests/deployment.yaml
# kubectl apply -f manifests/registration.yaml
# kubectl apply -f manifests/service.yaml
