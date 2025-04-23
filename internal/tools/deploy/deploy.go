package deploy

import (
	"context"
	"fmt"
	"path"

	kubecli "github.com/krateoplatformops/oasgen-provider/internal/tools/kube"
	templates "github.com/krateoplatformops/oasgen-provider/internal/tools/objects"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	crd "github.com/krateoplatformops/oasgen-provider/internal/tools/crd"
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
	AuthenticationGVRs     []schema.GroupVersionResource
	KubeClient             client.Client
	NamespacedName         types.NamespacedName
	GVR                    schema.GroupVersionResource
	RBACFolderPath         string
	Log                    func(msg string, keysAndValues ...any)
	SkipCRD                bool
	DeploymentTemplatePath string
	ConfigmapTemplatePath  string
}

type DeployOptions struct {
	GVR                    schema.GroupVersionResource
	AuthenticationGVRs     []schema.GroupVersionResource
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

func createRBACResources(gvr schema.GroupVersionResource, rbacNSName types.NamespacedName, authenticationsGVRs []schema.GroupVersionResource, rbacFolderPath string) (corev1.ServiceAccount, rbacv1.ClusterRole, rbacv1.ClusterRoleBinding, rbacv1.Role, rbacv1.RoleBinding, error) {
	rbacNSName = types.NamespacedName{
		Namespace: rbacNSName.Namespace,
		Name:      rbacNSName.Name + controllerResourceSuffix,
	}

	sa := corev1.ServiceAccount{}
	err := templates.CreateK8sObject(&sa, gvr, rbacNSName, path.Join(rbacFolderPath, "serviceaccount.yaml"))
	if err != nil {
		return corev1.ServiceAccount{}, rbacv1.ClusterRole{}, rbacv1.ClusterRoleBinding{}, rbacv1.Role{}, rbacv1.RoleBinding{}, err
	}

	clusterrole := rbacv1.ClusterRole{}
	err = templates.CreateK8sObject(&clusterrole, gvr, rbacNSName, path.Join(rbacFolderPath, "clusterrole.yaml"))
	if err != nil {
		return corev1.ServiceAccount{}, rbacv1.ClusterRole{}, rbacv1.ClusterRoleBinding{}, rbacv1.Role{}, rbacv1.RoleBinding{}, err
	}

	clusterrolebinding := rbacv1.ClusterRoleBinding{}
	err = templates.CreateK8sObject(&clusterrolebinding, gvr, rbacNSName, path.Join(rbacFolderPath, "clusterrolebinding.yaml"), "serviceAccount", sa.Name, "saNamespace", sa.Namespace)
	if err != nil {
		return corev1.ServiceAccount{}, rbacv1.ClusterRole{}, rbacv1.ClusterRoleBinding{}, rbacv1.Role{}, rbacv1.RoleBinding{}, err
	}

	var authentications []string
	for _, crd := range authenticationsGVRs {
		authentications = append(authentications, crd.Resource)
	}

	role := rbacv1.Role{}
	err = templates.CreateK8sObject(&role, gvr, rbacNSName, path.Join(rbacFolderPath, "role.yaml"), "authentications", authentications)
	if err != nil {
		return corev1.ServiceAccount{}, rbacv1.ClusterRole{}, rbacv1.ClusterRoleBinding{}, rbacv1.Role{}, rbacv1.RoleBinding{}, err
	}

	rolebinding := rbacv1.RoleBinding{}
	err = templates.CreateK8sObject(&rolebinding, gvr, rbacNSName, path.Join(rbacFolderPath, "rolebinding.yaml"), "serviceAccount", sa.Name, "saNamespace", sa.Namespace)
	if err != nil {
		return corev1.ServiceAccount{}, rbacv1.ClusterRole{}, rbacv1.ClusterRoleBinding{}, rbacv1.Role{}, rbacv1.RoleBinding{}, err
	}

	return sa, clusterrole, clusterrolebinding, role, rolebinding, nil
}

func installRBACResources(ctx context.Context, kubeClient client.Client, clusterrole rbacv1.ClusterRole, clusterrolebinding rbacv1.ClusterRoleBinding, role rbacv1.Role, rolebinding rbacv1.RoleBinding, sa corev1.ServiceAccount, log func(msg string, keysAndValues ...any), applyOpts kubecli.ApplyOptions) error {

	err := kubecli.Apply(ctx, kubeClient, &clusterrole, applyOpts)
	if err != nil {
		logError(log, "Error installing clusterrole", err)
		return err
	}
	log("ClusterRole successfully installed", "name", clusterrole.Name, "namespace", clusterrole.Namespace)

	err = kubecli.Apply(ctx, kubeClient, &clusterrolebinding, applyOpts)
	if err != nil {
		logError(log, "Error installing clusterrolebinding", err)
		return err
	}
	log("ClusterRoleBinding successfully installed", "name", clusterrolebinding.Name, "namespace", clusterrolebinding.Namespace)

	err = kubecli.Apply(ctx, kubeClient, &role, applyOpts)
	if err != nil {
		logError(log, "Error installing role", err)
		return err
	}
	log("Role successfully installed", "name", role.Name, "namespace", role.Namespace)

	err = kubecli.Apply(ctx, kubeClient, &rolebinding, applyOpts)
	if err != nil {
		logError(log, "Error installing rolebinding", err)
		return err
	}
	log("RoleBinding successfully installed", "name", rolebinding.Name, "namespace", rolebinding.Namespace)

	err = kubecli.Apply(ctx, kubeClient, &sa, applyOpts)
	if err != nil {
		logError(log, "Error installing serviceaccount", err)
		return err
	}
	log("ServiceAccount successfully installed", "name", sa.Name, "namespace", sa.Namespace)

	return nil
}

func uninstallRBACResources(ctx context.Context, kubeClient client.Client, clusterrole rbacv1.ClusterRole, clusterrolebinding rbacv1.ClusterRoleBinding, role rbacv1.Role, rolebinding rbacv1.RoleBinding, sa corev1.ServiceAccount, log func(msg string, keysAndValues ...any)) error {
	err := kubecli.Uninstall(ctx, kubeClient, &clusterrole, kubecli.UninstallOptions{})
	if err != nil {
		logError(log, "Error uninstalling clusterrole", err)
		return err
	}
	log("ClusterRole successfully uninstalled", "name", clusterrole.Name, "namespace", clusterrole.Namespace)

	err = kubecli.Uninstall(ctx, kubeClient, &clusterrolebinding, kubecli.UninstallOptions{})
	if err != nil {
		logError(log, "Error uninstalling clusterrolebinding", err)
		return err
	}
	log("ClusterRoleBinding successfully uninstalled", "name", clusterrolebinding.Name, "namespace", clusterrolebinding.Namespace)

	err = kubecli.Uninstall(ctx, kubeClient, &role, kubecli.UninstallOptions{})
	if err != nil {
		logError(log, "Error uninstalling role", err)
		return err
	}
	log("Role successfully uninstalled", "name", role.Name, "namespace", role.Namespace)

	err = kubecli.Uninstall(ctx, kubeClient, &rolebinding, kubecli.UninstallOptions{})
	if err != nil {
		logError(log, "Error uninstalling rolebinding", err)
		return err
	}
	log("RoleBinding successfully uninstalled", "name", rolebinding.Name, "namespace", rolebinding.Namespace)

	err = kubecli.Uninstall(ctx, kubeClient, &sa, kubecli.UninstallOptions{})
	if err != nil {
		logError(log, "Error uninstalling serviceaccount", err)
		return err
	}
	log("ServiceAccount successfully uninstalled", "name", sa.Name, "namespace", sa.Namespace)

	return nil
}

func Deploy(ctx context.Context, kube client.Client, opts DeployOptions) (err error) {
	if opts.Log == nil {
		return fmt.Errorf("log function is required")
	}

	sa, clusterrole, clusterrolebinding, role, rolebinding, err := createRBACResources(opts.GVR, opts.NamespacedName, opts.AuthenticationGVRs, opts.RBACFolderPath)
	if err != nil {
		opts.Log("Error creating RBAC resources", "error", err)
		return err
	}
	applyOpts := kubecli.ApplyOptions{}
	err = installRBACResources(ctx, opts.KubeClient, clusterrole, clusterrolebinding, role, rolebinding, sa, opts.Log, applyOpts)
	if err != nil {
		opts.Log("Error installing RBAC resources", "error", err)
		return err
	}

	cm := corev1.ConfigMap{}
	err = templates.CreateK8sObject(&cm, opts.GVR, opts.NamespacedName, opts.ConfigmapTemplatePath,
		"composition_controller_sa_name", sa.Name,
		"composition_controller_sa_namespace", sa.Namespace)
	if err != nil {
		opts.Log("Error creating configmap object", "error", err)
		return err
	}

	err = kubecli.Apply(ctx, opts.KubeClient, &cm, applyOpts)
	if err != nil {
		opts.Log("Error installing configmap", "name", cm.Name, "namespace", cm.Namespace, "error", err)
		return fmt.Errorf("error installing configmap: %v", err)
	}
	opts.Log("Configmap successfully installed", "gvr", opts.GVR.String(), "name", cm.Name, "namespace", cm.Namespace)

	dep := appsv1.Deployment{}
	err = templates.CreateK8sObject(
		&dep,
		opts.GVR,
		opts.NamespacedName,
		opts.DeploymentTemplatePath,
		"serviceAccountName", sa.Name)
	if err != nil {
		opts.Log("Error creating deployment object", "error", err)
		return err
	}

	err = kubecli.Apply(ctx, opts.KubeClient, &dep, applyOpts)
	if err != nil {
		opts.Log("Error installing deployment", "name", dep.Name, "namespace", dep.Namespace, "error", err)
		return fmt.Errorf("error installing deployment: %v", err)
	}
	opts.Log("Deployment successfully installed", "gvr", opts.GVR.String(), "name", dep.Name, "namespace", dep.Namespace)

	return nil
}

func Undeploy(ctx context.Context, kube client.Client, opts UndeployOptions) error {
	if opts.Log == nil {
		return fmt.Errorf("log function is required")
	}

	if !opts.SkipCRD {
		err := crd.Uninstall(ctx, opts.KubeClient, opts.GVR.GroupResource())
		if err == nil && opts.Log != nil {
			opts.Log("CRD successfully uninstalled", "name", opts.GVR.GroupResource().String())
		}
		if err != nil {
			opts.Log("Error uninstalling CRD", "name", opts.GVR.GroupResource().String(), "error", err)
			return err
		}
	}

	sa, clusterrole, clusterrolebinding, role, rolebinding, err := createRBACResources(opts.GVR, opts.NamespacedName, opts.AuthenticationGVRs, opts.RBACFolderPath)
	if err != nil {
		opts.Log("Error creating RBAC resources", "error", err)
		return err
	}

	dep := appsv1.Deployment{}
	err = templates.CreateK8sObject(
		&dep,
		opts.GVR,
		opts.NamespacedName,
		opts.DeploymentTemplatePath,
		"serviceAccountName", sa.Name)
	if err != nil {
		opts.Log("Error creating deployment object", "error", err)
		return err
	}
	err = kubecli.Uninstall(ctx, opts.KubeClient, &dep, kubecli.UninstallOptions{})
	if err != nil {
		opts.Log("Error uninstalling deployment", "name", dep.Name, "namespace", dep.Namespace, "error", err)
		return fmt.Errorf("error uninstalling deployment: %v", err)
	}

	cm := corev1.ConfigMap{}
	err = templates.CreateK8sObject(&cm, opts.GVR, opts.NamespacedName, opts.ConfigmapTemplatePath,
		"composition_controller_sa_name", sa.Name,
		"composition_controller_sa_namespace", sa.Namespace)
	if err != nil {
		opts.Log("Error creating configmap object", "error", err)
		return err
	}

	err = kubecli.Uninstall(ctx, opts.KubeClient, &cm, kubecli.UninstallOptions{})
	if err != nil {
		opts.Log("Error uninstalling configmap", "name", cm.Name, "namespace", cm.Namespace, "error", err)
		return err
	}

	err = uninstallRBACResources(ctx, opts.KubeClient, clusterrole, clusterrolebinding, role, rolebinding, sa, opts.Log)
	if err != nil {
		opts.Log("Error uninstalling RBAC resources", "error", err)
		return err
	}

	opts.Log("RBAC resources successfully uninstalled", "gvr", opts.GVR.String(), "name", opts.NamespacedName.Name, "namespace", opts.NamespacedName.Namespace)

	return err
}
