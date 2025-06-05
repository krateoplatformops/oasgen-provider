//go:build integration
// +build integration

package crd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/krateoplatformops/oasgen-provider/internal/tools/plurals"
	"github.com/krateoplatformops/plumbing/e2e"
	xenv "github.com/krateoplatformops/plumbing/env"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/kind"
)

var (
	testenv     env.Environment
	clusterName string
	namespace   string
)

const (
	crdPath       = "../../../crds"
	testdataPath  = "../../../testdata"
	manifestsPath = "../../../manifests"
	scriptsPath   = "../../../scripts"

	testFileName = "compositiondefinition-common.yaml"
)

func TestMain(m *testing.M) {
	xenv.SetTestMode(true)

	namespace = "demo-system"
	clusterName = "krateo"
	testenv = env.New()

	testenv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), clusterName),
		envfuncs.SetupCRDs(crdPath, "core.krateo.io_compositiondefinitions.yaml"),
		e2e.CreateNamespace(namespace),
		e2e.CreateNamespace("krateo-system"),

		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				return ctx, err
			}
			r.WithNamespace(namespace)

			// Install CRDs
			err = decoder.DecodeEachFile(
				ctx, os.DirFS(filepath.Join(testdataPath, "compositiondefinitions_test/crds/finops")), "*.yaml",
				decoder.CreateHandler(r),
			)
			err = decoder.DecodeEachFile(
				ctx, os.DirFS(filepath.Join(testdataPath, "compositiondefinitions_test/crds/argocd")), "*.yaml",
				decoder.CreateHandler(r),
			)
			err = decoder.DecodeEachFile(
				ctx, os.DirFS(filepath.Join(testdataPath, "compositiondefinitions_test/crds/azuredevops-provider")), "*.yaml",
				decoder.CreateHandler(r),
			)
			err = decoder.DecodeEachFile(
				ctx, os.DirFS(filepath.Join(testdataPath, "compositiondefinitions_test/crds/git-provider")), "*.yaml",
				decoder.CreateHandler(r),
			)
			err = decoder.DecodeEachFile(
				ctx, os.DirFS(filepath.Join(testdataPath, "compositiondefinitions_test/crds/github-provider")), "*.yaml",
				decoder.CreateHandler(r),
			)
			err = decoder.DecodeEachFile(
				ctx, os.DirFS(filepath.Join(testdataPath, "compositiondefinitions_test/crds/resourcetree")), "*.yaml",
				decoder.CreateHandler(r),
			)

			return ctx, nil
		},
	).Finish(
		envfuncs.DeleteNamespace(namespace),
		envfuncs.DestroyCluster(clusterName),
	)

	os.Exit(testenv.Run(m))
}
func TestLookup(t *testing.T) {
	os.Setenv("DEBUG", "1")

	f := features.New("Setup").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Assess("Lookup", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		kube, err := client.New(cfg.Client().RESTConfig(), client.Options{})
		if err != nil {
			t.Fatal(err)
		}

		gvk := schema.GroupVersionKind{
			Group:   "core.krateo.io",
			Version: "v1alpha1",
			Kind:    "CompositionDefinition",
		}

		gvr := plurals.ToGroupVersionResource((gvk.GroupKind()).WithVersion(gvk.Version))

		ok, err := Lookup(context.Background(), kube, gvr)
		if err != nil {
			t.Fatal(err)
		}

		if ok {
			t.Logf("crd: %v, exists", gvk)
		} else {
			t.Logf("crd: %v, does not exists", gvk)
		}
		return ctx
	}).Feature()

	testenv.Test(t, f)
}
func TestUnmarshal(t *testing.T) {
	t.Run("ValidCRD", func(t *testing.T) {
		validCRD := `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: testresources.example.com
spec:
  group: example.com
  names:
    kind: TestResource
    listKind: TestResourceList
    plural: testresources
    singular: testresource
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
`
		crd, err := Unmarshal([]byte(validCRD))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if crd.Name != "testresources.example.com" {
			t.Errorf("expected CRD name to be 'testresources.example.com', got '%s'", crd.Name)
		}
	})

	t.Run("InvalidCRD", func(t *testing.T) {
		invalidCRD := `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: testresources.example.com
spec:
  group: example.com
  names:
    kind: TestResource
    listKind: TestResourceList
    plural: testresources
    singular: testresource
  scope: Namespaced
  versions:
  - name: v1
	served: true
    storage: true
    invalidField: true
`
		_, err := Unmarshal([]byte(invalidCRD))
		if err == nil {
			t.Fatal("expected an error, but got none")
		}
	})

	t.Run("EmptyInput", func(t *testing.T) {
		_, err := Unmarshal([]byte{})
		if err == nil {
			t.Fatal("expected an error for empty input, but got none")
		}
	})
}
func TestUninstall(t *testing.T) {
	os.Setenv("DEBUG", "1")

	f := features.New("Setup").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Assess("Lookup", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		kube, err := client.New(cfg.Client().RESTConfig(), client.Options{})
		if err != nil {
			t.Fatal(err)
		}

		gvk := schema.GroupVersionKind{
			Group:   "core.krateo.io",
			Version: "v1alpha1",
			Kind:    "CompositionDefinition",
		}

		gvr := plurals.ToGroupVersionResource((gvk.GroupKind()).WithVersion(gvk.Version))

		err = Uninstall(context.Background(), kube, schema.GroupResource{
			Group:    gvk.Group,
			Resource: gvr.Resource,
		})
		if err != nil {
			t.Fatal(err)
		}

		ok, err := Lookup(context.Background(), kube, gvr)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Logf("crd: %v, still exists", gvk)
		} else {
			t.Logf("crd: %v, does not exists", gvk)
		}

		return ctx
	}).Feature()

	testenv.Test(t, f)
}
