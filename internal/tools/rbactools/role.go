package rbactools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/avast/retry-go"
	"github.com/gobuffalo/flect"
	"github.com/krateoplatformops/oasgen-provider/internal/templates"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateRole creates a Role object from a template file, with the given GroupVersionResource and NamespacedName
// The path is the path to the template file, and additionalvalues are key-value pairs that will be used to render the template
func CreateRole(gvr schema.GroupVersionResource, nn types.NamespacedName, path string, additionalvalues ...any) (rbacv1.Role, error) {
	templateF, err := os.ReadFile(path)
	if err != nil {
		return rbacv1.Role{}, fmt.Errorf("failed to read role template file: %w", err)
	}
	values := templates.Values(templates.Renderoptions{
		Group:     gvr.Group,
		Version:   gvr.Version,
		Resource:  gvr.Resource,
		Namespace: nn.Namespace,
		Name:      nn.Name,
	})

	if len(additionalvalues)%2 != 0 {
		return rbacv1.Role{}, fmt.Errorf("additionalvalues must be in pairs: %w", err)
	}
	for i := 0; i < len(additionalvalues); i += 2 {
		key, ok := additionalvalues[i].(string)
		if !ok {
			return rbacv1.Role{}, fmt.Errorf("additionalvalues key must be a string: %w", err)
		}
		values[key] = additionalvalues[i+1]
	}

	template := templates.Template(string(templateF))
	dat, err := template.Render(values)
	if err != nil {
		return rbacv1.Role{}, err
	}

	s := json.NewYAMLSerializer(json.DefaultMetaFactory,
		clientsetscheme.Scheme,
		clientsetscheme.Scheme)

	res := rbacv1.Role{}
	_, _, err = s.Decode(dat, nil, &res)
	if err != nil {
		return rbacv1.Role{}, fmt.Errorf("failed to decode clusterrole: %w", err)
	}

	return res, err
}

func UninstallRole(ctx context.Context, opts UninstallOptions) error {
	return retry.Do(
		func() error {
			obj := rbacv1.Role{}
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
				opts.Log("Role successfully uninstalled",
					"name", obj.GetName(), "namespace", obj.GetNamespace())
			}

			return nil
		},
	)
}

func InstallRole(ctx context.Context, kube client.Client, obj *rbacv1.Role) error {
	return retry.Do(
		func() error {
			tmp := rbacv1.Role{}
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

func InitRole(resource string, opts types.NamespacedName) rbacv1.Role {
	kind := strings.ToLower(flect.Singularize(resource))
	role := rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "role",
				"app.kubernetes.io/instance":   "manager-role",
				"app.kubernetes.io/component":  "rbac",
				"app.kubernetes.io/created-by": kind,
				"app.kubernetes.io/part-of":    kind,
				"app.kubernetes.io/managed-by": "kustomize",
			},
		},
		Rules: []rbacv1.PolicyRule{},
	}

	return role
}
