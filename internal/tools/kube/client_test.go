//go:build integration
// +build integration

package kube

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	"github.com/krateoplatformops/plumbing/e2e"
	xenv "github.com/krateoplatformops/plumbing/env"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/kind"
)

var (
	testenv     env.Environment
	clusterName string
	namespace   string
)

func TestMain(m *testing.M) {
	xenv.SetTestMode(true)

	namespace = "demo-system"
	clusterName = "krateo"
	testenv = env.New()

	testenv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), clusterName),
		e2e.CreateNamespace(namespace),
		e2e.CreateNamespace("krateo-system"),
	).Finish(
		envfuncs.DeleteNamespace(namespace),
		envfuncs.DestroyCluster(clusterName),
	)

	os.Exit(testenv.Run(m))
}

func TestApply(t *testing.T) {

	os.Setenv("DEBUG", "1")

	f := features.New("Setup").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Assess("Apply", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		kube, err := client.New(cfg.Client().RESTConfig(), client.Options{})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		res := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "demo-system",
			},

			Spec: appsv1.DeploymentSpec{
				Replicas: func(i int32) *int32 { return &i }(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test-deployment",
					},
				},

				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "test-deployment",
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:  "test-container",
								Image: "nginx",
							},
						},
					},
				},
			},
		}

		rescp := res.DeepCopy()

		err = Apply(ctx, kube, res, ApplyOptions{})
		if err != nil {
			t.Fatalf("failed to apply clusterrole: %v", err)
		}

		// h := hasher.NewFNVObjectHash()
		// err = h.SumHash(res.Spec)
		// if err != nil {
		// 	t.Fatalf("failed to hash clusterrole: %v", err)
		// }

		err = Apply(ctx, kube, rescp, ApplyOptions{DryRun: []string{"All"}})
		if err != nil {
			t.Fatalf("failed to apply clusterrole: %v", err)
		}

		// hcp := hasher.NewFNVObjectHash()
		// err = hcp.SumHash(rescp.Spec)
		// if err != nil {
		// 	t.Fatalf("failed to hash clusterrole: %v", err)
		// }

		// hash := h.GetHash()
		// hashDryRun := hcp.GetHash()
		// if hash != hashDryRun {
		// 	t.Fatalf("hashes do not match: %s != %s", hash, hashDryRun)
		// }

		return ctx
	}).Feature()

	testenv.Test(t, f)
}
func TestUninstall(t *testing.T) {

	os.Setenv("DEBUG", "1")

	f := features.New("Uninstall").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Assess("Uninstall", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		kube, err := client.New(cfg.Client().RESTConfig(), client.Options{})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		res := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "demo-system",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: func(i int32) *int32 { return &i }(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test-deployment",
					},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "test-deployment",
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:  "test-container",
								Image: "nginx",
							},
						},
					},
				},
			},
		}

		rescp := res.DeepCopy()

		// Apply the resource first to ensure it exists
		err = Apply(ctx, kube, res, ApplyOptions{})
		if err != nil {
			t.Fatalf("failed to apply deployment: %v", err)
		}

		// Uninstall the resource
		err = Uninstall(ctx, kube, rescp, UninstallOptions{})
		if err != nil {
			t.Fatalf("failed to uninstall deployment: %v", err)
		}

		// Verify the resource no longer exists
		tmp := &appsv1.Deployment{}
		err = kube.Get(ctx, client.ObjectKeyFromObject(rescp), tmp)
		if err == nil {
			t.Fatalf("deployment still exists after uninstall")
		}
		if !apierrors.IsNotFound(err) {
			t.Fatalf("unexpected error while checking deployment existence: %v", err)
		}

		return ctx
	}).Feature()

	testenv.Test(t, f)
}
func TestGet(t *testing.T) {

	os.Setenv("DEBUG", "1")

	f := features.New("Get").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Assess("Get", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		kube, err := client.New(cfg.Client().RESTConfig(), client.Options{})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		res := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "demo-system",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: func(i int32) *int32 { return &i }(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test-deployment",
					},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "test-deployment",
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:  "test-container",
								Image: "nginx",
							},
						},
					},
				},
			},
		}

		rescp := res.DeepCopy()

		// Apply the resource first to ensure it exists
		err = Apply(ctx, kube, res, ApplyOptions{})
		if err != nil {
			t.Fatalf("failed to apply deployment: %v", err)
		}

		// Test the Get function
		err = Get(ctx, kube, rescp)
		if err != nil {
			t.Fatalf("failed to get deployment: %v", err)
		}

		// Verify the resource was retrieved successfully
		if rescp.GetName() != "test-deployment" || rescp.GetNamespace() != "demo-system" {
			t.Fatalf("retrieved deployment does not match expected values")
		}

		bres, _ := json.Marshal(res.Spec)
		brescp, _ := json.Marshal(rescp.Spec)
		if string(bres) != string(brescp) {
			t.Fatalf("retrieved deployment does not match expected values")
		}

		return ctx
	}).Feature()

	testenv.Test(t, f)
}
func TestUninstallWithGVK(t *testing.T) {

	os.Setenv("DEBUG", "1")

	f := features.New("UninstallWithGVK").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Assess("UninstallWithGVK", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		kube, err := client.New(cfg.Client().RESTConfig(), client.Options{})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		res := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "demo-system",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: func(i int32) *int32 { return &i }(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test-deployment",
					},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "test-deployment",
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:  "test-container",
								Image: "nginx",
							},
						},
					},
				},
			},
		}

		// Apply the resource first to ensure it exists
		err = Apply(ctx, kube, res, ApplyOptions{})
		if err != nil {
			t.Fatalf("failed to apply deployment: %v", err)
		}

		// Create an ObjectReference for the resource
		ref := v1.ObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-deployment",
			Namespace:  "demo-system",
		}

		// Uninstall the resource using UninstallWithGVK
		err = UninstallFromReference(ctx, kube, ref, UninstallOptions{})
		if err != nil {
			t.Fatalf("failed to uninstall deployment: %v", err)
		}

		// Verify the resource no longer exists
		tmp := &appsv1.Deployment{}
		err = kube.Get(ctx, client.ObjectKey{
			Name:      ref.Name,
			Namespace: ref.Namespace,
		}, tmp)
		if err == nil {
			t.Fatalf("deployment still exists after uninstall")
		}
		if !apierrors.IsNotFound(err) {
			t.Fatalf("unexpected error while checking deployment existence: %v", err)
		}

		return ctx
	}).Feature()

	testenv.Test(t, f)
}
func TestGetFromReference(t *testing.T) {

	os.Setenv("DEBUG", "1")

	f := features.New("GetFromReference").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Assess("GetFromReference", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		kube, err := client.New(cfg.Client().RESTConfig(), client.Options{})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		res := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "demo-system",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: func(i int32) *int32 { return &i }(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test-deployment",
					},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "test-deployment",
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:  "test-container",
								Image: "nginx",
							},
						},
					},
				},
			},
		}

		// Apply the resource first to ensure it exists
		err = Apply(ctx, kube, res, ApplyOptions{})
		if err != nil {
			t.Fatalf("failed to apply deployment: %v", err)
		}

		// Create an ObjectReference for the resource
		ref := v1.ObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-deployment",
			Namespace:  "demo-system",
		}

		// Test the GetFromReference function
		err = GetFromReference(ctx, kube, ref)
		if err != nil {
			t.Fatalf("failed to get deployment from reference: %v", err)
		}

		// Verify the resource was retrieved successfully
		tmp := &appsv1.Deployment{}
		err = kube.Get(ctx, client.ObjectKey{
			Name:      ref.Name,
			Namespace: ref.Namespace,
		}, tmp)
		if err != nil {
			t.Fatalf("failed to verify deployment existence: %v", err)
		}

		if tmp.GetName() != "test-deployment" || tmp.GetNamespace() != "demo-system" {
			t.Fatalf("retrieved deployment does not match expected values")
		}

		return ctx
	}).Feature()

	testenv.Test(t, f)
}
func TestCountResourcesWithGroup(t *testing.T) {

	os.Setenv("DEBUG", "1")

	f := features.New("CountRestResourcesWithGroup").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Assess("CountRestResourcesWithGroup", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		kube, err := client.New(cfg.Client().RESTConfig(), client.Options{})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		// Create a resource to count
		res := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "demo-system",
				Labels: map[string]string{
					"group": "test-group",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: func(i int32) *int32 { return &i }(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test-deployment",
					},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "test-deployment",
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:  "test-container",
								Image: "nginx",
							},
						},
					},
				},
			},
		}

		// Apply the resource
		err = Apply(ctx, kube, res, ApplyOptions{})
		if err != nil {
			t.Fatalf("failed to apply deployment: %v", err)
		}

		discovery := discovery.NewDiscoveryClientForConfigOrDie(cfg.Client().RESTConfig())

		// Count resources in the group
		_, count, err := CountRestResourcesWithGroup(ctx, kube, discovery, "apps")
		if err != nil {
			t.Fatalf("failed to count resources: %v", err)
		}

		// Verify the count is as expected
		if count == 0 {
			t.Fatalf("expected at least one resource in the group, got %d", count)
		}

		return ctx
	}).Feature()

	testenv.Test(t, f)
}
func TestCountRestDefinitionsWithGroup(t *testing.T) {

	os.Setenv("DEBUG", "1")

	f := features.New("CountRestDefinitionsWithGroup").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Assess("CountRestDefinitionsWithGroup", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		kube, err := client.New(cfg.Client().RESTConfig(), client.Options{})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		discovery := discovery.NewDiscoveryClientForConfigOrDie(cfg.Client().RESTConfig())

		// Test counting definitions in the "apps" group
		auth, count, err := CountRestDefinitionsWithGroup(ctx, kube, discovery, "apps")
		if err != nil {
			t.Fatalf("failed to count definitions: %v", err)
		}

		// Verify the count is as expected (apps group should have Deployment, ReplicaSet, etc.)
		if count == 0 {
			t.Fatalf("expected at least one definition in the apps group, got %d", count)
		}

		// Test with core group (empty string)
		auth2, count2, err := CountRestDefinitionsWithGroup(ctx, kube, discovery, "")
		if err != nil {
			t.Fatalf("failed to count definitions for core group: %v", err)
		}

		// Core group should have Pod, Service, ConfigMap, etc.
		if count2 == 0 {
			t.Fatalf("expected at least one definition in the core group, got %d", count2)
		}

		// Test with non-existent group
		auth3, count3, err := CountRestDefinitionsWithGroup(ctx, kube, discovery, "non-existent-group")
		if err != nil {
			t.Fatalf("failed to count definitions for non-existent group: %v", err)
		}

		// Non-existent group should have 0 definitions
		if count3 != 0 {
			t.Fatalf("expected 0 definitions in non-existent group, got %d", count3)
		}

		if auth3 {
			t.Fatalf("expected auth to be false for non-existent group")
		}

		// Verify auth is boolean (should not cause test failure, just verify type)
		_ = auth && auth2 && auth3

		return ctx
	}).Feature()

	testenv.Test(t, f)
}
