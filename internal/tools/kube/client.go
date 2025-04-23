package kube

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/avast/retry-go"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplyGenericObject applies a generic object to the cluster
func Apply(ctx context.Context, kube client.Client, obj client.Object, opts ApplyOptions) error {
	return retry.Do(
		func() error {
			tmp := &unstructured.Unstructured{}
			tmp.SetKind(obj.GetObjectKind().GroupVersionKind().Kind)
			tmp.SetAPIVersion(obj.GetObjectKind().GroupVersionKind().GroupVersion().String())
			err := kube.Get(ctx, client.ObjectKeyFromObject(obj), tmp)
			if err != nil {
				if apierrors.IsNotFound(err) {
					createOpts := &client.CreateOptions{
						DryRun:          opts.DryRun,
						FieldManager:    opts.FieldManager,
						FieldValidation: opts.FieldValidation,
					}
					return kube.Create(ctx, obj, createOpts)
				}
				return err
			}

			obj.SetResourceVersion(tmp.GetResourceVersion())
			updateOpts := &client.UpdateOptions{
				DryRun:          opts.DryRun,
				FieldManager:    opts.FieldManager,
				FieldValidation: opts.FieldValidation,
			}
			return kube.Update(ctx, obj, updateOpts)
		},
	)
}

func Uninstall(ctx context.Context, kube client.Client, obj client.Object, opts UninstallOptions) error {
	return retry.Do(
		func() error {
			tmp := &unstructured.Unstructured{}
			tmp.SetKind(obj.GetObjectKind().GroupVersionKind().Kind)
			tmp.SetAPIVersion(obj.GetObjectKind().GroupVersionKind().GroupVersion().String())
			err := kube.Get(ctx, client.ObjectKeyFromObject(obj), tmp)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}

				return err
			}

			obj.SetResourceVersion(tmp.GetResourceVersion())

			err = kube.Delete(ctx, obj, &client.DeleteOptions{
				DryRun:             opts.DryRun,
				Preconditions:      opts.Preconditions,
				PropagationPolicy:  opts.PropagationPolicy,
				GracePeriodSeconds: opts.GracePeriodSeconds,
			})
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}

				return err
			}
			return nil
		},
	)
}

func Get(ctx context.Context, kube client.Client, obj client.Object) error {
	return kube.Get(ctx, client.ObjectKeyFromObject(obj), obj)
}
