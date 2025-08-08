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