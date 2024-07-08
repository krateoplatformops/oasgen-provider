package deployment

import (
	"context"
	"fmt"
	"strings"

	"github.com/gobuffalo/flect"
	definitionsv1alpha1 "github.com/matteogastaldello/swaggergen-provider/apis/restdefinitions/v1alpha1"
	"github.com/matteogastaldello/swaggergen-provider/internal/tools/generation"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UndeployOptions struct {
	KubeClient      client.Client
	NamespacedName  types.NamespacedName
	GVR             schema.GroupVersionResource
	Log             func(msg string, keysAndValues ...any)
	SecuritySchemes *orderedmap.Map[string, *v3.SecurityScheme]
}

func Undeploy(ctx context.Context, opts UndeployOptions) error {
	err := UninstallDeployment(ctx, UninstallOptions{
		KubeClient: opts.KubeClient,
		NamespacedName: types.NamespacedName{
			Namespace: opts.NamespacedName.Namespace,
			Name:      fmt.Sprintf("%s-%s-controller", opts.GVR.Resource, opts.GVR.Version),
		},
		Log: opts.Log,
	})
	if err != nil {
		return err
	}

	err = UninstallRoleBinding(ctx, UninstallOptions{
		KubeClient:     opts.KubeClient,
		NamespacedName: opts.NamespacedName,
		Log:            opts.Log,
	})
	if err != nil {
		return err
	}
	if opts.Log != nil {
		opts.Log("RoleBinding successfully uninstalled", "name", opts.NamespacedName.String())
	}

	err = UninstallRole(ctx, UninstallOptions{
		KubeClient:     opts.KubeClient,
		NamespacedName: opts.NamespacedName,
		Log:            opts.Log,
	})
	if err != nil {
		return err
	}
	if opts.Log != nil {
		opts.Log("Role successfully uninstalled", "name", opts.NamespacedName.String())
	}

	err = UninstallServiceAccount(ctx, UninstallOptions{
		KubeClient:     opts.KubeClient,
		NamespacedName: opts.NamespacedName,
		Log:            opts.Log,
	})
	if err != nil {
		return err
	}
	if opts.Log != nil {
		opts.Log("ServiceAccount successfully uninstalled", "name", opts.NamespacedName.String())
	}

	err = UninstallCRD(ctx, opts.KubeClient, opts.GVR.GroupResource())
	if err == nil {
		if opts.Log != nil {
			opts.Log("CRD successfully uninstalled", "name", opts.GVR.GroupResource().String())
		}
	}
	if err != nil {
		if opts.Log != nil {
			opts.Log("failed to uninstall CRD", "name", opts.GVR.GroupResource().String(), "error", err)
		}
	}

	for secSchemaPair := opts.SecuritySchemes.First(); secSchemaPair != nil; secSchemaPair = secSchemaPair.Next() {
		authSchemaName, err := generation.GenerateAuthSchemaName(secSchemaPair.Value())
		if err != nil {
			continue
		}

		if opts.Log != nil {
			opts.Log("uninstalling CRD", "name", authSchemaName, "Group", opts.GVR.Group)
		}

		err = UninstallCRD(ctx, opts.KubeClient, schema.GroupResource{
			Group:    opts.GVR.Group,
			Resource: flect.Pluralize(strings.ToLower(authSchemaName)),
		})
		if err != nil {
			if opts.Log != nil {
				opts.Log("failed to uninstall CRD", "name", authSchemaName, "error", err)
			}
		}
		if err == nil {
			if opts.Log != nil {
				opts.Log("CRD successfully uninstalled", "name", opts.GVR.GroupResource().String())
			}
		}
	}

	return err
}

type DeployOptions struct {
	KubeClient      client.Client
	NamespacedName  types.NamespacedName
	Spec            *definitionsv1alpha1.RestDefinitionSpec
	ResourceVersion string
	Role            v1.Role
	Log             func(msg string, keysAndValues ...any)
}

func Deploy(ctx context.Context, opts DeployOptions) error {

	sa := CreateServiceAccount(opts.NamespacedName)
	if err := InstallServiceAccount(ctx, opts.KubeClient, &sa); err != nil {
		return fmt.Errorf("failed to install service account: %w", err)
	}
	if opts.Log != nil {
		opts.Log("ServiceAccount successfully installed",
			"name", sa.Name, "namespace", sa.Namespace)
	}

	gvr := ToGroupVersionResource(schema.GroupVersionKind{
		Group:   opts.Spec.ResourceGroup,
		Version: opts.ResourceVersion,
		Kind:    opts.Spec.Resource.Kind,
	})

	by, _ := yaml.Marshal(opts.Role)
	fmt.Println(string(by))

	if err := InstallRole(ctx, opts.KubeClient, &opts.Role); err != nil {
		return fmt.Errorf("failed to install role: %w", err)
	}
	if opts.Log != nil {
		opts.Log("Role successfully installed",
			"gvr", gvr.String(), "name", opts.Role.Name, "namespace", opts.Role.Namespace)
	}

	rb := CreateRoleBinding(opts.NamespacedName)
	if err := InstallRoleBinding(ctx, opts.KubeClient, &rb); err != nil {
		return err
	}
	if opts.Log != nil {
		opts.Log("RoleBinding successfully installed",
			"gvr", gvr.String(), "name", rb.Name, "namespace", rb.Namespace)
	}

	// cr := CreateClusterRole(opts.NamespacedName)
	// if err := InstallClusterRole(ctx, opts.KubeClient, &cr); err != nil {
	// 	return err
	// }
	// if opts.Log != nil {
	// 	opts.Log("ClusterRole successfully installed",
	// 		"gvr", gvr.String(), "name", cr.Name, "namespace", cr.Namespace)
	// }

	// crb := CreateClusterRoleBinding(opts.NamespacedName)
	// if err := InstallClusterRoleBinding(ctx, opts.KubeClient, &crb); err != nil {
	// 	return err
	// }
	// if opts.Log != nil {
	// 	opts.Log("ClusterRoleBinding successfully installed",
	// 		"gvr", gvr.String(), "name", crb.Name, "namespace", crb.Namespace)
	// }
	dep, err := CreateDeployment(gvr, opts.NamespacedName)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}
	// b, _ := yaml.Marshal(dep)
	// fmt.Println(string(b))

	err = InstallDeployment(ctx, opts.KubeClient, &dep)
	if err != nil {
		return fmt.Errorf("failed to install deployment: %w", err)
	}
	if opts.Log != nil {
		opts.Log("Deployment successfully installed",
			"gvr", gvr.String(), "name", dep.Name, "namespace", dep.Namespace)
	}

	return nil
}
