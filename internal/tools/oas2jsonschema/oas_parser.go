package oas2jsonschema

import (
	"errors"
	"fmt"

	"github.com/pb33f/libopenapi"
)

// Parse takes raw OpenAPI specification content and returns a document
// that conforms to the OASDocument interface. It handles parsing, building,
// and resolving the model.
func Parse(content []byte) (OASDocument, error) {
	d, err := libopenapi.NewDocument(content)
	if err != nil {
		return nil, fmt.Errorf("failed to create new libopenapi document: %w", err)
	}

	doc, modelErrors := d.BuildV3Model()
	if len(modelErrors) > 0 {
		return nil, fmt.Errorf("failed to build V3 model: %w", errors.Join(modelErrors...))
	}
	if doc == nil {
		return nil, errors.New("failed to build V3 model, resulting document was nil")
	}

	// Resolve model references
	resolvingErrors := doc.Index.GetResolver().Resolve()
	if len(resolvingErrors) > 0 {
		var errs []error
		for i := range resolvingErrors {
			errs = append(errs, resolvingErrors[i].ErrorRef)
		}
		return nil, fmt.Errorf("failed to resolve model references: %w", errors.Join(errs...))
	}

	return NewLibOASDocumentAdapter(doc), nil
}
