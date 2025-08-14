package oas2jsonschema

import (
	"encoding/json"
	"fmt"
)

// getPrimaryType returns the primary type from a slice of types introuduced in OpenAPI 3.1.
// which allows multiple types including "null".
// Source: https://www.openapis.org/blog/2021/02/16/migrating-from-openapi-3-0-to-3-1-0
func getPrimaryType(types []string) string {
	for _, t := range types {
		if t != "null" {
			return t
		}
	}
	return ""
}

// areTypesCompatible checks if two slices of types are compatible based on their primary non-null type (OAS 3.1).
// The opinionated compatibility rules are:
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

func prepareSchemaForCRD(schema *Schema) error {
	if schema == nil {
		return nil
	}

	if getPrimaryType(schema.Type) == "number" {
		convertNumberToInteger(schema)
	}

	if getPrimaryType(schema.Type) == "array" {
		return prepareSchemaForCRD(schema.Items)
	}

	for _, prop := range schema.Properties {
		if err := prepareSchemaForCRD(prop.Schema); err != nil {
			return err
		}
	}

	for _, allOfSchema := range schema.AllOf {
		if err := prepareSchemaForCRD(allOfSchema); err != nil {
			return err
		}
		schema.Properties = append(schema.Properties, allOfSchema.Properties...)
	}

	return nil
}

// convertNumberToInteger converts "number" types to "integer" types.
func convertNumberToInteger(schema *Schema) {
	for i, t := range schema.Type {
		if t == "number" {
			schema.Type[i] = "integer"
		}
	}
}

// schemaToMap converts our domain-specific Schema object into a map[string]interface{}
// suitable for JSON marshalling. This is the key to making the generator library-agnostic.
func schemaToMap(schema *Schema) (map[string]interface{}, error) {
	if schema == nil {
		return nil, nil
	}

	m := make(map[string]interface{})

	if len(schema.Type) > 0 {
		// Handle single vs. multiple types for JSON output
		if len(schema.Type) == 1 {
			m["type"] = schema.Type[0]
		} else {
			m["type"] = schema.Type
		}
	}

	if schema.Description != "" {
		m["description"] = schema.Description
	}

	if len(schema.Required) > 0 {
		m["required"] = schema.Required
	}

	if len(schema.Properties) > 0 {
		props := make(map[string]interface{})
		for _, p := range schema.Properties {
			propMap, err := schemaToMap(p.Schema)
			if err != nil {
				return nil, fmt.Errorf("could not convert property '%s': %w", p.Name, err)
			}
			if propMap != nil {
				props[p.Name] = propMap
			}
		}
		m["properties"] = props
	}

	if schema.Items != nil {
		itemsMap, err := schemaToMap(schema.Items)
		if err != nil {
			return nil, fmt.Errorf("could not convert items schema: %w", err)
		}
		if itemsMap != nil {
			m["items"] = itemsMap
		}
	}

	if len(schema.AllOf) > 0 {
		var allOfList []interface{}
		for _, s := range schema.AllOf {
			allOfMap, err := schemaToMap(s)
			if err != nil {
				return nil, fmt.Errorf("could not convert allOf item: %w", err)
			}
			if allOfMap != nil {
				allOfList = append(allOfList, allOfMap)
			}
		}
		m["allOf"] = allOfList
	}

	return m, nil
}

// GenerateJsonSchema converts a domain-specific Schema object into a JSON schema byte slice.
func GenerateJsonSchema(schema *Schema) ([]byte, error) {
	schemaMap, err := schemaToMap(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to convert schema to map: %w", err)
	}

	// Add standard JSON schema fields
	schemaMap["$schema"] = "http://json-schema.org/draft-07/schema#"

	return json.MarshalIndent(schemaMap, "", "  ")
}
