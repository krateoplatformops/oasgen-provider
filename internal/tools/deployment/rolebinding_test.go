package deployment

// import (
// 	"context"
// 	"testing"

// 	"k8s.io/apimachinery/pkg/types"
// )

// func TestInstallRoleBinding(t *testing.T) {
// 	kube, err := setupKubeClient()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	obj := CreateRoleBinding(types.NamespacedName{
// 		Name:      "demo",
// 		Namespace: "default",
// 	})

// 	err = InstallRoleBinding(context.TODO(), kube, &obj)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }

// func TestUninstallRoleBinding(t *testing.T) {
// 	kube, err := setupKubeClient()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	err = UninstallRoleBinding(context.TODO(), UninstallOptions{
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
