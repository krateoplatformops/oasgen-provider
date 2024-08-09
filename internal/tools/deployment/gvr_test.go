package deployment_test

import (
	"testing"

	"github.com/krateoplatformops/oasgen-provider/internal/tools/deployment"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestToGroupVersionResource(t *testing.T) {
	gvk := schema.GroupVersionKind{
		Group:   "petstore.swagger.io",
		Version: "v1alpha1",
		Kind:    "Pet",
	}
	expected := schema.GroupVersionResource{
		Group:    "petstore.swagger.io",
		Version:  "v1alpha1",
		Resource: "pets",
	}

	result := deployment.ToGroupVersionResource(gvk)

	if result != expected {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}
