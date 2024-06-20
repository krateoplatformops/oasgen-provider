// //go:build integration
// // +build integration

package deployment

// import (
// 	"context"
// 	"testing"

// 	"github.com/krateoplatformops/core-provider/apis/definitions/v1alpha1"
// 	"k8s.io/apimachinery/pkg/runtime/schema"
// 	"k8s.io/apimachinery/pkg/types"
// )

// func TestDeploy(t *testing.T) {
// 	nfo := &v1alpha1.ChartInfo{
// 		Url:     "oci://registry-1.docker.io/bitnamicharts/postgresql",
// 		Version: "12.8.3",
// 	}

// 	kube, err := setupKubeClient()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	err = Deploy(context.TODO(), DeployOptions{
// 		KubeClient: kube,
// 		Spec:       nfo,
// 		NamespacedName: types.NamespacedName{
// 			Namespace: "default",
// 			Name:      "postgresql-repo",
// 		},
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }

// func TestUndeploy(t *testing.T) {
// 	kube, err := setupKubeClient()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	gvr := schema.GroupVersionResource{
// 		Group:    "",
// 		Version:  "v12-8-3",
// 		Resource: "postgresqls",
// 	}

// 	err = Undeploy(context.TODO(), UndeployOptions{
// 		KubeClient: kube,
// 		GVR:        gvr,
// 		NamespacedName: types.NamespacedName{
// 			Namespace: "default",
// 			Name:      "postgresql-repo",
// 		},
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }
