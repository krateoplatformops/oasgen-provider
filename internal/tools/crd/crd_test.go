//go:build integration
// +build integration

package crd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/krateoplatformops/snowplow/plumbing/e2e"
	xenv "github.com/krateoplatformops/snowplow/plumbing/env"
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

		gvr := InferGroupResource(gvk.GroupKind()).WithVersion(gvk.Version)

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

func TestInferGroupResource(t *testing.T) {
	table := []struct {
		gk   schema.GroupKind
		want schema.GroupResource
	}{
		{
			gk:   schema.GroupKind{Group: "core.krateo.io", Kind: "CardTemplate"},
			want: schema.GroupResource{Group: "core.krateo.io", Resource: "cardtemplates"},
		},
	}

	for i, tc := range table {
		got := InferGroupResource(tc.gk)
		if diff := cmp.Diff(got, tc.want); len(diff) > 0 {
			t.Fatalf("[tc: %d] diff: %s", i, diff)
		}
	}
}

func TestGet(t *testing.T) {

	f := features.New("Setup").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			kube, err := client.New(cfg.Client().RESTConfig(), client.Options{})

			if err != nil {
				t.Fatal(err)
			}

			gvr := schema.GroupVersionResource{
				Group:    "core.krateo.io",
				Version:  "v1alpha1",
				Resource: "CompositionDefinition",
			}

			crd, err := Get(context.Background(), kube, gvr)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if crd != nil {
				t.Logf("CRD exists: %v", crd.Name)
			} else {
				t.Logf("CRD does not exist")
			}
			return ctx
		}).Assess("Lookup", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		return ctx
	}).Feature()

	testenv.Test(t, f)

}
