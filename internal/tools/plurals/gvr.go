package plurals

import (
	"strings"

	"github.com/gobuffalo/flect"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func ToGroupVersionResource(gvk schema.GroupVersionKind) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(flect.Pluralize(gvk.Kind)),
	}
}
