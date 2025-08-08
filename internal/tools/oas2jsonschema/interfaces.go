package oas2jsonschema

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
	GetParameters() []ParameterInfo     // There can be multiple parameters for an operation.
	GetRequestBody() RequestBodyInfo    // There is only one request body per operation.
	GetResponses() map[int]ResponseInfo // The keys are HTTP status codes and therefore there could be multiple responses.
}
