package deployment

import (
	"context"

	"github.com/avast/retry-go"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UninstallServiceAccount(ctx context.Context, opts UninstallOptions) error {
	return retry.Do(
		func() error {
			obj := corev1.ServiceAccount{}
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
				opts.Log("ServiceAccount successfully uninstalled",
					"name", obj.GetName(), "namespace", obj.GetNamespace())
			}

			return nil
		},
	)
}

func InstallServiceAccount(ctx context.Context, kube client.Client, obj *corev1.ServiceAccount) error {
	return retry.Do(
		func() error {
			tmp := corev1.ServiceAccount{}
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

func CreateServiceAccount(opts types.NamespacedName) corev1.ServiceAccount {
	return corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
		},
	}
}
