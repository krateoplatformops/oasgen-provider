package oas2jsonschema

import (
	"context"
	"fmt"
	"strings"

	pathparsing "github.com/krateoplatformops/oasgen-provider/internal/tools/pathparsing"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/safety"
)

// BuildSpecSchema generates the complete spec schema for a given resource.
// It returns the schema as a byte slice with a list of warnings (non-fatal errors as a slice of errors).
// It returns a fatal error if the schema generation fails.
func (g *OASSchemaGenerator) BuildSpecSchema() ([]byte, []error, error) {
	var warnings []error

	// Create the base schema for the spec
	// which is the request body of the 'create' action.
	baseSchema, err := g.getBaseSchemaForSpec()
	if err != nil {
		return nil, nil, fmt.Errorf("could not determine base schema for spec: %w", err)
	}

	// Add parameters to the spec schema.
	warnings = append(warnings, g.addParametersToSpec(baseSchema)...)

	// Add identifiers to the spec schema, if configured.
	// Kept for legacy reasons, disabled by default.
	if g.generatorConfig.IncludeIdentifiersInSpec {
		addIdentifiersToSpec(baseSchema, g.resourceConfig.Identifiers)
	}

	// Schema preparation for CRD compatibility.
	if err := prepareSchemaForCRD(baseSchema, g.generatorConfig); err != nil {
		return nil, warnings, fmt.Errorf("could not prepare spec schema for CRD: %w", err)
	}

	// If the resource has configuration fields, add a reference to the configuration schema.
	// This is done only if there are configuration fields or security schemes defined.
	if len(g.resourceConfig.ConfigurationFields) > 0 || len(g.doc.SecuritySchemes()) > 0 {
		addConfigurationRefToSpec(baseSchema)
	}

	// Remove configured fields from the spec schema.
	warnings = append(warnings, g.removeConfiguredFields(baseSchema)...)

	// Remove excluded fields from the spec schema.
	warnings = append(warnings, g.removeExcludedSpecFields(baseSchema)...)

	// Convert the schema to JSON schema format.
	byteSchema, err := GenerateJsonSchema(baseSchema, g.generatorConfig)
	if err != nil {
		return nil, warnings, fmt.Errorf("could not generate final JSON schema: %w", err)
	}

	//log.Printf("Generated spec schema: %s", string(byteSchema))

	return byteSchema, warnings, nil
}

// addParametersToSpec adds the parameters from all verbs to the schema.
// Assumption: it adds parameters at the root level of the spec schema and does not support nested parameters.
// Nested parameters do not make sense in the context of path/query/header/cookie parameters.
func (g *OASSchemaGenerator) addParametersToSpec(schema *Schema) []error {
	var warnings []error

	uniqueParams := make(map[string]struct{})

	// Internal helper function to check if property already exists in schema
	propertyExists := func(name string) bool {
		for _, prop := range schema.Properties {
			if prop.Name == name {
				return true
			}
		}
		return false
	}

	for _, verb := range g.resourceConfig.Verbs {

		// 1. Path lookup
		path, ok := g.doc.FindPath(verb.Path)
		if !ok {
			warnings = append(warnings, SchemaGenerationError{Code: CodePathNotFound, Message: fmt.Sprintf("path '%s' set in RestDefinition not found in OAS", verb.Path)})
			continue
		}

		// 2. Operation lookup
		ops := path.GetOperations()
		op, ok := ops[strings.ToLower(verb.Method)]
		if !ok {
			continue
		}

		// 3. Parameters lookup
		for _, param := range op.GetParameters() {
			// if is authorization header, skip it as it is managed by the Configuration CR within the authentication section.
			if isAuthorizationHeader(param) {
				//fmt.Printf("Skipping authorization header: %s\n", param.Name)
				continue
			}

			// Add parameter to spec only if not already present in uniqueParams AND not in existing base schema
			// Therefore we give precedence to base schema properties over parameters.
			if _, exists := uniqueParams[param.Name]; !exists && !propertyExists(param.Name) {
				// Deep copy the schema to avoid modifying the original object, which might be shared.
				schemaCopy := param.Schema.deepCopy()

				if param.Description == "" {
					schemaCopy.Description = fmt.Sprintf("PARAMETER: %s", param.In)
				} else {
					schemaCopy.Description = fmt.Sprintf("PARAMETER: %s - %s", param.In, param.Description)
				}
				schema.Properties = append(schema.Properties, Property{Name: param.Name, Schema: schemaCopy})
				if param.Required {
					schema.Required = append(schema.Required, param.Name)
				}

				uniqueParams[param.Name] = struct{}{}
			}
		}
	}

	return warnings
}

