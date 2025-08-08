package oas2jsonschema

import (
	"fmt"

	rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
)

// GeneratorConfig holds configuration options for the schema generator.
type GeneratorConfig struct {
	AcceptedMIMETypes        []string
	SuccessCodes             []int
	IncludeIdentifiersInSpec bool
}

// DefaultGeneratorConfig returns a new GeneratorConfig with default values.
func DefaultGeneratorConfig() *GeneratorConfig {
	return &GeneratorConfig{
		AcceptedMIMETypes:        []string{"application/json"},
		SuccessCodes:             []int{200, 201},
		IncludeIdentifiersInSpec: false,
	}
}

// --- Library-Agnostic Domain Models ---

// Property represents a single key-value pair in a schema's properties.
// Using a slice of these preserves order.
type Property struct {
	Name   string
	Schema *Schema
}

// Schema is a library-agnostic representation of a JSON Schema Object, which is used
// within the OpenAPI specification to define the structure of data payloads.
// It is not a representation of the entire OpenAPI document itself.
type Schema struct {
	Type        []string
	Description string
	Properties  []Property
	Items       *Schema
	AllOf       []*Schema
	Required    []string
}

// SecuritySchemeType defines the type of a security scheme (e.g., http, apiKey).
type SecuritySchemeType string

// Source: https://swagger.io/docs/specification/v3_0/authentication/
const (
	SchemeTypeHTTP          SecuritySchemeType = "http"
	SchemeTypeAPIKey        SecuritySchemeType = "apiKey"        // Currently not supported
	SchemeTypeOAuth2        SecuritySchemeType = "oauth2"        // Currently not supported
	SchemeTypeOpenIDConnect SecuritySchemeType = "openIdConnect" // Currently not supported
)

// SecuritySchemeInfo is a library-agnostic representation of a security scheme.
// It mirrors the structure of an OpenAPI security scheme.
// In this Go code, it is a "sum type" that captures different security scheme types.
// The 'Type' field is the high-level category (e.g., 'http', 'apiKey', 'oauth2', 'openIdConnect').
// The 'Scheme' field is a sub-detail used only when Type is 'http' (e.g., 'basic', 'bearer').
// Other fields like 'In' and 'ParamName' are used for other types (e.g., 'apiKey').
type SecuritySchemeInfo struct {
	Name      string
	Type      SecuritySchemeType
	Scheme    string // e.g., "basic", "bearer"
	In        string // e.g., "header", "query"
	ParamName string // The name of the header or query parameter (for apiKey)
}

// ParameterInfo is a library-agnostic representation of an API parameter.
type ParameterInfo struct {
	Name        string
	In          string
	Description string
	Schema      *Schema
}

// RequestBodyInfo is a library-agnostic representation of a request body.
// Type name reflect the OpenAPI spec's 'requestBody' object
type RequestBodyInfo struct {
	Content map[string]*Schema
}

// ResponseInfo is a library-agnostic representation of a response.
// Type name reflects the OpenAPI spec's single response object under the 'responses' map.
type ResponseInfo struct {
	Content map[string]*Schema
}

// GenerationResult holds the output of the schema generation process.
type GenerationResult struct {
	SpecSchema         []byte
	StatusSchema       []byte
	AuthCRDSchemas     map[string][]byte
	GenerationWarnings []error
	ValidationWarnings []error
}

// GenerationCode defines a machine-readable code for the type of generation warning.
type GenerationCode string

const (
	// CodeDuplicateParameter indicates that a parameter is defined in multiple verbs.
	CodeDuplicateParameter GenerationCode = "DuplicateParameter"
	// CodePathNotFound indicates that a path specified in the RestDefinition was not found in the OpenAPI spec.
	CodePathNotFound GenerationCode = "PathNotFound"
	// CodeStatusFieldNotFound indicates that a status field was not found in the response schema.
	CodeStatusFieldNotFound GenerationCode = "StatusFieldNotFound"
	// CodeNoRootSchema indicates that no base schema could be found for the spec.
	CodeNoRootSchema GenerationCode = "NoRootSchema"
	// CodeNoStatusSchema indicates that no schema could be found for the status.
	CodeNoStatusSchema GenerationCode = "NoStatusSchema"
)

// SchemaGenerationError defines a structured error for schema generation warnings.
type SchemaGenerationError struct {
	Path    string
	Code    GenerationCode
	Message string

	Got      any
	Expected any
}

func (e SchemaGenerationError) Error() string {
	return fmt.Sprintf("generation error at %s: %s", e.Path, e.Message)
}

// ValidationCode defines a machine-readable code for the type of error.
type ValidationCode string

const (
	// CodeMissingBaseAction indicates that no 'get' or 'findby' action was found.
	CodeMissingBaseAction ValidationCode = "MissingBaseAction"
	// CodeActionSchemaMissing indicates that the schema for a specific action is nil.
	CodeActionSchemaMissing ValidationCode = "ActionSchemaMissing"
	// CodeTypeMismatch indicates a type mismatch between two schemas.
	CodeTypeMismatch ValidationCode = "TypeMismatch"
	// CodePropertyMismatch indicates that one schema has properties while the other does not.
	CodePropertyMismatch ValidationCode = "PropertyMismatch"
	// CodeMissingArrayItems indicates that one schema has array items while the other does not.
	CodeMissingArrayItems ValidationCode = "MissingArrayItems"
)

// SchemaValidationError defines a structured error for schema validation.
type SchemaValidationError struct {
	Path    string
	Code    ValidationCode
	Message string

	Got      any
	Expected any
}

func (e SchemaValidationError) Error() string {
	return fmt.Sprintf("validation error at %s: %s", e.Path, e.Message)
}

type BasicAuth struct {
	Username    string                 `json:"username"`
	PasswordRef rtv1.SecretKeySelector `json:"passwordRef"`
}

type BearerAuth struct {
	TokenRef rtv1.SecretKeySelector `json:"tokenRef"`
}
