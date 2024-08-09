package deployment_test

// Cannot Install CRD in fake client, timeout error

// import (
// 	"context"
// 	"fmt"
// 	"testing"

// 	"gopkg.in/yaml.v2"
// 	apierrors "k8s.io/apimachinery/pkg/api/errors"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// 	_ "embed"

// 	"github.com/krateoplatformops/oasgen-provider/internal/tools/deployment"
// 	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
// 	"k8s.io/apimachinery/pkg/runtime/schema"
// 	"sigs.k8s.io/controller-runtime/pkg/client"
// 	"sigs.k8s.io/controller-runtime/pkg/client/fake"
// )

// //go:embed swaggergen.krateo.io_restdefinitions.yaml
// var crdYAML []byte

// func TestLookupCRD(t *testing.T) {
// 	ctx := context.Background()
// 	kube := fake.NewFakeClient()

// 	crd := &apiextensionsv1.CustomResourceDefinition{}
// 	err := yaml.Unmarshal(crdYAML, crd)
// 	if err != nil {
// 		t.Fatalf("failed to unmarshal CRD: %v", err)
// 	}

// 	fmt.Println(string(crdYAML))

// 	// // Create a test CustomResourceDefinition
// 	// crd := &apiextensionsv1.CustomResourceDefinition{
// 	// 	ObjectMeta: metav1.ObjectMeta{
// 	// 		Name: "test-crd",
// 	// 	},
// 	// 	Spec: apiextensionsv1.CustomResourceDefinitionSpec{
// 	// 		Group: "testgroup",
// 	// 		Names: apiextensionsv1.CustomResourceDefinitionNames{
// 	// 			Plural: "testcrds",
// 	// 			Kind:   "TestCRD",
// 	// 		},
// 	// 		Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
// 	// 			{
// 	// 				Name: "v1",
// 	// 			},
// 	// 		},
// 	// 	},
// 	// }

// 	// Install the test CustomResourceDefinition
// 	err = deployment.InstallCRD(ctx, kube, crd)
// 	if err != nil {
// 		t.Fatalf("failed to install CRD: %v", err)
// 	}

// 	// Lookup the installed CustomResourceDefinition
// 	gvr := schema.GroupVersionResource{
// 		Group:    "testgroup",
// 		Version:  "v1",
// 		Resource: "testcrds",
// 	}
// 	found, err := deployment.LookupCRD(ctx, kube, gvr)
// 	if err != nil {
// 		t.Fatalf("failed to lookup CRD: %v", err)
// 	}

// 	// Verify that the CustomResourceDefinition was found
// 	if !found {
// 		t.Errorf("expected CRD to be found, but it was not")
// 	}
// }
// func TestUninstallCRD(t *testing.T) {
// 	ctx := context.Background()
// 	kube := fake.NewFakeClient()

// 	gr := schema.GroupResource{
// 		Group:    "testgroup",
// 		Resource: "testcrds",
// 	}

// 	// Uninstall the test CustomResourceDefinition
// 	err := deployment.UninstallCRD(ctx, kube, gr)
// 	if err != nil {
// 		t.Fatalf("failed to uninstall CRD: %v", err)
// 	}

// 	// Verify that the CustomResourceDefinition is uninstalled
// 	obj := &apiextensionsv1.CustomResourceDefinition{}
// 	err = kube.Get(ctx, client.ObjectKey{Name: gr.String()}, obj)
// 	if !apierrors.IsNotFound(err) {
// 		t.Errorf("expected CRD to be uninstalled, but it still exists")
// 	}
// }

// func TestInstallCRD(t *testing.T) {
// 	ctx := context.Background()
// 	kube := fake.NewFakeClient()

// 	// Create a test CustomResourceDefinition
// 	crd := &apiextensionsv1.CustomResourceDefinition{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: "test-crd",
// 		},
// 		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
// 			Group: "testgroup",
// 			Names: apiextensionsv1.CustomResourceDefinitionNames{
// 				Plural: "testcrds",
// 				Kind:   "TestCRD",
// 			},
// 			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
// 				{
// 					Name: "v1",
// 				},
// 			},
// 		},
// 	}

// 	// Install the test CustomResourceDefinition
// 	err := deployment.InstallCRD(ctx, kube, crd)
// 	if err != nil {
// 		t.Logf("failed to install CRD: %v", err)
// 		// t.Fatalf("failed to install CRD: %v", err)
// 	}

// 	// Lookup the installed CustomResourceDefinition
// 	gvr := schema.GroupVersionResource{
// 		Group:    "testgroup",
// 		Version:  "v1",
// 		Resource: "testcrds",
// 	}
// 	found, err := deployment.LookupCRD(ctx, kube, gvr)
// 	if err != nil {
// 		t.Fatalf("failed to lookup CRD: %v", err)
// 	}

// 	// Verify that the CustomResourceDefinition was found
// 	if !found {
// 		t.Errorf("expected CRD to be found, but it was not")
// 	}
// }
