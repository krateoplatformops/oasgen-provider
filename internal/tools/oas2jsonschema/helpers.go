package oas2jsonschema

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/krateoplatformops/oasgen-provider/internal/tools/safety"
)

// getPrimaryType returns the primary type from a slice of types.
// The "type" slice of types was introduced in OpenAPI 3.1 which allows multiple types including "null".
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

// prepareSchemaForCRD prepares a schema for Kubernetes CRD generation by applying transformations.
// Namely:
// - it converts "number" types to "integer"
// - it merges "allOf" schemas for object types.
// It handles circular references by tracking visited schemas to prevent infinite recursion.
func prepareSchemaForCRD(schema *Schema, config *GeneratorConfig) error {
	if schema == nil {
		return nil
	}

	guard := safety.NewRecursionGuard(config.MaxRecursionDepth, config.MaxRecursionNodes, config.RecursionTimeout)
	ctx, cancel := guard.WithContext()
	defer cancel()

	return prepareSchemaForCRDWithVisited(ctx, schema, guard, make(map[*Schema]*Schema), 0)
}

// prepareSchemaForCRDWithVisited is the internal implementation that tracks visited schemas
// to handle circular references safely using placeholder schemas.
func prepareSchemaForCRDWithVisited(
	ctx context.Context,
	schema *Schema,
	guard *safety.RecursionGuard,
	visited map[*Schema]*Schema,
	depth int,
) error {
	if schema == nil {
		return nil
	}

	// Check recursion limits
	if err := guard.Check(ctx, depth); err != nil {
		log.Printf("CRD preparation recursion aborted at depth %d: %v", depth, err)
		return fmt.Errorf("recursion limit exceeded: %w", err)
	}

	// Detect already processed: if we've seen this schema before, skip processing
	if _, exists := visited[schema]; exists {
		//log.Printf("Already processed schema at depth %d: %v", depth, schema)
		//log.Printf("Schema info: Type=%v, Description=%q, Properties=%d", schema.Type, schema.Description, len(schema.Properties))
		//log.Printf("Skipping further processing.")
		return nil // Already processed
	}

	// Mark this schema as being processed
	visited[schema] = schema

	// Gracefully handle any panics during processing
	defer func() {
		if r := recover(); r != nil {
			log.Printf("CRD preparation panic at depth %d: %v", depth, r)
		}
	}()

	// Convert number types to integer for CRD compatibility
	if getPrimaryType(schema.Type) == "number" {
		convertNumberToInteger(schema)
	}

	// Process array items
	if getPrimaryType(schema.Type) == "array" && schema.Items != nil {
		if err := prepareSchemaForCRDWithVisited(ctx, schema.Items, guard, visited, depth+1); err != nil {
			return fmt.Errorf("failed to process array items: %w", err)
		}
	}

	// Process AllOf schemas and merge properties for object types
	if len(schema.AllOf) > 0 {
		// Create temporary slices to hold merged properties, required fields and enum values
		var mergedProperties []Property
		var mergedRequired []string
		var mergedEnum []interface{} // This is the case where there are only enum values and no properties

		for _, allOfSchema := range schema.AllOf {
			// Recursively prepare each schema within the allOf list
			if err := prepareSchemaForCRDWithVisited(ctx, allOfSchema, guard, visited, depth+1); err != nil {
				return fmt.Errorf("failed to process allOf schema: %w", err)
			}

			// Merge from the child schema
			if allOfSchema != nil {
				mergedProperties = append(mergedProperties, allOfSchema.Properties...)
				mergedRequired = append(mergedRequired, allOfSchema.Required...)
				if len(allOfSchema.Enum) > 0 && len(allOfSchema.Properties) == 0 {
					// If the allOf schema has enum values but no properties, we consider its enum values
					// Example reference: SubnetType in ArubaCloud Subnet schema
					mergedEnum = append(mergedEnum, allOfSchema.Enum...)
				}

				// Inherit type only if the main schema doesn't have one
				if len(schema.Type) == 0 && len(allOfSchema.Type) > 0 {
					schema.Type = allOfSchema.Type
				} else {
					// If both have types, ensure they are compatible
					// Otherwise, log a warning
					if !areTypesCompatible(schema.Type, allOfSchema.Type) {
						log.Printf("Warning: Incompatible types in allOf merge: %v vs %v", schema.Type, allOfSchema.Type)
						log.Printf("Schema info: Type=%v, Description=%q, Properties=%d", schema.Type, schema.Description, len(schema.Properties))
					}
				}
			}
		}

		// Append the merged properties to the original schema
		propIndex := make(map[string]int)
		// start with existing properties and map their names to indexes
		for i, p := range schema.Properties {
			propIndex[p.Name] = i
		}
		for _, p := range mergedProperties {
			if _, exists := propIndex[p.Name]; exists {
				// If property already exists, we keep the existing one (no overwrite policy)
				//log.Printf("Property '%s' already exists in schema; skipping merge from allOf", p.Name)
			} else {
				// New property, add it
				propIndex[p.Name] = len(schema.Properties) // new index (last position)
				schema.Properties = append(schema.Properties, p)
			}
		}

		// Handle 'required' fields with deduplication
		// We need to avoid duplicates in the 'required' list like ["id", "name", "id"]
		requiredSet := make(map[string]struct{})
		for _, req := range schema.Required { // Add existing required fields from the main schema
			requiredSet[req] = struct{}{}
		}
		for _, req := range mergedRequired { // Add merged required fields from allOf schemas
			requiredSet[req] = struct{}{}
		}
		newRequired := make([]string, 0, len(requiredSet))
		for req := range requiredSet {
			newRequired = append(newRequired, req)
		}
		schema.Required = newRequired

		// Handle 'enum' values with deduplication
		// We need to avoid duplicates in the 'enum' list like ["Basic", "Advanced", "Basic"]
		enumSet := make(map[interface{}]struct{})
		for _, enumVal := range schema.Enum { // Existing enum values
			enumSet[enumVal] = struct{}{}
		}
		for _, enumVal := range mergedEnum { // Merged enum values from allOf
			enumSet[enumVal] = struct{}{}
		}
		newEnum := make([]interface{}, 0, len(enumSet))
		for enumVal := range enumSet {
			newEnum = append(newEnum, enumVal)
		}
		schema.Enum = newEnum

		// Clear AllOf field after merging
		schema.AllOf = nil
	}

	// Process object properties recursively
	for _, prop := range schema.Properties {
		if err := prepareSchemaForCRDWithVisited(ctx, prop.Schema, guard, visited, depth+1); err != nil {
			return fmt.Errorf("failed to process property '%s': %w", prop.Name, err)
		}
	}

	return nil
}

