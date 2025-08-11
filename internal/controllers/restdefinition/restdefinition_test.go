//go:build integration
// +build integration

package restdefinition

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krateoplatformops/oasgen-provider/apis"
	definitionv1alpha1 "github.com/krateoplatformops/oasgen-provider/apis/restdefinitions/v1alpha1"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/oas2jsonschema"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/objects"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/plurals"

	"github.com/krateoplatformops/plumbing/e2e"
	xenv "github.com/krateoplatformops/plumbing/env"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"github.com/krateoplatformops/provider-runtime/pkg/reconciler"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/record"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
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
	crdPath      = "../../../crds"
	testdataPath = "testdata"
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
		e2e.CreateNamespace("demo-system"),
		e2e.CreateNamespace("krateo-system"),
		e2e.CreateNamespace("gh-system"),
	).Finish(
		envfuncs.DeleteNamespace(namespace),
		envfuncs.DestroyCluster(clusterName),
	)

	os.Exit(testenv.Run(m))
}

type fakelogger struct {
}

var _ logging.Logger = &fakelogger{}

func (l *fakelogger) Debug(msg string, keysAndValues ...interface{}) {
	fmt.Println("DEBUG", msg, keysAndValues)
}

func (l *fakelogger) Info(msg string, keysAndValues ...interface{}) {
	fmt.Println("INFO", msg, keysAndValues)
}

func (l *fakelogger) WithValues(keysAndValues ...interface{}) logging.Logger {
	return l
}

func TestLifecycle_Simple(t *testing.T) {
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

			scenarioDir := filepath.Join(testdataPath, "simple")
			err = decoder.DecodeEachFile(ctx, os.DirFS(scenarioDir),
				"*.yaml",
				decoder.CreateIgnoreAlreadyExists(r))
			if err != nil {
				t.Fatal(err)
			}

			err = decoder.DecodeFile(os.DirFS(scenarioDir), "restdefinition.yaml", &mg)
			if err != nil {
				t.Fatal(err)
			}

			disc := discovery.NewDiscoveryClientForConfigOrDie(cfg.Client().RESTConfig())
			apis.AddToScheme(r.GetScheme())
			conn := connector{
				kube:     kube,
				log:      &fakelogger{},
				recorder: record.NewFakeRecorder(100),
				disc:     disc,
				parser:   oas2jsonschema.NewLibOASParser(),
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

		time.Sleep(5 * time.Second)

		err = r.Get(ctx, mg.GetName(), mg.GetNamespace(), &mg)
		if err != nil {
			t.Fatal(err)
		}

		obs, err := handler.Observe(ctx, &mg)
		if err != nil {
			t.Fatal(err)
		}
		if obs.ResourceExists == false && obs.ResourceUpToDate == true {
			err = handler.Create(ctx, &mg)
			if err != nil {
				t.Fatal(err)
			}
		} else {
			t.Fatal("Unexpected state", obs)
		}

		time.Sleep(50 * time.Second)

		err = r.Get(ctx, mg.GetName(), mg.GetNamespace(), &mg)
		if err != nil {
			t.Fatal(err)
		}

		obs, err = handler.Observe(ctx, &mg)
		if err != nil {
			t.Fatal(err)
		}

		err = r.Get(ctx, mg.GetName(), mg.GetNamespace(), &mg)
		if err != nil {
			t.Fatal(err)
		}

		ctx, err = handleObservation(t, ctx, handler, obs, &mg)
		if err != nil {
			t.Fatal(err)
		}

		err = r.Get(ctx, mg.GetName(), mg.GetNamespace(), &mg)
		if err != nil {
			t.Fatal(err)
		}
		obs, err = handler.Observe(ctx, &mg)

		if obs.ResourceExists == true && obs.ResourceUpToDate == true {
			gvr := plurals.ToGroupVersionResource(schema.GroupVersionKind{
				Group:   mg.Spec.ResourceGroup,
				Version: resourceVersion,
				Kind:    mg.Spec.Resource.Kind,
			})

			// Check if the CRD is generated correctly
			crd := apiextensionsv1.CustomResourceDefinition{}
			err := r.Get(ctx, gvr.Resource+"."+gvr.Group, "", &crd)
			assert.Nil(t, err, "expecting nil error getting generated crd")

			schema := crd.Spec.Versions[0].Schema.OpenAPIV3Schema
			assert.NotNil(t, schema, "expecting schema to be not nil")

			specProps := schema.Properties["spec"].Properties
			_, ok := specProps["name"]
			assert.True(t, ok, "expecting spec to have 'name' property")
			_, ok = specProps["description"]
			assert.True(t, ok, "expecting spec to have 'description' property")

			statusProps := schema.Properties["status"].Properties // name is in the identifiers so it should in the status
			_, ok = statusProps["name"]
			assert.True(t, ok, "expecting status to have 'name' property")

			// Check if the Auth CRD is generated correctly
			authCrd := apiextensionsv1.CustomResourceDefinition{}
			err = r.Get(ctx, "bearerauths."+gvr.Group, "", &authCrd)
			assert.Nil(t, err, "expecting nil error getting generated auth crd")

			authSchema := authCrd.Spec.Versions[0].Schema.OpenAPIV3Schema
			assert.NotNil(t, authSchema, "expecting auth schema to be not nil")

			spec, ok := authSchema.Properties["spec"]
			assert.True(t, ok, "expecting auth schema to have a 'spec' property")
			authSpecProps := spec.Properties
			_, ok = authSpecProps["tokenRef"]
			assert.True(t, ok, "expecting auth spec to have 'token' property")

			return ctx
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

		gvk := schema.GroupVersionKind{
			Group:   mg.Spec.ResourceGroup,
			Version: resourceVersion,
			Kind:    mg.Spec.Resource.Kind,
		}

		gvr := plurals.ToGroupVersionResource(gvk)

		depl := appsv1.Deployment{}
		err = objects.CreateK8sObject(&depl, gvr, types.NamespacedName{
			Namespace: mg.GetNamespace(),
			Name:      mg.GetName(),
		}, filepath.Join(testdataPath, "rdc", "deployment.yaml"))
		if err != nil {
			t.Fatal(err)
		}

		err = wait.For(
			conditions.New(r).ResourceDeleted(&depl),
			wait.WithTimeout(time.Second*30),
			wait.WithInterval(time.Second*1),
		)
		if err != nil {
			t.Fatal(err)
		}

		return ctx
	}).Feature()

	testenv.Test(t, f)
}

