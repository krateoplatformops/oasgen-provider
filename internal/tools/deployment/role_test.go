// //go:build integration
// // +build integration
package deployment

// import (
// 	"context"
// 	"fmt"
// 	"testing"

// 	"github.com/krateoplatformops/core-provider/apis/definitions/v1alpha1"
// 	"github.com/krateoplatformops/core-provider/internal/tools/chartfs"
// 	rbacv1 "k8s.io/api/rbac/v1"
// 	"k8s.io/apimachinery/pkg/types"
// 	"sigs.k8s.io/yaml"
// )

// const (
// 	testChartUrl = "https://github.com/krateoplatformops/postgresql-demo-chart/releases/download/12.8.3/postgresql-12.8.3.tgz"
// )

// func TestInstallRole(t *testing.T) {
// 	kube, err := setupKubeClient()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	obj, err := createRoleFromURL()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	err = InstallRole(context.TODO(), kube, &obj)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }

// func TestUninstallRole(t *testing.T) {
// 	kube, err := setupKubeClient()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	err = UninstallRole(context.TODO(), UninstallOptions{
// 		KubeClient: kube,
// 		NamespacedName: types.NamespacedName{
// 			Name:      "postgresqls-controller",
// 			Namespace: "default",
// 		},
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }

// func TestCreateRoleFromTGZ(t *testing.T) {
// 	obj, err := createRoleFromURL()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	dat, err := yaml.Marshal(obj)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	fmt.Println(string(dat))
// }

// func createRoleFromURL() (rbacv1.Role, error) {
// 	pkg, err := chartfs.ForSpec(&v1alpha1.ChartInfo{
// 		Url: testChartUrl,
// 	})
// 	if err != nil {
// 		return rbacv1.Role{}, err
// 	}

// 	gvr, err := GroupVersionResource(pkg)
// 	if err != nil {
// 		return rbacv1.Role{}, err
// 	}

// 	return CreateRole(pkg, gvr.Resource, types.NamespacedName{
// 		Name:      fmt.Sprintf("%s-controller", gvr.Resource),
// 		Namespace: "default",
// 	})
// }
