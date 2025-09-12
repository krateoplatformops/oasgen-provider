package oas2jsonschema

import "fmt"

// ParserErrorCode defines the type for parser-specific error codes.
type ParserErrorCode string

const (
	// CodeDocumentCreationError indicates an error when creating a new document from content.
	CodeDocumentCreationError ParserErrorCode = "DocumentCreationError"
	// CodeModelBuildError indicates an error when building the V3 model from the document.
	CodeModelBuildError ParserErrorCode = "ModelBuildError"
	// CodeModelResolutionError indicates an error when resolving references within the model.
	CodeModelResolutionError ParserErrorCode = "ModelResolutionError"
)

// ParserError represents a structured error from the OAS parser.
type ParserError struct {
	// Code is the machine-readable error code.
	Code ParserErrorCode
	// Message is the human-readable error message.
	Message string
	// Err is the underlying error, if any.
	Err error
}

// Error implements the error interface.
func (e ParserError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("parser error [%s]: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("parser error [%s]: %s", e.Code, e.Message)
}

// Unwrap provides compatibility for Go's errors.Is and errors.As.
func (e ParserError) Unwrap() error {
	return e.Err
}

// --------------------------------------------------------------------------------------------

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

// --------------------------------------------------------------------------------------------

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
	// CodeRecursionLimitExceeded indicates that the recursion limit was exceeded during validation.
	CodeRecursionLimitExceeded ValidationCode = "RecursionLimitExceeded"
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
