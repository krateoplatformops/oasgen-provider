//go:build integration
// +build integration

package definition

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/krateoplatformops/oasgen-provider/apis"
	definitionv1alpha1 "github.com/krateoplatformops/oasgen-provider/apis/restdefinitions/v1alpha1"

	"github.com/go-logr/logr"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"github.com/krateoplatformops/provider-runtime/pkg/reconciler"
	"github.com/krateoplatformops/snowplow/plumbing/e2e"
	xenv "github.com/krateoplatformops/snowplow/plumbing/env"
	"k8s.io/client-go/tools/record"

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
	testdataPath  = "testdata"
	manifestsPath = "../../../manifests"
	scriptsPath   = "../../scripts"

	testFileName = "rdworkflows.yaml"
)

func TestMain(m *testing.M) {
	xenv.SetTestMode(true)

	namespace = "gh-system"
	clusterName = "krateo"
	testenv = env.New()

	testenv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), clusterName),
		envfuncs.SetupCRDs(crdPath, "swaggergen.krateo.io_restdefinitions.yaml"),
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
	// envfuncs.DeleteNamespace(namespace),
	// envfuncs.DestroyCluster(clusterName),
	)

	os.Exit(testenv.Run(m))
}
func TestDefinition(t *testing.T) {
	os.Setenv("DEBUG", "1")

	var handler reconciler.ExternalClient
	mg := definitionv1alpha1.RestDefinition{}
	f := features.New("Setup").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			kube, err := client.New(cfg.Client().RESTConfig(), client.Options{})
			if err != nil {
				t.Fatal(err)
			}

			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			os.Setenv("RDC_TEMPLATE_DEPLOYMENT_PATH", filepath.Join(testdataPath, "rdc", "deployment.yaml"))
			os.Setenv("RDC_TEMPLATE_CONFIGMAP_PATH", filepath.Join(testdataPath, "rdc", "configmap.yaml"))
			os.Setenv("RDC_RBAC_CONFIG_FOLDER", filepath.Join(testdataPath, "rdc", "rbac"))

			err = decoder.DecodeEachFile(ctx, os.DirFS(filepath.Join(testdataPath)),
				"*.yaml",
				decoder.CreateIgnoreAlreadyExists(r))
			if err != nil {
				t.Fatal(err)
			}

			err = decoder.DecodeFile(os.DirFS(filepath.Join(testdataPath)), testFileName, &mg)
			if err != nil {
				t.Fatal(err)
			}
			apis.AddToScheme(r.GetScheme())
			conn := connector{
				kube:     kube,
				log:      logging.NewLogrLogger(logr.Logger{}),
				recorder: record.NewFakeRecorder(100),
			}

			handler, err = conn.Connect(ctx, &mg)
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		}).Assess("Create", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		err = r.Get(ctx, mg.GetName(), mg.GetNamespace(), &mg)
		if err != nil {
			t.Fatal(err)
		}

		err = handler.Create(ctx, &mg)
		if err != nil {
			t.Fatal(err)
		}

		return ctx
	}).Assess("Delete", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
		}

		err = r.Get(ctx, mg.GetName(), mg.GetNamespace(), &mg)
		if err != nil {
			t.Fatal(err)
		}

		err = handler.Delete(ctx, &mg)
		if err != nil {
			t.Fatal(err)
		}

		err = r.Get(ctx, mg.GetName(), mg.GetNamespace(), &mg)
		if apierrors.IsNotFound(err) {
			return ctx
		} else if err != nil {
			t.Fatal(err)
		} else if err == nil {
			t.Fatal("Resource not deleted")
		}

		return ctx
	}).Feature()

	testenv.Test(t, f)
}
