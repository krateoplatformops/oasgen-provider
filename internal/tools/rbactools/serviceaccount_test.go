package rbactools

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestUninstallServiceAccount(t *testing.T) {
	ctx := context.TODO()

	// Create a fake client
	kubeClient := fake.NewFakeClient()

	// Create the service account object
	serviceAccount := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service-account",
			Namespace: "test-namespace",
		},
	}

	// Install the service account
	err := InstallServiceAccount(ctx, kubeClient, &serviceAccount)
	if err != nil {
		t.Fatalf("Failed to install service account: %v", err)
	}

	// Uninstall the service account
	err = UninstallServiceAccount(ctx, UninstallOptions{
		KubeClient:     kubeClient,
		NamespacedName: types.NamespacedName{Name: "test-service-account", Namespace: "test-namespace"},
	})
	if err != nil {
		t.Fatalf("Failed to uninstall service account: %v", err)
	}

	// Verify that the service account is uninstalled
	err = kubeClient.Get(ctx, types.NamespacedName{Name: "test-service-account", Namespace: "test-namespace"}, &corev1.ServiceAccount{})
	if !apierrors.IsNotFound(err) {
		t.Fatalf("Expected service account to be uninstalled, but it still exists: %v", err)
	}
}
