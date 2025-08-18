package oas2jsonschema

import (
	"fmt"
)

const (
	ActionGet    = "get"
	ActionFindBy = "findby"
	ActionCreate = "create"
	ActionUpdate = "update"
)

func ValidateSchemas(doc OASDocument, verbs []Verb, config *GeneratorConfig) []error {
	baseAction, err := determineBaseAction(verbs)
	if err != nil {
		return []error{err}
	}

	var errors []error
	for _, verb := range verbs {
		// Determine if the current verb is one we need to compare against the base.
		isComparable := verb.Action == ActionCreate || verb.Action == ActionUpdate
		if baseAction == ActionGet && verb.Action == ActionFindBy {
			isComparable = true
		}

		if isComparable {
			// Perform the comparison against the base action schema.
			errors = append(errors, compareActionResponseSchemas(doc, verbs, verb.Action, baseAction, config)...)
		}
	}

	return errors
}

// TODO: make it configurable in the config
func determineBaseAction(verbs []Verb) (string, error) {
	hasGet := false
	hasFindBy := false
	for _, verb := range verbs {
		if verb.Action == ActionGet {
			hasGet = true
			break // 'get' takes precedence
		}
		if verb.Action == ActionFindBy {
			hasFindBy = true
		}
	}

	if hasGet {
		return ActionGet, nil
	}
	if hasFindBy {
		return ActionFindBy, nil
	}

	return "", SchemaValidationError{
		Code:    CodeMissingBaseAction,
		Message: "no 'get' or 'findby' action found to serve as a base for schema validation",
	}
}

func compareActionResponseSchemas(doc OASDocument, verbs []Verb, action1, action2 string, config *GeneratorConfig) []error {
	schema2, err := ExtractSchemaForAction(doc, verbs, action2, config)
	if err != nil {
		return []error{SchemaValidationError{
			Code:    CodeActionSchemaMissing,
			Message: fmt.Sprintf("could not extract base schema for action '%s': %v", action2, err),
		}}
	}

	schema1, err := ExtractSchemaForAction(doc, verbs, action1, config)
	if err != nil {
		return []error{SchemaValidationError{
			Code:    CodeActionSchemaMissing,
			Message: fmt.Sprintf("could not extract schema for action '%s' to compare: %v", action1, err),
		}}
	}

	return compareSchemas(".", schema1, schema2, action1, action2)
}

func compareSchemas(path string, schema1, schema2 *Schema, action1, action2 string) []error {
	var errors []error

	if schema1 == nil && schema2 == nil {
		return nil
	}
	if schema1 == nil {
		return []error{SchemaValidationError{Path: path, Message: "first schema is nil"}}
	}
	if schema2 == nil {
		return []error{SchemaValidationError{Path: path, Message: "second schema is nil"}}
	}

	schema1HasProps := len(schema1.Properties) > 0
	schema2HasProps := len(schema2.Properties) > 0

	if !schema1HasProps && !schema2HasProps {
		// Check if primary types are compatible
		if !areTypesCompatible(schema1.Type, schema2.Type) {
			errors = append(errors, SchemaValidationError{
				Path:     path,
				Code:     CodeTypeMismatch,
				Message:  fmt.Sprintf("type mismatch: first schema types are '%v', second are '%v'", schema1.Type, schema2.Type),
				Got:      schema1.Type,
				Expected: schema2.Type,
			})
		}
		return errors
	}

	if schema1HasProps != schema2HasProps {
		msg := "schema mismatch: response for action '%s' has properties but response for action '%s' does not"
		if !schema1HasProps {
			// Swap the message to be accurate
			msg = "schema mismatch: response for action '%s' does not have properties but response for action '%s' does"
		}
		errors = append(errors, SchemaValidationError{
			Path:    path,
			Code:    CodePropertyMismatch,
			Message: fmt.Sprintf(msg, action1, action2),
		})
		return errors
	}

	props2 := make(map[string]Property)
	for _, p := range schema2.Properties {
		props2[p.Name] = p
	}

	for _, prop1 := range schema1.Properties {
		prop2, ok := props2[prop1.Name]
		if !ok {
			continue
		}

		currentPath := buildPath(path, prop1.Name)

		if prop1.Schema == nil || prop2.Schema == nil {
			if prop1.Schema != prop2.Schema { // One is nil, the other is not
				errors = append(errors, SchemaValidationError{
					Path:    currentPath,
					Code:    CodePropertyMismatch,
					Message: fmt.Sprintf("schema for property '%s' is nil in one definition but not the other", currentPath),
				})
			}
			continue
		}

		if !areTypesCompatible(prop1.Schema.Type, prop2.Schema.Type) {
			errors = append(errors, SchemaValidationError{
				Path:     currentPath,
				Code:     CodeTypeMismatch,
				Message:  fmt.Sprintf("type mismatch for field '%s': first schema types are '%v', second are '%v'", currentPath, prop1.Schema.Type, prop2.Schema.Type),
				Got:      prop1.Schema.Type,
				Expected: prop2.Schema.Type,
			})
			continue
		}

		switch getPrimaryType(prop1.Schema.Type) {
		case "object":
			// recursively compare object schemas
			errors = append(errors, compareSchemas(currentPath, prop1.Schema, prop2.Schema, action1, action2)...)
		case "array":
			if prop1.Schema.Items != nil && prop2.Schema.Items != nil {
				// recursively compare array item schemas
				errors = append(errors, compareSchemas(currentPath, prop1.Schema.Items, prop2.Schema.Items, action1, action2)...)
			} else if prop1.Schema.Items != nil && prop2.Schema.Items == nil {
				errors = append(errors, SchemaValidationError{
					Path:    currentPath,
					Code:    CodeMissingArrayItems,
					Message: "second schema has no items for array",
				})
			} else if prop1.Schema.Items == nil && prop2.Schema.Items != nil {
				errors = append(errors, SchemaValidationError{
					Path:    currentPath,
					Code:    CodeMissingArrayItems,
					Message: "first schema has no items for array",
				})
			}
		}
	}

	return errors
}

func buildPath(base, field string) string {
	if base == "." {
		return field
	}
	return fmt.Sprintf("%s.%s", base, field)
}