// addConfigurationRefToSpec adds the `configurationRef` property to the schema (at the root level of spec).
func addConfigurationRefToSpec(schema *Schema) {
	configRefSchema := &Schema{
		Type:        []string{"object"},
		Description: "A reference to the Configuration CR that holds all the needed configuration for this resource. OASGen Provider added this field automatically.",
		Properties: []Property{
			{Name: "name", Schema: &Schema{Type: []string{"string"}}},
			{Name: "namespace", Schema: &Schema{Type: []string{"string"}, Description: "Namespace of the referenced Configuration CR. If not provided, the same namespace will be used."}},
		},
		Required: []string{"name"}, // Namespace not required, if namespace is not provided, it is Rest Dynamic Controller's duty to use the same namespace as the resource.
	}
	schema.Properties = append(schema.Properties, Property{Name: "configurationRef", Schema: configRefSchema})
	schema.Required = append(schema.Required, "configurationRef")
}

// addIdentifiersToSpec adds the identifiers to the schema.
// Kept for legacy reasons, disabled by default.
// TODO: this will be removed in future versions.
func addIdentifiersToSpec(schema *Schema, identifiers []string) {
	for _, identifier := range identifiers {
		found := false
		for i, p := range schema.Properties {
			if p.Name == identifier {
				// Field already exists, append to its description
				if p.Schema.Description != "" {
					schema.Properties[i].Schema.Description += " "
				}
				schema.Properties[i].Schema.Description += fmt.Sprintf("(IDENTIFIER: %s)", identifier)
				found = true
				break
			}
		}
		// If the identifier is not found, add it to the schema.
		if !found {
			schema.Properties = append(schema.Properties, Property{
				Name: identifier,
				Schema: &Schema{
					Description: fmt.Sprintf("IDENTIFIER: %s", identifier),
					Type:        []string{"string"},
				},
			})
		}
	}
}

// isAuthorizationHeader checks if the given parameter is an authorization header (case-insensitive).
func isAuthorizationHeader(param ParameterInfo) bool {
	return strings.EqualFold(param.In, "header") && strings.Contains(strings.ToLower(param.Name), "authorization")
}

// removeConfiguredFields removes the fields from the schema that are defined in the configurationFields list.
func (g *OASSchemaGenerator) removeConfiguredFields(schema *Schema) []error {
	if len(g.resourceConfig.ConfigurationFields) == 0 {
		return nil
	}

	var warnings []error
	for _, field := range g.resourceConfig.ConfigurationFields {
		pathSegments, err := pathparsing.ParsePath(field.FromOpenAPI.Name)
		//log.Printf("Removing configured field '%s' with path segments: %v", field.FromOpenAPI.Name, pathSegments)
		if err != nil {
			warnings = append(warnings, SchemaGenerationError{Code: CodeFieldNotFound, Message: fmt.Sprintf("invalid path format for configured field '%s': %v", field.FromOpenAPI.Name, err)})
			continue
		}

		if !g.removeFieldAtPath(schema, pathSegments) {
			warnings = append(warnings, SchemaGenerationError{Code: CodeFieldNotFound, Message: fmt.Sprintf("field '%s' set in configurationFields not found in schema", field.FromOpenAPI.Name)})
		}
	}
	return warnings
}

