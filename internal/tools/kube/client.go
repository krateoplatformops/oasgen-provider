package kube

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/avast/retry-go"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
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

func UninstallFromReference(ctx context.Context, kube client.Client, ref v1.ObjectReference, opts UninstallOptions) error {
	return retry.Do(
		func() error {
			tmp := &unstructured.Unstructured{}

			// tmp.SetAPIVersion(gvk.GroupVersion().String())
			// tmp.SetKind(gvk.Kind)

			tmp.SetGroupVersionKind(ref.GroupVersionKind())
			err := kube.Get(ctx, client.ObjectKey{
				Name:      ref.Name,
				Namespace: ref.Namespace,
			}, tmp)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}
				return err
			}

			err = kube.Delete(ctx, tmp, &client.DeleteOptions{
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

func GetFromReference(ctx context.Context, kube client.Client, ref v1.ObjectReference) error {
	tmp := &unstructured.Unstructured{}
	tmp.SetGroupVersionKind(ref.GroupVersionKind())
	err := kube.Get(ctx, client.ObjectKey{
		Name:      ref.Name,
		Namespace: ref.Namespace,
	}, tmp)
	if err != nil {
		return err
	}
	return nil
}

func Get(ctx context.Context, kube client.Client, obj client.Object) error {
	return kube.Get(ctx, client.ObjectKeyFromObject(obj), obj)
}

func CountRestResourcesWithGroup(ctx context.Context, kube client.Client, discovery discovery.DiscoveryInterface, group string) (auth bool, rest int, err error) {
	_, apiResourceList, err := discovery.ServerGroupsAndResources()
	if err != nil {
		return false, 0, fmt.Errorf("failed to discover API resources: %v", err)
	}

	if len(apiResourceList) == 0 {
		return false, 0, fmt.Errorf("no API resources found")
	}
	auth = false
	rest = 0

	for _, apiResource := range apiResourceList {
		gv, err := schema.ParseGroupVersion(apiResource.GroupVersion)
		if err != nil {
			return auth, 0, fmt.Errorf("failed to parse group version: %v", err)
		}

		if gv.Group == group {
			// list the resources of each gvk
			li := unstructured.UnstructuredList{}

			for _, resource := range apiResource.APIResources {
				if resource.Kind == "" {
					continue
				}
				li.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   gv.Group,
					Version: gv.Version,
					Kind:    resource.Kind,
				})

				if strings.HasSuffix(resource.Kind, "Auth") {
					auth = true
				} else {
					err = kube.List(ctx, &li)
					if err != nil {
						if !strings.Contains(err.Error(), "no matches for") {
							return auth, 0, fmt.Errorf("failed to list resources: %v", err)
						}
					}

					rest += len(li.Items)
				}
			}
		}
	}

	return auth, rest, nil
}
