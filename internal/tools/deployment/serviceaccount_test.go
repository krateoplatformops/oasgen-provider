package deployment

// import (
// 	"context"
// 	"testing"

// 	"k8s.io/apimachinery/pkg/types"
// )

// func TestInstallServiceAccount(t *testing.T) {
// 	kube, err := setupKubeClient()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	obj := CreateServiceAccount(types.NamespacedName{
// 		Name:      "demo",
// 		Namespace: "default",
// 	})

// 	err = InstallServiceAccount(context.TODO(), kube, &obj)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }

// func TestUninstallServiceAccount(t *testing.T) {
// 	kube, err := setupKubeClient()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	err = UninstallServiceAccount(context.TODO(), UninstallOptions{
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
