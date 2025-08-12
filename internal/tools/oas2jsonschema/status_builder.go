package oas2jsonschema

import "fmt"

// generateStatusSchema generates the complete status schema for a given resource.
func (g *OASSchemaGenerator) generateStatusSchema() ([]byte, []error, error) {
	var warnings []error

	allStatusFields := append(g.resourceConfig.Identifiers, g.resourceConfig.AdditionalStatusFields...)
	if len(allStatusFields) == 0 {
		return nil, []error{SchemaGenerationError{Code: CodeNoStatusSchema, Message: "no identifiers or additional status fields defined, skipping status schema generation"}}, nil
	}

	responseSchema, err := g.getBaseSchemaForStatus()
	if err != nil {
		warnings = append(warnings, SchemaGenerationError{Message: fmt.Sprintf("schema validation warning: %v", err)})
	}
	if responseSchema == nil {
		warnings = append(warnings, SchemaGenerationError{Code: CodeNoStatusSchema, Message: "could not find a GET or FINDBY response schema for status generation"})
	}

	statusSchema, buildWarnings := buildStatusSchema(allStatusFields, responseSchema)
	warnings = append(warnings, buildWarnings...)

	if err := prepareSchemaForCRD(statusSchema); err != nil {
		return nil, warnings, fmt.Errorf("could not prepare status schema for CRD: %w", err)
	}

	byteSchema, err := GenerateJsonSchema(statusSchema)
	if err != nil {
		return nil, warnings, fmt.Errorf("could not generate final JSON schema for status: %w", err)
	}

	return byteSchema, warnings, nil
}

// buildStatusSchema builds the status schema from the response schema and the list of status fields.
func buildStatusSchema(allStatusFields []string, responseSchema *Schema) (*Schema, []error) {
	var warnings []error
	var props []Property
	for _, fieldName := range allStatusFields {
		found := false
		if responseSchema != nil {
			for _, p := range responseSchema.Properties {
				if p.Name == fieldName {
					props = append(props, p)
					found = true
					break
				}
			}
		}
		if !found {
			warnings = append(warnings, SchemaGenerationError{Code: CodeStatusFieldNotFound, Message: fmt.Sprintf("status field '%s' not found in response, defaulting to string", fieldName)})
			props = append(props, Property{Name: fieldName, Schema: &Schema{Type: []string{"string"}}})
		}
	}
	return &Schema{Type: []string{"object"}, Properties: props}, warnings
}
