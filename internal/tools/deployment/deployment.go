package deployment

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func LookupDeployment(ctx context.Context, kube client.Client, obj *appsv1.Deployment) (bool, bool, error) {
	err := kube.Get(ctx, client.ObjectKeyFromObject(obj), obj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, false, nil
		}

		return false, false, err
	}

	ready := obj.Spec.Replicas != nil && *obj.Spec.Replicas == obj.Status.ReadyReplicas

	return true, ready, nil
}

func RestartDeployment(ctx context.Context, kube client.Client, obj *appsv1.Deployment) error {
	patch := client.MergeFrom(obj.DeepCopy())

	// Set the annotation to trigger a rollout
	if obj.Spec.Template.Annotations == nil {
		obj.Spec.Template.Annotations = map[string]string{}
	}
	obj.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	// patch the deployment
	return kube.Patch(ctx, obj, patch)
}

func CleanFromRestartAnnotation(obj *appsv1.Deployment) {
	if obj.Spec.Template.Annotations != nil {
		delete(obj.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
	}
}
