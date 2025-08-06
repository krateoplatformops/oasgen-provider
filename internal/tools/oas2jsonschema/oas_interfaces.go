package oas2jsonschema

// GeneratorConfig holds configuration options for the schema generator.
type GeneratorConfig struct {
	AcceptedMIMETypes []string
	SuccessCodes      []int
}

// DefaultGeneratorConfig returns a new GeneratorConfig with default values.
func DefaultGeneratorConfig() *GeneratorConfig {
	return &GeneratorConfig{
		AcceptedMIMETypes: []string{"application/json"},
		SuccessCodes:      []int{200, 201},
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
// The 'Type' field is the high-level category (e.g., 'http', 'apiKey').
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

// --- Interfaces ---

// OASDocument defines the contract for accessing an OpenAPI specification.
type OASDocument interface {
	FindPath(path string) (PathItem, bool)
	SecuritySchemes() []SecuritySchemeInfo
}

// PathItem defines the contract for a single API path.
type PathItem interface {
	GetOperations() map[string]Operation
}

// Operation defines the contract for a single API operation.
type Operation interface {
	GetParameters() []ParameterInfo
	GetRequestBody() RequestBodyInfo
	GetResponses() map[int]ResponseInfo
}

// GenerationResult holds the output of the schema generation process.
type GenerationResult struct {
	SpecSchema   []byte
	StatusSchema []byte
	AuthSchemas  map[string][]byte
	Warnings     []error
}

