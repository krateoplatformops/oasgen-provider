package deploy

import (
	"context"
	"fmt"
	"path"

	corev1 "k8s.io/api/core/v1"

	"github.com/krateoplatformops/oasgen-provider/internal/tools/configmap"
	crd "github.com/krateoplatformops/oasgen-provider/internal/tools/crd"
	deployment "github.com/krateoplatformops/oasgen-provider/internal/tools/deployment"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/rbactools"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	controllerResourceSuffix = "-controller"
	configmapResourceSuffix  = "-configmap"
)

type UndeployOptions struct {
	KubeClient     client.Client
	NamespacedName types.NamespacedName
	GVR            schema.GroupVersionResource
	RBACFolderPath string
	Log            func(msg string, keysAndValues ...any)
	SkipCRD        bool
}

type DeployOptions struct {
	GVR                    schema.GroupVersionResource
	KubeClient             client.Client
	NamespacedName         types.NamespacedName
	RBACFolderPath         string
	DeploymentTemplatePath string
	ConfigmapTemplatePath  string
	Log                    func(msg string, keysAndValues ...any)
}

func logError(log func(msg string, keysAndValues ...any), msg string, err error) {
	if log != nil {
		log(msg, "error", err)
	}
}

func createRBACResources(gvr schema.GroupVersionResource, rbacNSName types.NamespacedName, rbacFolderPath string) (corev1.ServiceAccount, rbacv1.ClusterRole, rbacv1.ClusterRoleBinding, rbacv1.Role, rbacv1.RoleBinding, error) {
	sa, err := rbactools.CreateServiceAccount(gvr, rbacNSName, path.Join(rbacFolderPath, "serviceaccount.yaml"))
	if err != nil {
		return corev1.ServiceAccount{}, rbacv1.ClusterRole{}, rbacv1.ClusterRoleBinding{}, rbacv1.Role{}, rbacv1.RoleBinding{}, err
	}

	clusterrole, err := rbactools.CreateClusterRole(gvr, rbacNSName, path.Join(rbacFolderPath, "clusterrole.yaml"))
	if err != nil {
		return corev1.ServiceAccount{}, rbacv1.ClusterRole{}, rbacv1.ClusterRoleBinding{}, rbacv1.Role{}, rbacv1.RoleBinding{}, err
	}

	clusterrolebinding, err := rbactools.CreateClusterRoleBinding(gvr, rbacNSName, path.Join(rbacFolderPath, "clusterrolebinding.yaml"), "serviceAccount", sa.Name, "saNamespace", sa.Namespace)
	if err != nil {
		return corev1.ServiceAccount{}, rbacv1.ClusterRole{}, rbacv1.ClusterRoleBinding{}, rbacv1.Role{}, rbacv1.RoleBinding{}, err
	}

	role, err := rbactools.CreateRole(gvr, rbacNSName, path.Join(rbacFolderPath, "role.yaml"))
	if err != nil {
		return corev1.ServiceAccount{}, rbacv1.ClusterRole{}, rbacv1.ClusterRoleBinding{}, rbacv1.Role{}, rbacv1.RoleBinding{}, err
	}

	rolebinding, err := rbactools.CreateRoleBinding(gvr, rbacNSName, path.Join(rbacFolderPath, "rolebinding.yaml"), "serviceAccount", sa.Name, "saNamespace", sa.Namespace)
	if err != nil {
		return corev1.ServiceAccount{}, rbacv1.ClusterRole{}, rbacv1.ClusterRoleBinding{}, rbacv1.Role{}, rbacv1.RoleBinding{}, err
	}

	return sa, clusterrole, clusterrolebinding, role, rolebinding, nil
}

