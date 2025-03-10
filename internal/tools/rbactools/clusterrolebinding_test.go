package rbactools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateClusterRoleBinding(t *testing.T) {
	sa := types.NamespacedName{Name: "test-sa", Namespace: "test-namespace"}
	gvr := schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "rolebindings",
	}
	roleBinding, err := CreateClusterRoleBinding(gvr, sa, "testdata/clusterrolebinding_template.yaml", "serviceAccount", "test-sa")

	assert.NoError(t, err)

	expectedRoleBinding := rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      gvr.Resource + "-" + gvr.Version,
			Namespace: sa.Namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
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
}

func TestUninstallClusterRoleBinding(t *testing.T) {
	ctx := context.TODO()

	// Create a fake client
	fakeClient := fake.NewClientBuilder().Build()

	gvr := schema.GroupVersionResource{
		Group:    "testgroup",
		Version:  "v1",
		Resource: "testresource",
	}
	// Create a ClusterRoleBinding object
	clusterRoleBinding, err := CreateClusterRoleBinding(gvr, types.NamespacedName{
		Name: "test-clusterrolebinding",
	}, "testdata/clusterrolebinding_template.yaml")
	require.NoError(t, err)

	// Install the ClusterRoleBinding
	err = InstallClusterRoleBinding(ctx, fakeClient, &clusterRoleBinding)
	require.NoError(t, err)

	// Uninstall the ClusterRoleBinding
	err = UninstallClusterRoleBinding(ctx, UninstallOptions{
		KubeClient: fakeClient,
		NamespacedName: types.NamespacedName{
			Name: gvr.Resource + "-" + gvr.Version,
		},
		Log: nil,
	})
	require.NoError(t, err)

	// Verify that the ClusterRoleBinding is uninstalled
	crb := &rbacv1.ClusterRoleBinding{}
	err = fakeClient.Get(ctx, client.ObjectKeyFromObject(&clusterRoleBinding), crb)
	assert.True(t, apierrors.IsNotFound(err))
}
func TestPopulateClusterRoleBinding(t *testing.T) {
	tests := []struct {
		name     string
		tmp      *rbacv1.ClusterRoleBinding
		obj      *rbacv1.ClusterRoleBinding
		expected []rbacv1.Subject
	}{
		{
			name: "No subjects in tmp",
			tmp:  &rbacv1.ClusterRoleBinding{},
			obj: &rbacv1.ClusterRoleBinding{
				Subjects: []rbacv1.Subject{
					{Kind: "User", Name: "user1", Namespace: "default"},
				},
			},
			expected: []rbacv1.Subject{
				{Kind: "User", Name: "user1", Namespace: "default"},
			},
		},
		{
			name: "No new subjects in obj",
			tmp: &rbacv1.ClusterRoleBinding{
				Subjects: []rbacv1.Subject{
					{Kind: "User", Name: "user1", Namespace: "default"},
				},
			},
			obj: &rbacv1.ClusterRoleBinding{
				Subjects: []rbacv1.Subject{
					{Kind: "User", Name: "user1", Namespace: "default"},
				},
			},
			expected: []rbacv1.Subject{
				{Kind: "User", Name: "user1", Namespace: "default"},
			},
		},
		{
			name: "New subjects in obj",
			tmp: &rbacv1.ClusterRoleBinding{
				Subjects: []rbacv1.Subject{
					{Kind: "User", Name: "user1", Namespace: "default"},
				},
			},
			obj: &rbacv1.ClusterRoleBinding{
				Subjects: []rbacv1.Subject{
					{Kind: "User", Name: "user2", Namespace: "default"},
				},
			},
			expected: []rbacv1.Subject{
				{Kind: "User", Name: "user1", Namespace: "default"},
				{Kind: "User", Name: "user2", Namespace: "default"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			populateClusterRoleBinding(tt.tmp, tt.obj)
			assert.Equal(t, tt.expected, tt.tmp.Subjects)
		})
	}
}
