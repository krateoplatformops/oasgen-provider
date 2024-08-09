package deployment

import (
	"context"
	"testing"

	definitionsv1alpha1 "github.com/krateoplatformops/oasgen-provider/apis/restdefinitions/v1alpha1"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/rbactools"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Cannot Undeploy in fake client, because of crds not working in fake client
// func TestUndeploy(t *testing.T) {
// 	ctx := context.TODO()

// 	// Create a mock KubeClient
// 	mockKubeClient := fake.NewFakeClient()

// 	// Create mock NamespacedName
// 	mockNamespacedName := types.NamespacedName{
// 		Namespace: "mock-namespace",
// 		Name:      "mock-name",
// 	}

// 	// Create mock GVR
// 	mockGVR := schema.GroupVersionResource{
// 		Group:    "mock-group",
// 		Version:  "mock-version",
// 		Resource: "mock-resource",
// 	}

// 	// Create mock Log function
// 	mockLog := func(msg string, keysAndValues ...interface{}) {}

// 	// Create mock SecuritySchemes
// 	mockSecuritySchemes := &orderedmap.Map[string, *v3.SecurityScheme]{}

// 	// Create UndeployOptions
// 	opts := UndeployOptions{
// 		KubeClient:      mockKubeClient,
// 		NamespacedName:  mockNamespacedName,
// 		GVR:             mockGVR,
// 		Log:             mockLog,
// 		SecuritySchemes: mockSecuritySchemes,
// 	}

// 	err := Undeploy(ctx, opts)
// 	if err != nil {
// 		t.Errorf("Undeploy() returned an error: %v", err)
// 	}
// }

func TestDeploy(t *testing.T) {
	ctx := context.TODO()

	// Create a mock KubeClient
	mockKubeClient := fake.NewFakeClient()

	// Create mock NamespacedName
	mockNamespacedName := types.NamespacedName{
		Namespace: "mock-namespace",
		Name:      "mock-name",
	}

	// Create mock RestDefinitionSpec
	mockSpec := &definitionsv1alpha1.RestDefinitionSpec{
		ResourceGroup: "mock-resource-group",
		Resource: definitionsv1alpha1.Resource{
			Kind: "mock-kind",
		},
	}

	// Create mock Role
	mockRole, _ := rbactools.InitRole(types.NamespacedName{
		Namespace: "mock-namespace",
		Name:      "mock-name",
	})

	// Create mock ResourceVersion
	mockResourceVersion := "mock-resource-version"

	// Create mock Log function
	mockLog := func(msg string, keysAndValues ...interface{}) {}

	// Create DeployOptions
	opts := DeployOptions{
		KubeClient:      mockKubeClient,
		NamespacedName:  mockNamespacedName,
		Spec:            mockSpec,
		ResourceVersion: mockResourceVersion,
		Role:            mockRole,
		Log:             mockLog,
	}

	err := Deploy(ctx, opts)
	if err != nil {
		t.Errorf("Deploy() returned an error: %v", err)
	}
}