func installRBACResources(ctx context.Context, kubeClient client.Client, clusterrole rbacv1.ClusterRole, clusterrolebinding rbacv1.ClusterRoleBinding, role rbacv1.Role, rolebinding rbacv1.RoleBinding, sa corev1.ServiceAccount, log func(msg string, keysAndValues ...any)) error {
	err := rbactools.InstallClusterRole(ctx, kubeClient, &clusterrole)
	if err != nil {
		logError(log, "Error installing clusterrole", err)
		return err
	}
	log("ClusterRole successfully installed", "name", clusterrole.Name, "namespace", clusterrole.Namespace)

	err = rbactools.InstallClusterRoleBinding(ctx, kubeClient, &clusterrolebinding)
	if err != nil {
		logError(log, "Error installing clusterrolebinding", err)
		return err
	}
	log("ClusterRoleBinding successfully installed", "name", clusterrolebinding.Name, "namespace", clusterrolebinding.Namespace)

	err = rbactools.InstallRole(ctx, kubeClient, &role)
	if err != nil {
		logError(log, "Error installing role", err)
		return err
	}
	log("Role successfully installed", "name", role.Name, "namespace", role.Namespace)

	err = rbactools.InstallRoleBinding(ctx, kubeClient, &rolebinding)
	if err != nil {
		logError(log, "Error installing rolebinding", err)
		return err
	}
	log("RoleBinding successfully installed", "name", rolebinding.Name, "namespace", rolebinding.Namespace)

	err = rbactools.InstallServiceAccount(ctx, kubeClient, &sa)
	if err != nil {
		logError(log, "Error installing serviceaccount", err)
		return err
	}
	log("ServiceAccount successfully installed", "name", sa.Name, "namespace", sa.Namespace)

	return nil
}

func uninstallRBACResources(ctx context.Context, kubeClient client.Client, clusterrole rbacv1.ClusterRole, clusterrolebinding rbacv1.ClusterRoleBinding, role rbacv1.Role, rolebinding rbacv1.RoleBinding, sa corev1.ServiceAccount, log func(msg string, keysAndValues ...any)) error {
	err := rbactools.UninstallClusterRole(ctx, rbactools.UninstallOptions{
		KubeClient: kubeClient,
		NamespacedName: types.NamespacedName{
			Namespace: clusterrole.Namespace,
			Name:      clusterrole.Name,
		},
		Log: log,
	})
	if err != nil {
		logError(log, "Error uninstalling clusterrole", err)
		return err
	}
	log("ClusterRole successfully uninstalled", "name", clusterrole.Name, "namespace", clusterrole.Namespace)

	err = rbactools.UninstallClusterRoleBinding(ctx, rbactools.UninstallOptions{
		KubeClient: kubeClient,
		NamespacedName: types.NamespacedName{
			Namespace: clusterrolebinding.Namespace,
			Name:      clusterrolebinding.Name,
		},
		Log: log,
	})
	if err != nil {
		logError(log, "Error uninstalling clusterrolebinding", err)
		return err
	}
	log("ClusterRoleBinding successfully uninstalled", "name", clusterrolebinding.Name, "namespace", clusterrolebinding.Namespace)

	err = rbactools.UninstallRole(ctx, rbactools.UninstallOptions{
		KubeClient: kubeClient,
		NamespacedName: types.NamespacedName{
			Namespace: role.Namespace,
			Name:      role.Name,
		},
		Log: log,
	})
	if err != nil {
		logError(log, "Error uninstalling role", err)
		return err
	}
	log("Role successfully uninstalled", "name", role.Name, "namespace", role.Namespace)

	err = rbactools.UninstallRoleBinding(ctx, rbactools.UninstallOptions{
		KubeClient: kubeClient,
		NamespacedName: types.NamespacedName{
			Namespace: rolebinding.Namespace,
			Name:      rolebinding.Name,
		},
		Log: log,
	})
	if err != nil {
		logError(log, "Error uninstalling rolebinding", err)
		return err
	}
	log("RoleBinding successfully uninstalled", "name", rolebinding.Name, "namespace", rolebinding.Namespace)

	err = rbactools.UninstallServiceAccount(ctx, rbactools.UninstallOptions{
		KubeClient: kubeClient,
		NamespacedName: types.NamespacedName{
			Namespace: sa.Namespace,
			Name:      sa.Name,
		},
		Log: log,
	})
	if err != nil {
		logError(log, "Error uninstalling serviceaccount", err)
		return err
	}
	log("ServiceAccount successfully uninstalled", "name", sa.Name, "namespace", sa.Namespace)

	return nil
}

