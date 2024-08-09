package deployment_test

import (
	"context"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/krateoplatformops/oasgen-provider/internal/tools/deployment"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestUninstallDeployment(t *testing.T) {
	ctx := context.TODO()

	// Create a fake client
	client := fake.NewClientBuilder().Build()

	// Create a deployment object
	deploymentObj := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "test-namespace",
		},
	}

	// Create uninstall options
	uninstallOpts := deployment.UninstallOptions{
		KubeClient:     client,
		NamespacedName: types.NamespacedName{Name: deploymentObj.Name, Namespace: deploymentObj.Namespace},
		Log:            nil, // Provide your own log function if needed
	}

	// Uninstall the deployment
	err := deployment.UninstallDeployment(ctx, uninstallOpts)
	if err != nil {
		t.Errorf("failed to uninstall deployment: %v", err)
	}

	// Verify that the deployment is uninstalled
	err = client.Get(ctx, types.NamespacedName{Name: deploymentObj.Name, Namespace: deploymentObj.Namespace}, deploymentObj)
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected deployment to be uninstalled, got error: %v", err)
	}
}

func TestInstallDeployment(t *testing.T) {
	ctx := context.TODO()

	// Create a fake client
	client := fake.NewClientBuilder().Build()

	// Create a deployment object
	deploymentObj := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "test-namespace",
		},
	}

	// Install the deployment
	err := deployment.InstallDeployment(ctx, client, deploymentObj)
	if err != nil {
		t.Errorf("failed to install deployment: %v", err)
	}

	// Verify that the deployment is installed
	err = client.Get(ctx, types.NamespacedName{Name: deploymentObj.Name, Namespace: deploymentObj.Namespace}, deploymentObj)
	if err != nil {
		t.Errorf("expected deployment to be installed, got error: %v", err)
	}
}

func TestCreateDeployment(t *testing.T) {
	// Create a GroupVersionResource and NamespacedName
	gvr := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	nn := types.NamespacedName{
		Namespace: "test-namespace",
		Name:      "test-deployment",
	}

	// Create the deployment
	deploymentObj, err := deployment.CreateDeployment(gvr, nn)
	if err != nil {
		t.Errorf("failed to create deployment: %v", err)
	}

	// Verify the deployment fields
	if deploymentObj.Name != fmt.Sprintf("%s-%s-controller", gvr.Resource, gvr.Version) {
		t.Errorf("expected deployment name to be %s, got %s", nn.Name, deploymentObj.Name)
	}
	if deploymentObj.Namespace != nn.Namespace {
		t.Errorf("expected deployment namespace to be %s, got %s", nn.Namespace, deploymentObj.Namespace)
	}
	// Add more assertions for other fields if needed
}

func TestLookupDeployment(t *testing.T) {
	ctx := context.TODO()

	// Create a fake client
	client := fake.NewClientBuilder().Build()

	// Create a deployment object
	deploymentObj := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "test-namespace",
		},
	}

	// Create the deployment
	err := client.Create(ctx, deploymentObj)
	if err != nil {
		t.Fatalf("failed to create deployment: %v", err)
	}

	// Lookup the deployment
	found, ready, err := deployment.LookupDeployment(ctx, client, deploymentObj)
	if err != nil {
		t.Errorf("failed to lookup deployment: %v", err)
	}

	// Verify the lookup results
	if !found {
		t.Errorf("expected deployment to be found, got not found")
	}
	if !ready {
		t.Logf("deployment is not ready")
	}
}
