package deployment

import (
	"context"

	"github.com/avast/retry-go"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func InstallClusterRole(ctx context.Context, kube client.Client, obj *rbacv1.ClusterRole) error {
	return retry.Do(
		func() error {
			tmp := rbacv1.ClusterRole{}
			err := kube.Get(ctx, client.ObjectKeyFromObject(obj), &tmp)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return kube.Create(ctx, obj)
				}

				return err
			}

			return nil
		},
	)
}

func UninstallClusterRole(ctx context.Context, opts UninstallOptions) error {
	return retry.Do(
		func() error {
			obj := rbacv1.ClusterRole{}
			err := opts.KubeClient.Get(ctx, opts.NamespacedName, &obj, &client.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}

				return err
			}

			err = opts.KubeClient.Delete(ctx, &obj, &client.DeleteOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}

				return err
			}

			if opts.Log != nil {
				opts.Log("ClusterRole successfully uninstalled",
					"name", obj.GetName(), "namespace", obj.GetNamespace())
			}

			return nil
		},
	)
}

func CreateClusterRole(opts types.NamespacedName) rbacv1.ClusterRole {
	return rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: opts.Name,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{"apiextensions.k8s.io/v1"},
				Resources: []string{"crds"},
				Verbs:     []string{"*"},
			},
		},
	}
}