func Deploy(ctx context.Context, kube client.Client, opts DeployOptions) (err error) {
	rbacNSName := types.NamespacedName{
		Namespace: opts.NamespacedName.Namespace,
		Name:      opts.NamespacedName.Name + controllerResourceSuffix,
	}

	sa, clusterrole, clusterrolebinding, role, rolebinding, err := createRBACResources(opts.GVR, rbacNSName, opts.RBACFolderPath)
	if err != nil {
		return err
	}

	err = installRBACResources(ctx, opts.KubeClient, clusterrole, clusterrolebinding, role, rolebinding, sa, opts.Log)
	if err != nil {
		return err
	}

	cmNSName := types.NamespacedName{
		Namespace: opts.NamespacedName.Namespace,
		Name:      opts.NamespacedName.Name + configmapResourceSuffix,
	}

	cm, err := configmap.CreateConfigmap(opts.GVR, cmNSName, opts.ConfigmapTemplatePath,
		"composition_controller_sa_name", sa.Name,
		"composition_controller_sa_namespace", sa.Namespace)
	if err != nil {
		return err
	}

	err = configmap.InstallConfigmap(ctx, opts.KubeClient, &cm)
	if err != nil {
		return fmt.Errorf("Error installing configmap: %v", err)
	}
	opts.Log("Configmap successfully installed", "gvr", opts.GVR.String(), "name", cm.Name, "namespace", cm.Namespace)

	deploymentNSName := types.NamespacedName{
		Namespace: opts.NamespacedName.Namespace,
		Name:      opts.NamespacedName.Name + controllerResourceSuffix,
	}
	dep, err := deployment.CreateDeployment(
		opts.GVR,
		deploymentNSName,
		opts.DeploymentTemplatePath,
		"serviceAccountName", sa.Name)
	if err != nil {
		return err
	}

	err = deployment.InstallDeployment(ctx, opts.KubeClient, &dep)
	if err != nil {
		return err
	}
	opts.Log("Deployment successfully installed", "gvr", opts.GVR.String(), "name", dep.Name, "namespace", dep.Namespace)

	return nil
}

func Undeploy(ctx context.Context, kube client.Client, opts UndeployOptions) error {
	if !opts.SkipCRD {
		err := crd.Uninstall(ctx, opts.KubeClient, opts.GVR.GroupResource())
		if err == nil && opts.Log != nil {
			opts.Log("CRD successfully uninstalled", "name", opts.GVR.GroupResource().String())
		}
		if err != nil {
			opts.Log("Error uninstalling CRD", "name", opts.GVR.GroupResource().String(), "error", err)
			return err
		}

		// labelreq, err := labels.NewRequirement(CompositionVersionLabel, selection.Equals, []string{opts.GVR.Version})
		// if err != nil {
		// 	return err
		// }
		// selector := labels.NewSelector().Add(*labelreq)

		// li, err := opts.DynamicClient.Resource(opts.GVR).List(ctx, metav1.ListOptions{
		// 	LabelSelector: selector.String(),
		// })
		// if err != nil {
		// 	return err
		// }

		// if len(li.Items) > 0 {
		// 	return fmt.Errorf("%v for %s", ErrCompositionStillExist, opts.GVR.String())
		// }
	}

	err := deployment.UninstallDeployment(ctx, deployment.UninstallOptions{
		KubeClient: opts.KubeClient,
		NamespacedName: types.NamespacedName{
			Namespace: opts.NamespacedName.Namespace,
			Name:      opts.NamespacedName.Name + controllerResourceSuffix,
		},
		Log: opts.Log,
	})
	if err != nil {
		return err
	}

	err = configmap.UninstallConfigmap(ctx, configmap.UninstallOptions{
		KubeClient: opts.KubeClient,
		NamespacedName: types.NamespacedName{
			Namespace: opts.NamespacedName.Namespace,
			Name:      opts.NamespacedName.Name + configmapResourceSuffix,
		},
	})
	if err != nil {
		return err
	}

	rbacNSName := types.NamespacedName{
		Namespace: opts.NamespacedName.Namespace,
		Name:      opts.NamespacedName.Name + controllerResourceSuffix,
	}

	sa, clusterrole, clusterrolebinding, role, rolebinding, err := createRBACResources(opts.GVR, rbacNSName, opts.RBACFolderPath)
	if err != nil {
		return err
	}

	err = uninstallRBACResources(ctx, opts.KubeClient, clusterrole, clusterrolebinding, role, rolebinding, sa, opts.Log)
	if err != nil {
		return err
	}

	return err
}
