package rbactools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestUninstallRoleBinding(t *testing.T) {
	ctx := context.TODO()
	opts := UninstallOptions{
		KubeClient:     fake.NewClientBuilder().Build(),
		NamespacedName: types.NamespacedName{Name: "test-name", Namespace: "test-namespace"},
		Log:            nil,
	}

	err := UninstallRoleBinding(ctx, opts)
	assert.NoError(t, err)
}

func TestInstallRoleBinding(t *testing.T) {
	ctx := context.TODO()
	kube := fake.NewClientBuilder().Build()
	obj := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-name",
			Namespace: "test-namespace",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "test-name",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "test-sa",
				Namespace: "test-namespace",
			},
		},
	}

	err := InstallRoleBinding(ctx, kube, obj)
	assert.NoError(t, err)

	err = UninstallRoleBinding(ctx, UninstallOptions{
		KubeClient: kube,
		NamespacedName: types.NamespacedName{
			Name:      "test-name",
			Namespace: "test-namespace",
		},
	},
	)
	assert.NoError(t, err)

	// Verify that the RoleBinding is uninstalled
	err = kube.Get(ctx, types.NamespacedName{
		Name:      "test-name",
		Namespace: "test-namespace",
	}, obj)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestCreateRoleBinding(t *testing.T) {
	sa := types.NamespacedName{Name: "test-sa", Namespace: "test-namespace"}
	gvr := schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "rolebindings",
	}
	roleBinding, err := CreateRoleBinding(gvr, sa, "testdata/rolebinding_template.yaml", "serviceAccount", "test-sa")

	assert.NoError(t, err)

	expectedRoleBinding := rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      gvr.Resource + "-" + gvr.Version,
			Namespace: sa.Namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     gvr.Resource + "-" + gvr.Version,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "test-sa",
				Namespace: sa.Namespace,
			},
		},
	}

	assert.Equal(t, expectedRoleBinding.RoleRef, roleBinding.RoleRef)
	assert.Equal(t, expectedRoleBinding.Subjects, roleBinding.Subjects)
	assert.Equal(t, expectedRoleBinding.ObjectMeta.Name, roleBinding.ObjectMeta.Name)
	assert.Equal(t, expectedRoleBinding.ObjectMeta.Namespace, roleBinding.ObjectMeta.Namespace)
}
