package v1alpha1

import (
	rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VerbsDescription struct {
	// Name of the action to perform when this api is called [create, update, get, delete, findby]
	// +kubebuilder:validation:Enum=create;update;get;delete;findby
	// +required
	Action string `json:"action"`
	// Method: the http method to use [GET, POST, PUT, DELETE, PATCH]
	// +kubebuilder:validation:Enum=GET;POST;PUT;DELETE;PATCH
	// +required
	Method string `json:"method"`
	// Path: the path to the api - has to be the same path as the one in the OAS file you are referencing
	// +required
	Path string `json:"path"`
}

type Resource struct {
	// Name: the name of the resource to manage
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Kind is immutable, you cannot change that once the CRD has been generated"
	// +required
	Kind string `json:"kind"`
	// VerbsDescription: the list of verbs to use on this resource
	// +required
	VerbsDescription []VerbsDescription `json:"verbsDescription"`
	// Identifiers: the list of fields to use as identifiers - used to populate the status of the resource
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Identifiers are immutable, you cannot change them once the CRD has been generated"
	// +optional
	Identifiers []string `json:"identifiers,omitempty"`
	// AdditionalStatusFields: the list of fields to use as additional status fields - used to populate the status of the resource
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="AdditionalStatusFields are immutable, you cannot change them once the CRD has been generated"
	// +optional
	AdditionalStatusFields []string `json:"additionalStatusFields,omitempty"`
	// ConfigurationFields: the list of fields to use as configuration fields
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ConfigurationFields are immutable, you cannot change them once the CRD has been generated"
	// +optional
	ConfigurationFields []ConfigurationField `json:"configurationFields,omitempty"`
}

// RestDefinitionSpec is the specification of a RestDefinition.
type RestDefinitionSpec struct {
	// Path to the OpenAPI specification. This value can change over time, for example if the OAS file is updated but be sure to not change the requestbody of the `create` verb.
	// +required
	// - configmap://<namespace>/<name>/<key>
	// - http(s)://<url>
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^(configmap:\/\/([a-z0-9-]+)\/([a-z0-9-]+)\/([a-zA-Z0-9.-_]+)|https?:\/\/\S+)$`
	OASPath string `json:"oasPath"`
	// Group: the group of the resource to manage
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ResourceGroup is immutable, you cannot change that once the CRD has been generated"
	// +required
	ResourceGroup string `json:"resourceGroup"`
	// The resource to manage
	// +required
	Resource Resource `json:"resource"`
}

type ConfigurationField struct {
	FromOpenAPI        FromOpenAPI        `json:"fromOpenAPI"`
	FromRestDefinition FromRestDefinition `json:"fromRestDefinition"`
}

type FromOpenAPI struct {
	Name string `json:"name"`
	In   string `json:"in"` // "query", "path", "header", "cookie"
}

type FromRestDefinition struct {
	Action string `json:"action"`
}

type KindApiVersion struct {
	// APIVersion: the api version of the resource
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`

	// Kind: the kind of the resource
	// +optional
	Kind string `json:"kind,omitempty"`
}

// RestDefinitionStatus is the status of a RestDefinition.
type RestDefinitionStatus struct {
	rtv1.ConditionedStatus `json:",inline"`

	// OASPath: the path to the OAS Specification file.
	OASPath string `json:"oasPath"`

	// Resource: the resource to manage
	// +optional
	Resource KindApiVersion `json:"resource"`

	// Configuration: the configuration of the resource
	// +optional
	Configuration KindApiVersion `json:"configuration"`

	// Digest: the digest of the managed resources
	// +optional
	Digest string `json:"digest,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={krateo,restdefinition,core}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="API VERSION",type="string",JSONPath=".status.resource.apiVersion",priority=10
// +kubebuilder:printcolumn:name="KIND",type="string",JSONPath=".status.resource.kind",priority=10
// +kubebuilder:printcolumn:name="OAS PATH",type="string",JSONPath=".status.oasPath",priority=10
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

// GetCondition of this RestDefinition.
func (mg *RestDefinition) GetCondition(ct rtv1.ConditionType) rtv1.Condition {
	return mg.Status.GetCondition(ct)
}

// SetConditions of this RestDefinition.
func (mg *RestDefinition) SetConditions(c ...rtv1.Condition) {
	mg.Status.SetConditions(c...)
}
