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

func validateSchemas(doc OASDocument, verbs []definitionv1alpha1.VerbsDescription, config *GeneratorConfig) []error {
	var errors []error

	availableActions := make(map[string]bool)
	for _, verb := range verbs {
		availableActions[verb.Action] = true
	}

	baseAction := ""
	if availableActions[ActionGet] {
		baseAction = ActionGet
	} else if availableActions[ActionFindBy] {
		baseAction = ActionFindBy
	}

	if baseAction == "" {
		errors = append(errors, SchemaValidationError{
			Code:    CodeMissingBaseAction,
			Message: "no 'get' or 'findby' action found to serve as a base for schema validation",
		})
		return errors
	}

	actionsToCompare := []string{ActionCreate, ActionUpdate}
	if baseAction == ActionGet && availableActions[ActionFindBy] {
		actionsToCompare = append(actionsToCompare, ActionFindBy)
	}

	for _, action := range actionsToCompare {
		if availableActions[action] {
			errors = append(errors, compareActionResponseSchemas(doc, verbs, action, baseAction, config)...)
		}
	}

	return errors
}

func compareActionResponseSchemas(doc OASDocument, verbs []definitionv1alpha1.VerbsDescription, action1, action2 string, config *GeneratorConfig) []error {
	schema2, err := extractSchemaForAction(doc, verbs, action2, config)
	if err != nil {
		return []error{SchemaValidationError{
			Message: fmt.Sprintf("error when calling extractSchemaForAction for action %s: %v", action2, err),
		}}
	}
	if schema2 == nil {
		return []error{SchemaValidationError{
			Code:    CodeActionSchemaMissing,
			Message: fmt.Sprintf("schema for action %s is nil, cannot compare", action2),
		}}
	}

	schema1, err := extractSchemaForAction(doc, verbs, action1, config)
	if err != nil {
		return []error{SchemaValidationError{
			Message: fmt.Sprintf("error when calling extractSchemaForAction for action %s: %v", action1, err),
		}}
	}
	if schema1 == nil {
		return []error{SchemaValidationError{
			Code:    CodeActionSchemaMissing,
			Message: fmt.Sprintf("schema for action %s is nil, cannot compare", action1),
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
// The compatibility rules are designed to be lenient but safe for CRD generation:
//  1. If both have a primary type (e.g., "string", "object"), they must be identical.
//  2. If one schema defines a type (e.g., ["string", "null"]) and the other is just ["null"] or empty,
//     they are considered compatible. This handles cases where a field is optional in one response but not another.
//  3. If both are effectively "null" or empty, they are compatible.
func areTypesCompatible(types1, types2 []string) bool {
	primaryType1 := getPrimaryType(types1)
	primaryType2 := getPrimaryType(types2)

	if primaryType1 != "" && primaryType2 != "" {
		return primaryType1 == primaryType2
	}

	if primaryType1 != "" && primaryType2 == "" {
		for _, t := range types1 {
			if t == "null" {
				return true
			}
		}
		return false
	}

	if primaryType1 == "" && primaryType2 != "" {
		for _, t := range types2 {
			if t == "null" {
				return true
			}
		}
		return false
	}

	return true
}

func extractSchemaForAction(doc OASDocument, verbs []definitionv1alpha1.VerbsDescription, targetAction string, config *GeneratorConfig) (*Schema, error) {
	for _, verb := range verbs {
		if !strings.EqualFold(verb.Action, targetAction) {
			continue
		}

		path, ok := doc.FindPath(verb.Path)
		if !ok {
			continue
		}

		ops := path.GetOperations()
		op, ok := ops[strings.ToLower(verb.Method)]
		if !ok {
			continue
		}

		responses := op.GetResponses()
		if responses == nil {
			continue
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

				if strings.EqualFold(targetAction, ActionFindBy) && schema.Items != nil {
					return schema.Items, nil
				}
				return schema, nil
			}
		}
	}
	return nil, nil
}
