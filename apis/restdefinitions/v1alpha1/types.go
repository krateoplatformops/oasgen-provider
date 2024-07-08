package v1alpha1

import (
	rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VerbsDescription struct {
	// Name of the action to perform when this api is called [create, update, get, delete, findby]
	// +kubebuilder:validation:Enum=create;update;get;delete;findby
	// +immutable
	// +required
	Action string `json:"action"`
	// Method: the http method to use [GET, POST, PUT, DELETE, PATCH]
	// +kubebuilder:validation:Enum=GET;POST;PUT;DELETE;PATCH
	// +immutable
	// +required
	Method string `json:"method"`
	// Path: the path to the api - has to be the same path as the one in the swagger file you are referencing
	// +immutable
	// +required
	Path string `json:"path"`
	// AltFieldMapping: the alternative mapping of the fields to use in the request
	// +optional
	AltFieldMapping map[string]string `json:"altFieldMapping,omitempty"`
}

type GVK struct {
	// Group: the group of the resource
	// +optional
	Group string `json:"group,omitempty"`

	// Version: the version of the resource
	// +optional
	Version string `json:"version,omitempty"`

	// Kind: the kind of the resource
	// +optional
	Kind string `json:"kind,omitempty"`
}

type ReferenceInfo struct {
	// Field: the field to use as reference - represents the id of the resource
	// +optional
	Field string `json:"field,omitempty"`

	// GVK: the group, version, kind of the resource
	// +optional
	GroupVersionKind GVK `json:"groupVersionKind,omitempty"`
}

type Resource struct {
	// Name: the name of the resource to manage
	// +immutable
	Kind string `json:"kind"`

	// OwnerRefs: Set GVK to resources which the defined resource have ownerReference.
	// +optional
	OwnerRefs []ReferenceInfo `json:"ownerRefs,omitempty"`

	// VerbsDescription: the list of verbs to use on this resource
	// +optional
	VerbsDescription []VerbsDescription `json:"verbsDescription"`
	// Identifiers: the list of fields to use as identifiers
	// +optional
	Identifiers []string `json:"identifiers,omitempty"`
	// CompareList: the list of fields to compare when checking if the resource is the same
	// +optional
	CompareList []string `json:"compareList,omitempty"`
}

// RestDefinitionSpec is the specification of a RestDefinition.
type RestDefinitionSpec struct {
	rtv1.ManagedSpec `json:",inline"`
	// Represent the path to the OAS Specification file
	OASPath string `json:"oasPath"`
	// Group: the group of the resource to manage
	// +immutable
	ResourceGroup string `json:"resourceGroup"`
	// The resource to manage
	// +optional
	Resource Resource `json:"resource"`
}

// RestDefinitionStatus is the status of a RestDefinition.
type RestDefinitionStatus struct {
	rtv1.ManagedStatus `json:",inline"`

	OASPath string `json:"oasPath"`
	// Created bool `json:"created"`
	// // Resource: the generated custom resource
	// // +optional
	// Resources  `json:"resource,omitempty"`

	// // PackageURL: .tgz or oci chart direct url
	// // +optional
	// PackageURL string `json:"packageUrl,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespaced,categories={krateo,restdefinition,core}
//+kubebuilder:printcolumn:name="RESOURCE",type="string",JSONPath=".status.oasPath"
//+kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp",priority=10

// RestDefinition is a RestDefinition type with a spec and a status.
type RestDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RestDefinitionSpec   `json:"spec,omitempty"`
	Status RestDefinitionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// RestDefinitionList is a list of RestDefinition objects.
type RestDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []RestDefinition `json:"items"`
}
