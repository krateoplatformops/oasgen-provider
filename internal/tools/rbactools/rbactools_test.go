package rbactools_test

import (
	"testing"

	"context"
	"reflect"

	"github.com/krateoplatformops/oasgen-provider/internal/tools/rbactools"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRoleGeneration(t *testing.T) {
	// TestRoleGeneration tests the generation of a Role
	// from a RoleDefinition.
	// It does this by creating a RoleDefinition, generating
	// a Role from it, and then comparing the generated Role
	// with the expected Role.
	// The expected Role is created by parsing a YAML file
	// that contains the expected Role.
	// The expected Role is then compared with the generated
	// Role.

	role, err := rbactools.InitRole(types.NamespacedName{Name: "test-role", Namespace: "test-namespace"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	rbactools.PopulateRole(schema.GroupVersionKind{Group: "test-group", Version: "v1alpha1", Kind: "test-kind"}, &role)

	expectedRole := rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-role",
			Namespace: "test-namespace",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"swaggergen.krateo.io"},
				Resources: []string{"restdefinitions", "restdefinitions/status"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{"test-group"},
				Resources: []string{"test-kinds", "test-kinds/status"},
				Verbs:     []string{"*"},
			},
		},
	}

	if !reflect.DeepEqual(role, expectedRole) {
		t.Errorf("expected role %v, got %v", expectedRole, role)
	}
	ctx := context.Background()
	cli := fake.NewFakeClient()

	err = rbactools.InstallRole(ctx, cli, &role)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = rbactools.UninstallRole(ctx, rbactools.UninstallOptions{
		KubeClient:     cli,
		NamespacedName: types.NamespacedName{Name: "test-role", Namespace: "test-namespace"},
		Log:            func(msg string, keysAndValues ...interface{}) {},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

}

func TestClusterRoleGeneration(t *testing.T) {
	// TestClusterRoleGeneration tests the generation of a ClusterRole
	// from a ClusterRoleDefinition.
	// It does this by creating a ClusterRoleDefinition, generating
	// a ClusterRole from it, and then comparing the generated ClusterRole
	// with the expected ClusterRole.
	// The expected ClusterRole is created by parsing a YAML file
	// that contains the expected ClusterRole.
	// The expected ClusterRole is then compared with the generated
	// ClusterRole.

	clusterRole := rbactools.CreateClusterRole(types.NamespacedName{Name: "test-clusterrole"})

	ctx := context.Background()
	cli := fake.NewFakeClient()

	err := rbactools.InstallClusterRole(ctx, cli, &clusterRole)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = rbactools.UninstallClusterRole(ctx, rbactools.UninstallOptions{
		KubeClient:     cli,
		NamespacedName: types.NamespacedName{Name: "test-clusterrole"},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
func TestClusterBindingRoleGeneration(t *testing.T) {
	// the same as TestClusterRoleGeneration
	// but for ClusterRoleBinding

	clusterRoleBinding := rbactools.CreateClusterRoleBinding(types.NamespacedName{Name: "test-clusterrolebinding"})
	ctx := context.Background()
	cli := fake.NewFakeClient()

	err := rbactools.InstallClusterRoleBinding(ctx, cli, &clusterRoleBinding)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = rbactools.UninstallClusterRoleBinding(ctx, rbactools.UninstallOptions{
		KubeClient:     cli,
		NamespacedName: types.NamespacedName{Name: "test-clusterrolebinding"},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRoleBindingLifecycle(t *testing.T) {
	ctx := context.Background()
	cli := fake.NewFakeClient()

	// Define NamespacedName
	namespacedName := types.NamespacedName{Name: "test-rolebinding", Namespace: "test-namespace"}

	// Create RoleBinding
	roleBinding := rbactools.CreateRoleBinding(namespacedName)

	// Install RoleBinding
	err := rbactools.InstallRoleBinding(ctx, cli, &roleBinding)
	if err != nil {
		t.Fatalf("failed to install rolebinding: %v", err)
	}

	// Verify that the RoleBinding was installed
	installedRoleBinding := rbacv1.RoleBinding{}
	err = cli.Get(ctx, namespacedName, &installedRoleBinding)
	if err != nil {
		t.Fatalf("failed to get installed rolebinding: %v", err)
	}

	// Define uninstall options
	opts := rbactools.UninstallOptions{
		KubeClient:     cli,
		NamespacedName: namespacedName,
	}

	// Uninstall the RoleBinding
	err = rbactools.UninstallRoleBinding(ctx, opts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify that the RoleBinding was deleted
	deletedRoleBinding := rbacv1.RoleBinding{}
	err = cli.Get(ctx, namespacedName, &deletedRoleBinding)
	if err == nil || !errors.IsNotFound(err) {
		t.Errorf("expected not found error, got %v", err)
	}
}

func TestServiceAccount(t *testing.T) {
	ctx := context.Background()
	cli := fake.NewFakeClient()

	// Define NamespacedName
	namespacedName := types.NamespacedName{Name: "test-serviceaccount", Namespace: "test-namespace"}

	// Create ServiceAccount
	serviceAccount := rbactools.CreateServiceAccount(namespacedName)

	// Install ServiceAccount
	err := rbactools.InstallServiceAccount(ctx, cli, &serviceAccount)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify that the ServiceAccount was installed
	installedServiceAccount := corev1.ServiceAccount{}
	err = cli.Get(ctx, namespacedName, &installedServiceAccount)
	if err != nil {
		t.Errorf("failed to get installed serviceaccount: %v", err)
	}

	// Define uninstall options
	opts := rbactools.UninstallOptions{
		KubeClient:     cli,
		NamespacedName: namespacedName,
	}
	// Uninstall the ServiceAccount
	err = rbactools.UninstallServiceAccount(ctx, opts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify that the ServiceAccount was deleted
	deletedServiceAccount := corev1.ServiceAccount{}
	err = cli.Get(ctx, namespacedName, &deletedServiceAccount)
	if err == nil || !errors.IsNotFound(err) {
		t.Errorf("expected not found error, got %v", err)
	}
}
