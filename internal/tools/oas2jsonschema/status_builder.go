package oas2jsonschema

import (
	"context"
	"fmt"

	pathparsing "github.com/krateoplatformops/oasgen-provider/internal/tools/pathparsing"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/safety"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// BuildStatusSchema generates the complete status schema for a given resource.
func (g *OASSchemaGenerator) BuildStatusSchema() ([]byte, []error, error) {
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

	if err := prepareSchemaForCRD(responseSchema, g.generatorConfig); err != nil {
		return nil, warnings, fmt.Errorf("could not prepare status schema for CRD: %w", err)
	}

	statusSchema, buildWarnings := g.composeStatusSchema(allStatusFields, responseSchema)
	warnings = append(warnings, buildWarnings...)

	byteSchema, err := GenerateJsonSchema(statusSchema, g.generatorConfig)
	if err != nil {
		return nil, warnings, fmt.Errorf("could not generate final JSON schema for status: %w", err)
	}

	return byteSchema, warnings, nil
}

// composeStatusSchema builds the status schema by finding nested fields in the response schema
// and constructing a corresponding nested structure in the new status schema.
func (g *OASSchemaGenerator) composeStatusSchema(allStatusFields []string, responseSchema *Schema) (*Schema, []error) {
	var warnings []error
	statusSchema := &Schema{Type: []string{"object"}, Properties: []Property{}}

	for _, fieldName := range allStatusFields {
		pathSegments, err := pathparsing.ParsePath(fieldName)
		if err != nil {
			warnings = append(warnings, SchemaGenerationError{Code: CodeFieldNotFound, Message: fmt.Sprintf("invalid path format for status field '%s': %v", fieldName, err)})
			continue
		}

		// Find the property in the source response schema.
		foundProp, found := g.findPropertyByPath(responseSchema, pathSegments)
		if found {
			// `findPropertyByPath` returns a deep-copied property, so we can use it directly.
			g.addPropertyByPath(statusSchema, pathSegments, foundProp)
		} else {
			// Fallback for fields not found in the response schema.
			warnings = append(warnings, SchemaGenerationError{Code: CodeStatusFieldNotFound, Message: fmt.Sprintf("status field '%s' not found in response, defaulting to string", fieldName)})
			fallbackProp := Property{Name: pathSegments[len(pathSegments)-1], Schema: &Schema{Type: []string{"string"}}} // Fallback to string type
			g.addPropertyByPath(statusSchema, pathSegments, fallbackProp)
		}
	}

	// Iterate over the top-level properties of the status schema and add the `x-crdgen-identifier-name` annotation.
	for i, prop := range statusSchema.Properties {
		if prop.Schema != nil && getPrimaryType(prop.Schema.Type) == "object" {
			if prop.Schema.Extensions == nil {
				prop.Schema.Extensions = make(map[string]interface{})
			}
			prop.Schema.Extensions["x-crdgen-identifier-name"] = "Status" + cases.Title(language.English).String(prop.Name)
			statusSchema.Properties[i] = prop
		}
	}

	return statusSchema, warnings
}

// findPropertyByPath is the public entry point for finding a nested property.
// It sets up a recursion guard, inspired by the pattern in spec_builder.go.
func (g *OASSchemaGenerator) findPropertyByPath(schema *Schema, path []string) (Property, bool) {
	guard := safety.NewRecursionGuard(g.generatorConfig.MaxRecursionDepth, g.generatorConfig.MaxRecursionNodes, g.generatorConfig.RecursionTimeout)
	ctx, cancel := guard.WithContext()
	defer cancel()
	return g.findPropertyByPathRec(ctx, schema, path, guard, 0)
}

// findPropertyByPathRec recursively traverses a schema to find a nested property.
// Returns a deep copy of the found property and true, or an empty property and false if not found.
func (g *OASSchemaGenerator) findPropertyByPathRec(ctx context.Context, schema *Schema, path []string, guard *safety.RecursionGuard, depth int) (Property, bool) {
	if schema == nil || len(path) == 0 || guard.Check(ctx, depth) != nil {
		return Property{}, false
	}

	fieldName := path[0]
	//log.Printf("Processing field: %s", fieldName)
	remainingPath := path[1:]
	//log.Printf("Remaining fields to process: %v", remainingPath)

	for _, prop := range schema.Properties {
		if prop.Name == fieldName {
			if len(remainingPath) == 0 {
				// Return a deep copy
				return Property{
					Name:   prop.Name,
					Schema: prop.Schema.deepCopy(),
				}, true
			}
			if prop.Schema == nil || getPrimaryType(prop.Schema.Type) != "object" {
				// Can't traverse further if it's not an object.
				// E.g., the case of "metadata.nested.leaf" where "nested" is a string.
				continue
			}
			// Continue traversing into the sub-schema.
			return g.findPropertyByPathRec(ctx, prop.Schema, remainingPath, guard, depth+1)
		}
	}

	return Property{}, false // Not found at this level
}

// addPropertyByPath is the public entry point for adding a nested property.
// It sets up a recursion guard for safety.
func (g *OASSchemaGenerator) addPropertyByPath(schema *Schema, path []string, propToAdd Property) {
	guard := safety.NewRecursionGuard(g.generatorConfig.MaxRecursionDepth, g.generatorConfig.MaxRecursionNodes, g.generatorConfig.RecursionTimeout)
	ctx, cancel := guard.WithContext()
	defer cancel()
	g.addPropertyByPathRec(ctx, schema, path, propToAdd, guard, 0)
}

// addPropertyByPathRec recursively builds the nested object structure in a schema
// and adds the target property at the correct location.
func (g *OASSchemaGenerator) addPropertyByPathRec(ctx context.Context, schema *Schema, path []string, propToAdd Property, guard *safety.RecursionGuard, depth int) {
	if schema == nil || len(path) == 0 || guard.Check(ctx, depth) != nil {
		return
	}

	fieldName := path[0]
	remainingPath := path[1:]

	// If this is the last part of the path, add the property here.
	if len(remainingPath) == 0 {
		// Avoid adding duplicates.
		for _, p := range schema.Properties {
			if p.Name == fieldName {
				return
			}
		}
		schema.Properties = append(schema.Properties, propToAdd)
		return
	}

	//  Intermediate path segment. Find or create the next object schema.
	var nextSchema *Schema
	found := false
	for _, p := range schema.Properties {
		if p.Name == fieldName {
			nextSchema = p.Schema
			found = true
			break
		}
	}

	if nextSchema != nil && getPrimaryType(nextSchema.Type) != "object" {
		// Error: expected an object to traverse further, but found a different type.
		// Example: if the path is "metadata.nested.leaf" but "nested" is a string and not an object.
		// So we cannot reach "leaf".
		//log.Printf("Warning: expected object type at '%s' but found type '%v'.", fieldName, nextSchema.Type)
		return
	}

	if !found {
		nextSchema = &Schema{Type: []string{"object"}, Properties: []Property{}}
		schema.Properties = append(schema.Properties, Property{Name: fieldName, Schema: nextSchema})
	}

	// Recurse into the next level.
	g.addPropertyByPathRec(ctx, nextSchema, remainingPath, propToAdd, guard, depth+1)
}
