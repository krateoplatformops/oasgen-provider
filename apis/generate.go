//go:build generate
// +build generate

// Remove existing CRDs
//go:generate rm -rf ../crds

// Generate deepcopy methodsets and CRD manifests
//go:generate go run -tags generate sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../hack/boilerplate.go.txt paths=./... crd:crdVersions=v1 output:artifacts:config=../crds

package apis

import (
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen" //nolint:typecheck
)