// removeExcludedSpecFields removes the fields from the schema that are defined in the `excludedSpecFields` list.
func (g *OASSchemaGenerator) removeExcludedSpecFields(schema *Schema) []error {
	if len(g.resourceConfig.ExcludedSpecFields) == 0 {
		return nil
	}

	var warnings []error
	for _, excludedField := range g.resourceConfig.ExcludedSpecFields {
		pathSegments, err := pathparsing.ParsePath(excludedField)
		//log.Printf("Removing excluded field '%s' with path segments: %v", excludedField, pathSegments)
		if err != nil {
			warnings = append(warnings, SchemaGenerationError{Code: CodeFieldNotFound, Message: fmt.Sprintf("invalid path format for excluded field '%s': %v", excludedField, err)})
			continue
		}

		if !g.removeFieldAtPath(schema, pathSegments) {
			warnings = append(warnings, SchemaGenerationError{Code: CodeFieldNotFound, Message: fmt.Sprintf("field '%s' set in excludedSpecFields not found in schema", excludedField)})
		}
	}
	return warnings
}

// removeFieldAtPath is the entry point for removing a nested field from a schema.
// It sets up a recursion guard and calls the recursive implementation.
func (g *OASSchemaGenerator) removeFieldAtPath(schema *Schema, fields []string) bool {
	guard := safety.NewRecursionGuard(g.generatorConfig.MaxRecursionDepth, g.generatorConfig.MaxRecursionNodes, g.generatorConfig.RecursionTimeout)
	ctx, cancel := guard.WithContext()
	defer cancel()

	return g.removeFieldAtPathRec(ctx, schema, fields, guard, 0)
}

// removeFieldAtPathRec recursively traverses the schema and removes the specified nested field represented by the fields slice.
func (g *OASSchemaGenerator) removeFieldAtPathRec(ctx context.Context, schema *Schema, fields []string, guard *safety.RecursionGuard, depth int) bool {
	if schema == nil || len(fields) == 0 {
		return false
	}

	if err := guard.Check(ctx, depth); err != nil {
		return false
	}

	fieldName := fields[0]
	//log.Printf("At depth %d, looking for field '%s' in schema", depth, fieldName)
	remainingFields := fields[1:]
	//log.Printf("Remaining fields to process: %v", remainingFields)

	propIndex := -1
	for i, prop := range schema.Properties {
		if prop.Name == fieldName {
			propIndex = i
			break
		}
	}

	if propIndex == -1 {
		return false // Property not found at this level
	}

	// If there are more segments, recurse into the child schema.
	if len(remainingFields) > 0 {
		return g.removeFieldAtPathRec(ctx, schema.Properties[propIndex].Schema, remainingFields, guard, depth+1)
	}

	// --- Field found at the current level, and it's the target to remove ---

	// Remove the property from the Properties slice.
	//log.Printf("Removing field '%s' from schema at depth %d", fieldName, depth)
	schema.Properties = append(schema.Properties[:propIndex], schema.Properties[propIndex+1:]...)
	if len(schema.Properties) == 0 {
		schema.Properties = []Property{} // Ensure it's an empty slice, not nil
	}

	// Remove the field name from the Required slice, if it exists.
	reqIndex := -1
	for i, req := range schema.Required {
		if req == fieldName {
			reqIndex = i
			break
		}
	}
	if reqIndex != -1 {
		//log.Printf("Removing field '%s' from required fields at depth %d", fieldName, depth)
		schema.Required = append(schema.Required[:reqIndex], schema.Required[reqIndex+1:]...)
		if len(schema.Required) == 0 {
			schema.Required = []string{} // Ensure it's an empty slice, not nil
		}
	}

	//log.Printf("Field '%s' successfully removed", fieldName)

	return true // Field was found and removed.
}
