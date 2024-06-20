// Package v1alpha1 contains API Schema definitions v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=swaggergen.krateo.io
// +versionName=v1alpha1
package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "swaggergen.krateo.io"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)

var (
	DefinitionKind             = reflect.TypeOf(Definition{}).Name()
	DefinitionGroupKind        = schema.GroupKind{Group: Group, Kind: DefinitionKind}.String()
	DefinitionKindAPIVersion   = DefinitionKind + "." + SchemeGroupVersion.String()
	DefinitionGroupVersionKind = SchemeGroupVersion.WithKind(DefinitionKind)
)

func init() {
	SchemeBuilder.Register(&Definition{}, &DefinitionList{})
}