// convertNumberToInteger converts "number" types to "integer" types for K8s CRD compatibility.
func convertNumberToInteger(schema *Schema) {
	if schema == nil {
		return
	}
	for i, t := range schema.Type {
		if t == "number" {
			schema.Type[i] = "integer"
		}
	}
}

// schemaToMap converts our domain-specific Schema object into a map[string]interface{}
// suitable for JSON marshalling.
// It handles circular references to prevent stack overflow.
func schemaToMap(schema *Schema, config *GeneratorConfig) (map[string]interface{}, error) {
	if schema == nil {
		return nil, nil
	}

	guard := safety.NewRecursionGuard(config.MaxRecursionDepth, config.MaxRecursionNodes, config.RecursionTimeout)
	ctx, cancel := guard.WithContext()
	defer cancel()

	return schemaToMapWithVisited(ctx, schema, guard, make(map[*Schema]map[string]interface{}), 0)
}

// schemaToMapWithVisited is the internal implementation that tracks visited schemas
// to handle circular references by creating reference placeholders instead of preserving cycles.
func schemaToMapWithVisited(
	ctx context.Context,
	schema *Schema,
	guard *safety.RecursionGuard,
	visited map[*Schema]map[string]interface{},
	depth int,
) (map[string]interface{}, error) {
	if schema == nil {
		return nil, nil
	}

	// Check recursion limits
	if err := guard.Check(ctx, depth); err != nil {
		log.Printf("Schema to map conversion recursion aborted at depth %d: %v", depth, err)
		// Return a simple reference placeholder to break the cycle
		return map[string]interface{}{
			"type":        "object",
			"description": "Circular reference detected - processing aborted",
		}, nil
	}

	// Return existing map if we've already started processing this schema
	if _, exists := visited[schema]; exists {
		// Instead of returning the same reference (which could cause JSON marshalling issues),
		// return a reference placeholder
		return map[string]interface{}{
			"type":        "object",
			"description": "Circular reference",
		}, nil
	}

	// Create new map and register it before processing to handle circular references
	m := make(map[string]interface{})
	visited[schema] = m

	// Gracefully handle any panics during conversion
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Schema to map conversion panic at depth %d: %v", depth, r)
		}
	}()

	// Process type field
	if len(schema.Type) > 0 {
		if len(schema.Type) == 1 {
			m["type"] = schema.Type[0]
		} else {
			m["type"] = schema.Type
		}
	}

	// Process optional string fields
	if schema.Description != "" {
		m["description"] = schema.Description
	}

	// Process required fields
	if len(schema.Required) > 0 {
		m["required"] = schema.Required
	}

	// Process default value
	if schema.Default != nil {
		m["default"] = schema.Default
	}

	// Process additional properties
	if schema.AdditionalProperties {
		m["additionalProperties"] = true
	}

	// Process max properties
	if schema.MaxProperties > 0 {
		m["maxProperties"] = schema.MaxProperties
	}

	// Process object properties
	if len(schema.Properties) > 0 {
		props := make(map[string]interface{})
		for _, p := range schema.Properties {
			propMap, err := schemaToMapWithVisited(ctx, p.Schema, guard, visited, depth+1)
			if err != nil {
				return nil, fmt.Errorf("failed to convert property '%s': %w", p.Name, err)
			}
			if propMap != nil {
				props[p.Name] = propMap
			}
		}
		if len(props) > 0 {
			m["properties"] = props
		}
	}

	// Process array items
	if schema.Items != nil {
		itemsMap, err := schemaToMapWithVisited(ctx, schema.Items, guard, visited, depth+1)
		if err != nil {
			return nil, fmt.Errorf("failed to convert items schema: %w", err)
		}
		if itemsMap != nil {
			m["items"] = itemsMap
		}
	}

	// Process AllOf
	// In theory, AllOf should have been merged already during CRD preparation (`prepareSchemaForCRD` function).
	// And the `AllOf` field should be empty after that.
	// Therefore no `AllOf` field should remain at this point.
	// Kept here for safety. TODO: consider removing this block.
	if len(schema.AllOf) > 0 {
		// consider adding a log here to indicate unexpected AllOf presence
		//log.Printf("[UNEXPTECTED] Processing allOf inside schemaToMapWithVisited at depth %d", depth)
		//log.Printf("[UNEXPTECTED] Schema info: Type=%v, Description=%q, Properties=%d", schema.Type, schema.Description, len(schema.Properties))
		allOfList := make([]interface{}, 0, len(schema.AllOf))
		for i, s := range schema.AllOf {
			allOfMap, err := schemaToMapWithVisited(ctx, s, guard, visited, depth+1)
			if err != nil {
				return nil, fmt.Errorf("failed to convert allOf item %d: %w", i, err)
			}
			if allOfMap != nil {
				allOfList = append(allOfList, allOfMap)
			}
		}
		if len(allOfList) > 0 {
			m["allOf"] = allOfList
		}
	}

	// Process enum values
	if len(schema.Enum) > 0 {
		m["enum"] = schema.Enum
	}

	// Process extensions
	if len(schema.Extensions) > 0 {
		for k, v := range schema.Extensions {
			m[k] = v
		}
	}

	return m, nil
}

// GenerateJsonSchema converts a domain-specific Schema object into a JSON schema byte slice.
func GenerateJsonSchema(schema *Schema, config *GeneratorConfig) ([]byte, error) {
	schemaMap, err := schemaToMap(schema, config)
	if err != nil {
		return nil, fmt.Errorf("failed to convert schema to map: %w", err)
	}

	if schemaMap == nil {
		return []byte("null"), nil
	}

	return json.MarshalIndent(schemaMap, "", "  ")
}
