package oas2jsonschema

import (
	"fmt"
	"strings"

	definitionv1alpha1 "github.com/krateoplatformops/oasgen-provider/apis/restdefinitions/v1alpha1"
)

const (
	ActionGet    = "get"
	ActionFindBy = "findby"
	ActionCreate = "create"
	ActionUpdate = "update"
)

func determineBaseAction(verbs []definitionv1alpha1.VerbsDescription) (string, error) {
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

func validateSchemas(doc OASDocument, verbs []definitionv1alpha1.VerbsDescription, config *GeneratorConfig) []error {
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
			errors = append(errors, compareActionResponseSchemas(doc, verbs, verb.Action, baseAction, config)...)
		}
	}

	return errors
}

func compareActionResponseSchemas(doc OASDocument, verbs []definitionv1alpha1.VerbsDescription, action1, action2 string, config *GeneratorConfig) []error {
	schema2, err := extractSchemaForAction(doc, verbs, action2, config)
	if err != nil {
		return []error{SchemaValidationError{
			Code:    CodeActionSchemaMissing,
			Message: fmt.Sprintf("could not extract base schema for action '%s': %v", action2, err),
		}}
	}

	schema1, err := extractSchemaForAction(doc, verbs, action1, config)
	if err != nil {
		return []error{SchemaValidationError{
			Code:    CodeActionSchemaMissing,
			Message: fmt.Sprintf("could not extract schema for action '%s' to compare: %v", action1, err),
		}}
	}

	return compareSchemas(".", schema1, schema2)
}

func compareSchemas(path string, schema1, schema2 *Schema) []error {
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
		errors = append(errors, SchemaValidationError{
			Path:    path,
			Code:    CodePropertyMismatch,
			Message: "one schema has properties but the other does not",
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
			errors = append(errors, compareSchemas(currentPath, prop1.Schema, prop2.Schema)...)
		case "array":
			if prop1.Schema.Items != nil && prop2.Schema.Items != nil {
				errors = append(errors, compareSchemas(currentPath, prop1.Schema.Items, prop2.Schema.Items)...)
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

func getPrimaryType(types []string) string {
	for _, t := range types {
		if t != "null" {
			return t
		}
	}
	return ""
}

// areTypesCompatible checks if two slices of types are compatible based on their primary non-null type.
// The compatibility rules are:
// 1. If both have a primary type (e.g., "string", "object"), they must be identical.
// 2. If one has a primary type and the other does not (i.e., is only "null" or empty), they are incompatible.
// 3. If neither has a primary type, they are compatible (e.g., ["null"] vs []).
func areTypesCompatible(types1, types2 []string) bool {
	primaryType1 := getPrimaryType(types1)
	primaryType2 := getPrimaryType(types2)

	// If both have a primary type, they must be the same.
	// If one has a primary type and the other doesn't, they are not compatible.
	return primaryType1 == primaryType2
}

func extractSchemaForAction(doc OASDocument, verbs []definitionv1alpha1.VerbsDescription, targetAction string, config *GeneratorConfig) (*Schema, error) {
	var verbFound bool
	for _, verb := range verbs {
		if !strings.EqualFold(verb.Action, targetAction) {
			continue
		}
		verbFound = true

		path, ok := doc.FindPath(verb.Path)
		if !ok {
			return nil, fmt.Errorf("path '%s' not found in OAS document", verb.Path)
		}

		ops := path.GetOperations()
		op, ok := ops[strings.ToLower(verb.Method)]
		if !ok {
			return nil, fmt.Errorf("method '%s' not found for path '%s'", verb.Method, verb.Path)
		}

		responses := op.GetResponses()
		if responses == nil {
			continue // Or return an error if responses are expected
		}

		for _, code := range config.SuccessCodes {
			resp, ok := responses[code]
			if !ok {
				continue
			}

			for _, mimeType := range config.AcceptedMIMETypes {
				schema, ok := resp.Content[mimeType]
				if !ok || schema == nil {
					continue
				}

				// If a schema is found, return it immediately.
				if strings.EqualFold(targetAction, ActionFindBy) && schema.Items != nil {
					return schema.Items, nil
				}
				return schema, nil
			}
		}
	}

	if !verbFound {
		return nil, fmt.Errorf("action '%s' not defined in resource verbs", targetAction)
	}

	return nil, fmt.Errorf("no suitable response schema found for action '%s'", targetAction)
}
