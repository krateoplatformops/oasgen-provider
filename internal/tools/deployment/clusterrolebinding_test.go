package deployment

// import (
// 	"context"
// 	"testing"

// 	"k8s.io/apimachinery/pkg/types"
// )

// func TestInstallClusterRoleBinding(t *testing.T) {
// 	kube, err := setupKubeClient()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	obj := CreateClusterRoleBinding(types.NamespacedName{
// 		Name:      "demo",
// 		Namespace: "default",
// 	})

// 	err = InstallClusterRoleBinding(context.TODO(), kube, &obj)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }

// func TestUninstallClusterRoleBinding(t *testing.T) {
// 	kube, err := setupKubeClient()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	err = UninstallClusterRoleBinding(context.TODO(), UninstallOptions{
// 		KubeClient: kube,
// 		NamespacedName: types.NamespacedName{
// 			Name:      "demo",
// 			Namespace: "default",
// 		},
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }
