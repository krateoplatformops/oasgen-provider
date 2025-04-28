package deployment

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLookupDeployment(t *testing.T) {
	// Setup the scheme for the fake client
	s := scheme.Scheme
	_ = appsv1.AddToScheme(s)

	tests := []struct {
		name          string
		deployment    *appsv1.Deployment
		clientObjects []runtime.Object
		expectedFound bool
		expectedReady bool
		expectError   bool
	}{
		{
			name: "Deployment not found",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
			},
			clientObjects: nil,
			expectedFound: false,
			expectedReady: false,
			expectError:   false,
		},
		{
			name: "Deployment found but not ready",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(3),
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 1,
				},
			},
			clientObjects: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-deployment",
						Namespace: "default",
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: int32Ptr(3),
					},
					Status: appsv1.DeploymentStatus{
						ReadyReplicas: 1,
					},
				},
			},
			expectedFound: true,
			expectedReady: false,
			expectError:   false,
		},
		{
			name: "Deployment found and ready",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(3),
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 3,
				},
			},
			clientObjects: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-deployment",
						Namespace: "default",
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: int32Ptr(3),
					},
					Status: appsv1.DeploymentStatus{
						ReadyReplicas: 3,
					},
				},
			},
			expectedFound: true,
			expectedReady: true,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake client with the provided objects
			fakeClient := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(tt.clientObjects...).Build()

			found, ready, err := LookupDeployment(context.TODO(), fakeClient, tt.deployment)

			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}
			if found != tt.expectedFound {
				t.Errorf("expected found: %v, got: %v", tt.expectedFound, found)
			}
			if ready != tt.expectedReady {
				t.Errorf("expected ready: %v, got: %v", tt.expectedReady, ready)
			}
		})
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}

func TestRestartDeployment(t *testing.T) {
	// Setup
	ctx := context.TODO()
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	deploymentObj := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(deploymentObj).Build()

	// Act
	err := RestartDeployment(ctx, client, deploymentObj)

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, deploymentObj.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
	_, err = time.Parse(time.RFC3339, deploymentObj.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"])
	assert.NoError(t, err)
}

func TestCleanFromRestartAnnotation(t *testing.T) {
	// Setup
	deploymentObj := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubectl.kubernetes.io/restartedAt": time.Now().Format(time.RFC3339),
					},
				},
			},
		},
	}

	// Act
	CleanFromRestartAnnotation(deploymentObj)

	// Assert
	assert.NotContains(t, deploymentObj.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
}
