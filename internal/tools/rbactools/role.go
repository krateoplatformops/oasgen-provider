package rbactools

import (
	"context"
	"fmt"
	"strings"

	"github.com/avast/retry-go"
	"github.com/gobuffalo/flect"
	"sigs.k8s.io/yaml"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

func PopulateRole(resource schema.GroupVersionKind, role *rbacv1.Role) {

	res := strings.ToLower(flect.Pluralize(resource.Kind))

	role.Rules = append(role.Rules, rbacv1.PolicyRule{
		APIGroups: []string{resource.Group},
		Resources: []string{res, fmt.Sprintf("%s/status", res)},
		Verbs:     []string{"*"},
	})
}

func InitRole(opts types.NamespacedName) (rbacv1.Role, error) {

	role := rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"swaggergen.krateo.io"},
				Resources: []string{"restdefinitions", "restdefinitions/status"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get"},
			},
		},
	}

	return role, nil
}

// func createRole(pkg *chartfs.ChartFS, resource string, opts types.NamespacedName) (rbacv1.Role, error) {
// 	rend, err := RenderChartTemplates(pkg)
// 	if err != nil {
// 		return rbacv1.Role{}, err
// 	}

// 	if len(rend) == 0 {
// 		return rbacv1.Role{}, fmt.Errorf("no entries in manifest")
// 	}

// 	role := rbacv1.Role{
// 		TypeMeta: metav1.TypeMeta{
// 			APIVersion: "rbac.authorization.k8s.io/v1",
// 			Kind:       "Role",
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      opts.Name,
// 			Namespace: opts.Namespace,
// 		},
// 		Rules: []rbacv1.PolicyRule{
// 			{
// 				APIGroups: []string{"core.krateo.io"},
// 				Resources: []string{"definitions", "definitions/status"},
// 				Verbs:     []string{"*"},
// 			},
// 			{
// 				APIGroups: []string{"composition.krateo.io"},
// 				Resources: []string{resource, fmt.Sprintf("%s/status", resource)},
// 				Verbs:     []string{"*"},
// 			},
// 		},
// 	}

// 	pols := map[string]map[string]bool{
// 		"": {"secrets": true, "namespaces": true},
// 	}

// 	mdr := utilyaml.NewYAMLReader(bufio.NewReader(strings.NewReader(rend)))
// 	for {
// 		nfo, err := createPolicyInfo(mdr)
// 		if err != nil {
// 			if errors.Is(err, io.EOF) {
// 				break
// 			}
// 			return role, err
// 		}
// 		if len(nfo.resource) == 0 {
// 			continue
// 		}

// 		lst, ok := pols[nfo.group]
// 		if !ok {
// 			lst = map[string]bool{}
// 			pols[nfo.group] = lst
// 		}
// 		if _, exists := lst[nfo.resource]; !exists {
// 			lst[nfo.resource] = true
// 		}
// 	}

// 	for grp, res := range pols {
// 		keys := make([]string, 0, len(res))
// 		for k := range res {
// 			keys = append(keys, k)
// 		}

// 		role.Rules = append(role.Rules, rbacv1.PolicyRule{
// 			APIGroups: []string{grp},
// 			Resources: keys,
// 			Verbs:     []string{"*"},
// 		})
// 	}

// 	return role, nil
// }

func createPolicyInfo(dec *utilyaml.YAMLReader) (nfo policytInfo, err error) {
	buf, err := dec.Read()
	if err != nil {
		return nfo, err
	}

	tm := metav1.TypeMeta{}
	if err := yaml.Unmarshal(buf, &tm); err != nil {
		return nfo, err
	}
	if tm.Kind == "" {
		return nfo, err
	}

	gv, err := schema.ParseGroupVersion(tm.APIVersion)
	if err != nil {
		return nfo, err
	}

	nfo.group = gv.Group
	nfo.resource = strings.ToLower(flect.Pluralize(tm.Kind))
	return nfo, err
}

type policytInfo struct {
	group    string
	resource string
}
