package plurals

import (
	"testing"

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

	result := ToGroupVersionResource(gvk)

	if result != expected {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}
