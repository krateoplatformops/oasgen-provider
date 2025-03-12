package rbactools

import (
	"context"
	"fmt"
	"os"

	"github.com/avast/retry-go"
	"github.com/krateoplatformops/oasgen-provider/internal/templates"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func populateClusterRoleBinding(tmp *rbacv1.ClusterRoleBinding, obj *rbacv1.ClusterRoleBinding) {
	for _, sub := range obj.Subjects {
		found := false
		for _, tmpSub := range tmp.Subjects {
			if sub.Name == tmpSub.Name && sub.Namespace == tmpSub.Namespace && sub.Kind == tmpSub.Kind {
				found = true
				break
			}
		}

		if !found {
			tmp.Subjects = append(tmp.Subjects, sub)
		}
	}
}

func CreateClusterRoleBinding(gvr schema.GroupVersionResource, nn types.NamespacedName, path string, additionalvalues ...string) (rbacv1.ClusterRoleBinding, error) {
	templateF, err := os.ReadFile(path)
	if err != nil {
		return rbacv1.ClusterRoleBinding{}, fmt.Errorf("failed to read clusterrole binding template file: %w", err)
	}

	values := templates.Values(templates.Renderoptions{
		Group:     gvr.Group,
		Version:   gvr.Version,
		Resource:  gvr.Resource,
		Namespace: nn.Namespace,
		Name:      nn.Name,
	})

	if len(additionalvalues)%2 != 0 {
		return rbacv1.ClusterRoleBinding{}, fmt.Errorf("additionalvalues must be in pairs: %w", err)
	}
	for i := 0; i < len(additionalvalues); i += 2 {
		values[additionalvalues[i]] = additionalvalues[i+1]
	}

	template := templates.Template(string(templateF))
	dat, err := template.Render(values)
	if err != nil {
		return rbacv1.ClusterRoleBinding{}, err
	}

	s := json.NewYAMLSerializer(json.DefaultMetaFactory,
		clientsetscheme.Scheme,
		clientsetscheme.Scheme)

	res := rbacv1.ClusterRoleBinding{}
	_, _, err = s.Decode(dat, nil, &res)
	if err != nil {
		return rbacv1.ClusterRoleBinding{}, fmt.Errorf("failed to decode clusterrole binding: %w", err)
	}

	return res, err
}

func InstallClusterRoleBinding(ctx context.Context, kube client.Client, obj *rbacv1.ClusterRoleBinding) error {
	return retry.Do(
		func() error {
			tmp := rbacv1.ClusterRoleBinding{}
			err := kube.Get(ctx, client.ObjectKeyFromObject(obj), &tmp)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return kube.Create(ctx, obj)
				}

				return err
			}
			populateClusterRoleBinding(&tmp, obj)
			return kube.Update(ctx, &tmp, &client.UpdateOptions{})
		},
	)
}

func UninstallClusterRoleBinding(ctx context.Context, opts UninstallOptions) error {
	return retry.Do(
		func() error {
			obj := rbacv1.ClusterRoleBinding{}
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
				opts.Log("ClusterRoleBinding successfully uninstalled",
					"name", obj.GetName(), "namespace", obj.GetNamespace())
			}

			return nil
		},
	)
}

// func CreateClusterRoleBinding(opts types.NamespacedName) rbacv1.ClusterRoleBinding {
// 	return rbacv1.ClusterRoleBinding{
// 		TypeMeta: metav1.TypeMeta{
// 			APIVersion: "rbac.authorization.k8s.io/v1",
// 			Kind:       "ClusterRoleBinding",
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: opts.Name,
// 		},
// 		RoleRef: rbacv1.RoleRef{
// 			APIGroup: "rbac.authorization.k8s.io",
// 			Kind:     "ClusterRole",
// 			Name:     opts.Name,
// 		},
// 		Subjects: []rbacv1.Subject{
// 			{
// 				Kind:      "ServiceAccount",
// 				Name:      opts.Name,
// 				Namespace: opts.Namespace,
// 			},
// 		},
// 	}
// }
