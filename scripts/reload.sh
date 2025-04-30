#!/bin/bash

./scripts/build.sh

# Uninstall

kubectl delete -f manifests/
# kubectl delete -f crds/

# Install

kubectl apply -f crds/
kubectl apply -f manifests/rdc/
kubectl apply -f manifests/



