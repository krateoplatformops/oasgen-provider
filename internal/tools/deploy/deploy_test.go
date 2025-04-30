//go:build integration
// +build integration

package deploy

import (
	"context"
	"os"
	"testing"

	"github.com/krateoplatformops/snowplow/plumbing/e2e"
	xenv "github.com/krateoplatformops/snowplow/plumbing/env"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
		e2e.CreateNamespace(namespace),
		e2e.CreateNamespace("krateo-system"),

		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				return ctx, err
			}
			r.WithNamespace(namespace)

			return ctx, nil
		},
	).Finish(
		envfuncs.DeleteNamespace(namespace),
		envfuncs.DestroyCluster(clusterName),
	)

	os.Exit(testenv.Run(m))
}

func TestDeploy(t *testing.T) {
	os.Setenv("DEBUG", "1")

	f := features.New("Setup").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Assess("Deploy", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		cli, err := client.New(cfg.Client().RESTConfig(), client.Options{})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
			return ctx
		}
		opts := DeployOptions{
			RBACFolderPath:         "testdata",
			DeploymentTemplatePath: "testdata/deploy.yaml",
			ConfigmapTemplatePath:  "testdata/cm.yaml",
			KubeClient:             cli,
			NamespacedName: types.NamespacedName{
				Namespace: "default",
				Name:      "test-deploy",
			},
			GVR: schema.GroupVersionResource{
				Group:    "compositions.krateo.io",
				Version:  "v1alpha1",
				Resource: "fireworksapps",
			},
			Log: func(msg string, keysAndValues ...any) {},
		}

		_, err = Deploy(context.Background(), cli, opts)
		assert.NoError(t, err)

		return ctx
	}).Assess("Undeploy", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		cli, err := client.New(cfg.Client().RESTConfig(), client.Options{})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
			return ctx
		}

		opts := UndeployOptions{
			RBACFolderPath:         "testdata",
			DeploymentTemplatePath: "testdata/deploy.yaml",
			ConfigmapTemplatePath:  "testdata/cm.yaml",
			KubeClient:             cli,
			NamespacedName: types.NamespacedName{
				Namespace: "default",
				Name:      "test-deploy",
			},
			Log:     func(msg string, keysAndValues ...any) {},
			SkipCRD: true,
			GVR: schema.GroupVersionResource{
				Group:    "compositions.krateo.io",
				Version:  "v1alpha1",
				Resource: "fireworksapps",
			},
		}

		err = Undeploy(context.Background(), opts.KubeClient, opts)
		assert.NoError(t, err)

		return ctx
	},
	).Feature()

	testenv.Test(t, f)
}

func TestLookup(t *testing.T) {
	os.Setenv("DEBUG", "1")

	f := features.New("Lookup").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Assess("Lookup Deployment State", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		cli, err := client.New(cfg.Client().RESTConfig(), client.Options{})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
			return ctx
		}

		opts := DeployOptions{
			RBACFolderPath:         "testdata",
			DeploymentTemplatePath: "testdata/deploy.yaml",
			ConfigmapTemplatePath:  "testdata/cm.yaml",
			KubeClient:             cli,
			NamespacedName: types.NamespacedName{
				Namespace: "default",
				Name:      "test-lookup",
			},
			GVR: schema.GroupVersionResource{
				Group:    "compositions.krateo.io",
				Version:  "v1alpha1",
				Resource: "fireworksapps",
			},
			Log: func(msg string, keysAndValues ...any) {},
		}

		// Deploy first to ensure resources exist for lookup
		ddig, err := Deploy(context.Background(), cli, opts)
		assert.NoError(t, err)
		assert.NotEmpty(t, ddig)

		// Perform the lookup
		digest, err := Lookup(context.Background(), cli, opts)
		assert.NoError(t, err)
		assert.NotEmpty(t, digest)

		assert.Equal(t, ddig, digest)

		return ctx
	}).Feature()

	testenv.Test(t, f)
}
