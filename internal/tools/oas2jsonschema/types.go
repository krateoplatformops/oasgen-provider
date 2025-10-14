package oas2jsonschema

import (
	"time"

	rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
)

// GeneratorConfig holds configuration options for the schema generator.
type GeneratorConfig struct {
	AcceptedMIMETypes        []string
	SuccessCodes             []int
	IncludeIdentifiersInSpec bool
	MaxRecursionDepth        int
	MaxRecursionNodes        int32
	RecursionTimeout         time.Duration
}

// DefaultGeneratorConfig returns a new GeneratorConfig with default values.
func DefaultGeneratorConfig() *GeneratorConfig {
	return &GeneratorConfig{
		AcceptedMIMETypes:        []string{"application/json"},
		SuccessCodes:             []int{200, 201},
		IncludeIdentifiersInSpec: false,
		MaxRecursionDepth:        50,
		MaxRecursionNodes:        5000,
		RecursionTimeout:         30 * time.Second,
	}
}

// ResourceConfig holds the necessary configuration extracted from a
// source (like a RestDefinition) to guide the schema generation process along with
// the OAS specification.
type ResourceConfig struct {
	Verbs                  []Verb
	Identifiers            []string
	AdditionalStatusFields []string
	ConfigurationFields    []ConfigurationField
	ExcludedSpecFields     []string
}

type ConfigurationField struct {
	FromOpenAPI        FromOpenAPI
	FromRestDefinition FromRestDefinition
}

type FromOpenAPI struct {
	Name string
	In   string // "query", "path", "header", "cookie" (TODO: add validation for this)
}

type FromRestDefinition struct {
	Actions []string
}

// Verb defines a specific API operation (action, method, path).
type Verb struct {
	Action string
	Method string
	Path   string
}

// --- Library-Agnostic Domain Models ---

// Schema is a library-agnostic representation of a JSON Schema Object, which is used
// within the OpenAPI specification to define the structure of data payloads.
// It is not a representation of the entire OpenAPI document itself.
// Potentially, this struct could be modified to include more fields in the future.
// It is the domainSchema defined in this domain (oas2jsonschema).
type Schema struct {
	Type                 []string // OAS 3.1 allows multiple types (e.g., ["string", "null"])
	Description          string
	Properties           []Property // Using a slice to preserve order of properties (TODO: consider using a map)
	Items                *Schema    // For array types, this defines the schema of items in the array
	AllOf                []*Schema
	Required             []string
	Default              interface{} // Default value for the schema
	Enum                 []interface{}
	AdditionalProperties bool
	MaxProperties        int
	Format               string                 // Not validated but added value to description if present
	Extensions           map[string]interface{} // For custom extensions like `x-crdgen-identifier-name`
}

// Property represents a single key-value pair in a schema's properties.
// Using a slice of these preserves order.
type Property struct {
	Name   string
	Schema *Schema
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
	Required    bool
	Schema      *Schema
}

// RequestBodyInfo is a library-agnostic representation of a request body.
// The Go type name reflects the OpenAPI spec's 'requestBody' object
type RequestBodyInfo struct {
	Content map[string]*Schema
}

// ResponseInfo is a library-agnostic representation of a response.
// The Go type name reflects the OpenAPI spec's single response object under the 'responses' map.
type ResponseInfo struct {
	Content map[string]*Schema
}

// GenerationResult holds the output of the schema generation process.
type GenerationResult struct {
	SpecSchema          []byte
	StatusSchema        []byte
	ConfigurationSchema []byte
	GenerationWarnings  []error
	ValidationWarnings  []error
}

type BasicAuth struct {
	UsernameRef rtv1.SecretKeySelector `json:"usernameRef"`
	PasswordRef rtv1.SecretKeySelector `json:"passwordRef"`
}

type BearerAuth struct {
	TokenRef rtv1.SecretKeySelector `json:"tokenRef"`
}

// deepCopy creates a deep copy of the Schema.
func (s *Schema) deepCopy() *Schema {
	// Initialize a map to track visited schemas to handle circular references.
	visited := make(map[*Schema]*Schema)
	return s.deepCopyRec(visited)
}

func (s *Schema) deepCopyRec(visited map[*Schema]*Schema) *Schema {
	if s == nil {
		return nil
	}

	// If we have already copied this schema, return the existing copy to break the cycle.
	if copied, ok := visited[s]; ok {
		return copied
	}

	// Create a new schema and register it in the visited map before recursing.
	newSchema := &Schema{}
	visited[s] = newSchema

	// Copy scalar fields and slices of basic types.
	if s.Type != nil {
		newSchema.Type = append([]string{}, s.Type...)
	}
	newSchema.Description = s.Description
	if s.Required != nil {
		newSchema.Required = append([]string{}, s.Required...)
	}

	// Note: Default and Enum are shallow-copied. This is an accepted limitation
	// as they are expected to contain primitive types.
	newSchema.Default = s.Default
	newSchema.AdditionalProperties = s.AdditionalProperties
	newSchema.MaxProperties = s.MaxProperties

	if s.Enum != nil {
		newSchema.Enum = make([]interface{}, len(s.Enum))
		copy(newSchema.Enum, s.Enum)
	}

	// Recursively copy nested schemas, passing the visited map along.
	if s.Items != nil {
		newSchema.Items = s.Items.deepCopyRec(visited)
	}

	if s.Properties != nil {
		newSchema.Properties = make([]Property, len(s.Properties))
		for i, p := range s.Properties {
			var copiedSchema *Schema
			if p.Schema != nil {
				copiedSchema = p.Schema.deepCopyRec(visited)
			}
			newSchema.Properties[i] = Property{
				Name:   p.Name,
				Schema: copiedSchema,
			}
		}
	}

	if s.AllOf != nil {
		newSchema.AllOf = make([]*Schema, len(s.AllOf))
		for i, allOfSchema := range s.AllOf {
			newSchema.AllOf[i] = allOfSchema.deepCopyRec(visited)
		}
	}

	return newSchema
}
