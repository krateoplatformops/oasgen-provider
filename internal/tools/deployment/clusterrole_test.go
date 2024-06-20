package deployment

// import (
// 	"context"
// 	"testing"

// 	"k8s.io/apimachinery/pkg/types"
// )

// func TestInstallClusterRole(t *testing.T) {
// 	kube, err := setupKubeClient()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	obj := CreateClusterRole(types.NamespacedName{
// 		Name:      "demo",
// 		Namespace: "default",
// 	})

// 	err = InstallClusterRole(context.TODO(), kube, &obj)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }

// func TestUninstallClusterRole(t *testing.T) {
// 	kube, err := setupKubeClient()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	err = UninstallClusterRole(context.TODO(), UninstallOptions{
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