func TestLifecycle_GitHubWorkflows(t *testing.T) {
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

			scenarioDir := filepath.Join(testdataPath, "github_workflows")
			err = decoder.DecodeEachFile(ctx, os.DirFS(scenarioDir),
				"*.yaml",
				decoder.CreateIgnoreAlreadyExists(r))
			if err != nil {
				t.Fatal(err)
			}

			err = decoder.DecodeFile(os.DirFS(scenarioDir), "restdefinition.yaml", &mg)
			if err != nil {
				t.Fatal(err)
			}

			disc := discovery.NewDiscoveryClientForConfigOrDie(cfg.Client().RESTConfig())
			apis.AddToScheme(r.GetScheme())
			conn := connector{
				kube:     kube,
				log:      &fakelogger{},
				recorder: record.NewFakeRecorder(100),
				disc:     disc,
				parser:   oas2jsonschema.NewLibOASParser(),
			}

			handler, err = conn.Connect(ctx, &mg)
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Assess("Create", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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

			time.Sleep(5 * time.Second)

			err = r.Get(ctx, mg.GetName(), mg.GetNamespace(), &mg)
			if err != nil {
				t.Fatal(err)
			}

			obs, err := handler.Observe(ctx, &mg)
			if err != nil {
				t.Fatal(err)
			}
			if obs.ResourceExists == false && obs.ResourceUpToDate == true {
				err = handler.Create(ctx, &mg)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal("Unexpected state", obs)
			}

			time.Sleep(50 * time.Second)

			err = r.Get(ctx, mg.GetName(), mg.GetNamespace(), &mg)
			if err != nil {
				t.Fatal(err)
			}

			obs, err = handler.Observe(ctx, &mg)
			if err != nil {
				t.Fatal(err)
			}

			err = r.Get(ctx, mg.GetName(), mg.GetNamespace(), &mg)
			if err != nil {
				t.Fatal(err)
			}

			ctx, err = handleObservation(t, ctx, handler, obs, &mg)
			if err != nil {
				t.Fatal(err)
			}

			err = r.Get(ctx, mg.GetName(), mg.GetNamespace(), &mg)
			if err != nil {
				t.Fatal(err)
			}
			obs, err = handler.Observe(ctx, &mg)

			if obs.ResourceExists == true && obs.ResourceUpToDate == true {
				gvr := plurals.ToGroupVersionResource(schema.GroupVersionKind{
					Group:   mg.Spec.ResourceGroup,
					Version: resourceVersion,
					Kind:    mg.Spec.Resource.Kind,
				})

				// Check if the CRD is generated correctly
				crd := apiextensionsv1.CustomResourceDefinition{}
				err := r.Get(ctx, gvr.Resource+"."+gvr.Group, "", &crd)
				assert.Nil(t, err, "expecting nil error getting generated crd")

				schema := crd.Spec.Versions[0].Schema.OpenAPIV3Schema
				assert.NotNil(t, schema, "expecting schema to be not nil")

				specProps := schema.Properties["spec"].Properties
				_, ok := specProps["ref"]
				assert.True(t, ok, "expecting spec to have 'ref' property")
				_, ok = specProps["inputs"]
				assert.True(t, ok, "expecting spec to have 'inputs' property")

				// Check if the Auth CRD is generated correctly
				authCrd := apiextensionsv1.CustomResourceDefinition{}
				err = r.Get(ctx, "accesstokens."+gvr.Group, "", &authCrd)
				assert.Nil(t, err, "expecting nil error getting generated auth crd")

				authSchema := authCrd.Spec.Versions[0].Schema.OpenAPIV3Schema
				assert.NotNil(t, authSchema, "expecting auth schema to be not nil")

				spec, ok := authSchema.Properties["spec"]
				assert.True(t, ok, "expecting auth schema to have a 'spec' property")
				authSpecProps := spec.Properties
				_, ok = authSpecProps["tokenRef"]
				assert.True(t, ok, "expecting auth spec to have 'token' property")

				return ctx
			}

			return ctx
		}).
		Assess("Delete", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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

			gvk := schema.GroupVersionKind{
				Group:   mg.Spec.ResourceGroup,
				Version: resourceVersion,
				Kind:    mg.Spec.Resource.Kind,
			}

			gvr := plurals.ToGroupVersionResource(gvk)

			depl := appsv1.Deployment{}
			err = objects.CreateK8sObject(&depl, gvr, types.NamespacedName{
				Namespace: mg.GetNamespace(),
				Name:      mg.GetName(),
			}, filepath.Join(testdataPath, "rdc", "deployment.yaml"))
			if err != nil {
				t.Fatal(err)
			}

			err = wait.For(
				conditions.New(r).ResourceDeleted(&depl),
				wait.WithTimeout(time.Second*30),
				wait.WithInterval(time.Second*1),
			)
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		}).Feature()
	testenv.Test(t, f)
}

// TODO: refactor since it refers to composition (probaly coming from CDC)
func handleObservation(t *testing.T, ctx context.Context, handler reconciler.ExternalClient, observation reconciler.ExternalObservation, u *definitionv1alpha1.RestDefinition) (context.Context, error) {
	var err error
	if observation.ResourceExists == true && observation.ResourceUpToDate == true {
		observation, err = handler.Observe(ctx, u)
		if err != nil {
			t.Error("Observing composition.", "error", err)
			return ctx, err
		}
		if observation.ResourceExists == true && observation.ResourceUpToDate == true {
			t.Log("Composition already exists and is ready.")
			return ctx, nil
		}
	} else if observation.ResourceExists == false && observation.ResourceUpToDate == true {
		err = handler.Delete(ctx, u)
		if err != nil {
			t.Error("Deleting composition.", "error", err)
			return ctx, err
		}
	} else if observation.ResourceExists == true && observation.ResourceUpToDate == false {
		err = handler.Update(ctx, u)
		if err != nil {
			t.Error("Updating composition.", "error", err)
			return ctx, err
		}
	} else if observation.ResourceExists == false && observation.ResourceUpToDate == false {
		err = handler.Create(ctx, u)
		if err != nil {
			t.Error("Creating composition.", "error", err)
			return ctx, err
		}
	}
	return ctx, nil
}
