package crds

import (
	"context"

	"github.com/avast/retry-go"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UninstallCRD(ctx context.Context, kube client.Client, gr schema.GroupResource) error {
	return retry.Do(
		func() error {
			obj := apiextensionsv1.CustomResourceDefinition{}
			err := kube.Get(ctx, client.ObjectKey{Name: gr.String()}, &obj, &client.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}

				return err
			}

			err = kube.Delete(ctx, &obj, &client.DeleteOptions{})
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

func InstallCRD(ctx context.Context, kube client.Client, obj *apiextensionsv1.CustomResourceDefinition) error {
	return retry.Do(
		func() error {
			tmp := apiextensionsv1.CustomResourceDefinition{}
			err := kube.Get(ctx, client.ObjectKeyFromObject(obj), &tmp)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return kube.Create(ctx, obj)
				}

				return err
			}

			gracePeriod := int64(0)
			_ = kube.Delete(ctx, &tmp, &client.DeleteOptions{GracePeriodSeconds: &gracePeriod})

			return kube.Create(ctx, obj)
		},
	)
}

func LookupCRD(ctx context.Context, kube client.Client, gvr schema.GroupVersionResource) (bool, error) {
	res := apiextensionsv1.CustomResourceDefinition{}
	err := kube.Get(ctx, client.ObjectKey{Name: gvr.GroupResource().String()}, &res, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}

	for _, el := range res.Spec.Versions {
		if el.Name == gvr.Version {
			return true, nil
		}
	}

	return false, nil
}

func UnmarshalCRD(dat []byte) (*apiextensionsv1.CustomResourceDefinition, error) {
	if !clientsetscheme.Scheme.IsGroupRegistered("apiextensions.k8s.io") {
		_ = apiextensionsscheme.AddToScheme(clientsetscheme.Scheme)
	}

	s := json.NewYAMLSerializer(json.DefaultMetaFactory,
		clientsetscheme.Scheme,
		clientsetscheme.Scheme)

	res := &apiextensionsv1.CustomResourceDefinition{}
	_, _, err := s.Decode(dat, nil, res)
	return res, err
}
