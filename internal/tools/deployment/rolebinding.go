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

func UninstallRoleBinding(ctx context.Context, opts UninstallOptions) error {
	return retry.Do(
		func() error {
			obj := rbacv1.RoleBinding{}
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
				opts.Log("RoleBinding successfully uninstalled",
					"name", obj.GetName(), "namespace", obj.GetNamespace())
			}

			return nil
		},
	)
}

func InstallRoleBinding(ctx context.Context, kube client.Client, obj *rbacv1.RoleBinding) error {
	return retry.Do(
		func() error {
			tmp := rbacv1.RoleBinding{}
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

func CreateRoleBinding(opts types.NamespacedName) rbacv1.RoleBinding {
	return rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     opts.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      opts.Name,
				Namespace: opts.Namespace,
			},
		},
	}
}
