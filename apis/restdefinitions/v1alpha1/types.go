package v1alpha1

import (
	rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Pagination defines the pagination strategy for a "findby" action.
// Currently, only 'continuationToken' is supported.
// +kubebuilder:validation:XValidation:rule="self.type == 'continuationToken' ? has(self.continuationToken) : true",message="continuationToken configuration must be provided when type is 'continuationToken'"
type Pagination struct {
	// Type specifies the pagination strategy. Currently, only 'continuationToken' is supported.
	// +kubebuilder:validation:Enum=continuationToken
	// +required
	Type string `json:"type"`
	// Configuration for 'continuationToken' pagination. Required if type is 'continuationToken'.
	// +optional
	ContinuationToken *ContinuationTokenConfig `json:"continuationToken,omitempty"`

	// (Future) Configuration for 'pageNumber' pagination.
	// +optional
	//PageNumber *PageNumberConfig `json:"pageNumber,omitempty"`

	// (Future) Configuration for 'offset' pagination.
	// +optional
	//Offset *OffsetConfig `json:"offset,omitempty"`
}

// ContinuationTokenConfig holds the specific settings for token-based pagination.
type ContinuationTokenConfig struct {
	// Request: defines how to include the pagination token in the API request.
	// +required
	Request ContinuationTokenRequest `json:"request"`
	// Response: defines how to extract the pagination token from the API response.
	// +required
	Response ContinuationTokenResponse `json:"response"`
}

// ContinuationTokenRequest defines how to include the pagination token in the API request.
type ContinuationTokenRequest struct {
	// Where the token is located: "query", "header" or "body". Currently, only "query" is supported.
	// +kubebuilder:validation:Enum=query
	// +required
	TokenIn string `json:"tokenIn"`
	// The path or name of the query parameter, header, or body field.
	// For query parameters and headers, this is simply the name.
	// For body fields, this should be a JSON path.
	// +required
	TokenPath string `json:"tokenPath"`
}

// ContinuationTokenResponse defines how to extract the pagination token from the API response.
type ContinuationTokenResponse struct {
	// Where the token is located: "header" or "body". Currently, only "header" is supported.
	// +kubebuilder:validation:Enum=header
	// +required
	TokenIn string `json:"tokenIn"`
	// The path or name of the header or body field.
	// For headers, this is simply the name.
	// For body fields, this should be a JSON path.
	// +required
	TokenPath string `json:"tokenPath"`
}

// PageNumberConfig is a placeholder for future page number pagination settings.
//type PageNumberConfig struct{}

// OffsetConfig is a placeholder for future offset pagination settings.
//type OffsetConfig struct{}

// RequestFieldMappingItem defines a single mapping from a path parameter, query parameter or body field
// to a field in the Custom Resource.
// +kubebuilder:validation:XValidation:rule="(has(self.inPath) ? 1 : 0) + (has(self.inQuery) ? 1 : 0) + (has(self.inBody) ? 1 : 0) == 1",message="Either inPath, inQuery or inBody must be set, but not more than one"
type RequestFieldMappingItem struct {
	// InPath defines the name of the path parameter to be mapped.
	// Only one of 'inPath', 'inQuery' or 'inBody' can be set.
	// +optional
	InPath string `json:"inPath,omitempty"`
	// InQuery defines the name of the query parameter to be mapped.
	// Only one of 'inPath', 'inQuery' or 'inBody' can be set.
	// +optional
	InQuery string `json:"inQuery,omitempty"`
	// InBody defines the name of the body parameter to be mapped.
	// Only one of 'inPath', 'inQuery' or 'inBody' can be set.
	// +optional
	InBody string `json:"inBody,omitempty"`
	// InCustomResource defines the JSONPath to the field within the Custom Resource that holds the value.
	// For example: 'spec.name' or 'status.metadata.id'.
	// Note: potentially we could add validation to ensure this is a valid path (e.g., starts with 'spec.' or 'status.').
	// Currently, no validation is enforced on the content of this field.
	// +kubebuilder:validation:Required
	InCustomResource string `json:"inCustomResource"`
}

// +kubebuilder:validation:XValidation:rule="self.action == 'findby' || !has(self.identifiersMatchPolicy)",message="identifiersMatchPolicy can only be set for 'findby' actions"
// +kubebuilder:validation:XValidation:rule="self.action == 'findby' || !has(self.pagination)",message="pagination can only be set for 'findby' actions"
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
	// RequestFieldMapping provides explicit mapping from API parameters (path, query, or body)
	// to fields in the Custom Resource.
	// +optional
	RequestFieldMapping []RequestFieldMappingItem `json:"requestFieldMapping,omitempty"`
	// IdentifiersMatchPolicy defines how to match identifiers for the 'findby' action. To be set only for 'findby' actions.
	// If not set, defaults to 'OR'.
	// Possible values are 'AND' or 'OR'.
	// - 'AND': all identifiers must match.
	// - 'OR': at least one identifier must match (the default behavior).
	// +kubebuilder:validation:Enum=AND;OR
	// +optional
	IdentifiersMatchPolicy string `json:"identifiersMatchPolicy,omitempty"`
	// Pagination defines the pagination strategy for 'findby' actions. To be set only for 'findby' actions.
	// If not set, no pagination will be used.
	// +optional
	Pagination *Pagination `json:"pagination,omitempty"`
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
	// ExcludedSpecFields: the list of fields to exclude from the spec of the generated CRD (for example server-generated technical IDs could be excluded)
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ExcludedSpecFields are immutable, you cannot change them once the CRD has been generated"
	// +optional
	ExcludedSpecFields []string `json:"excludedSpecFields,omitempty"`
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
	// Actions: the list of actions this configuration applies to. Use ["*"] to apply to all actions.
	// +kubebuilder:validation:MinItems=1
	// +required
	Actions []string `json:"actions"`
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
