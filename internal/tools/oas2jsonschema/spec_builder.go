package oas2jsonschema

import (
	"context"
	"fmt"
	"strings"

	"github.com/krateoplatformops/oasgen-provider/internal/tools/safety"
)

// BuildSpecSchema generates the complete spec schema for a given resource.
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
	if g.generatorConfig.IncludeIdentifiersInSpec {
		addIdentifiersToSpec(baseSchema, g.resourceConfig.Identifiers)
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

	// Schema preparation for CRD compatibility.
	if err := prepareSchemaForCRD(baseSchema, g.generatorConfig); err != nil {
		return nil, warnings, fmt.Errorf("could not prepare spec schema for CRD: %w", err)
	}

	//log.Printf("Spec schema AFTER prepareSchemaForCRD: %+v", baseSchema)

	// Convert the schema to JSON schema format.
	byteSchema, err := GenerateJsonSchema(baseSchema, g.generatorConfig)
	if err != nil {
		return nil, warnings, fmt.Errorf("could not generate final JSON schema: %w", err)
	}

	return byteSchema, warnings, nil
}

// addParametersToSpec adds the parameters from all verbs to the schema.
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
				if param.Description == "" {
					param.Schema.Description = fmt.Sprintf("PARAMETER: %s", param.In)
				} else {
					param.Schema.Description = fmt.Sprintf("PARAMETER: %s - %s", param.In, param.Description)
				}
				schema.Properties = append(schema.Properties, Property{Name: param.Name, Schema: param.Schema})
				if param.Required {
					//log.Printf("Adding required path / query parameter: %s\n", param.Name)
					schema.Required = append(schema.Required, param.Name)
				}

				uniqueParams[param.Name] = struct{}{}
			}
		}
	}

	return warnings
}

// addConfigurationRefToSpec adds the `configurationRef` property to the schema.
func addConfigurationRefToSpec(schema *Schema) {
	configRefSchema := &Schema{
		Type:        []string{"object"},
		Description: "A reference to the Configuration CR that holds all the needed configuration for this resource. OASGen Provider added this field automatically.",
		Properties: []Property{
			{Name: "name", Schema: &Schema{Type: []string{"string"}}},
			{Name: "namespace", Schema: &Schema{Type: []string{"string"}, Description: "Namespace of the referenced Configuration CR. If not provided, the same namespace will be used."}},
		},
		Required: []string{"name"}, // If namespace is not provided, it is Rest Dynamic Controller's duty to use the same namespace as the resource.
	}
	schema.Properties = append(schema.Properties, Property{Name: "configurationRef", Schema: configRefSchema})
	schema.Required = append(schema.Required, "configurationRef")
}

// addIdentifiersToSpec adds the identifiers to the schema.
// Kept for legacy reasons, disabled by default.
func addIdentifiersToSpec(schema *Schema, identifiers []string) {
	for _, identifier := range identifiers {
		found := false
		for _, p := range schema.Properties {
			if p.Name == identifier {
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
		//log.Printf("No configurationFields to remove.")
		return nil
	}

	//log.Printf("Removing configurationFields: ")
	//for _, field := range g.resourceConfig.ConfigurationFields {
	//	log.Printf("- %s", field.FromOpenAPI.Name)
	//}

	var warnings []error
	for _, field := range g.resourceConfig.ConfigurationFields {
		if !g.removeFieldAtPath(schema, strings.Split(field.FromOpenAPI.Name, ".")) {
			warnings = append(warnings, SchemaGenerationError{Code: CodeFieldNotFound, Message: fmt.Sprintf("field '%s' set in configurationFields not found in schema", field.FromOpenAPI.Name)})
		}
	}
	return warnings
}

// removeExcludedSpecFields removes the fields from the schema that are defined in the excludedSpecFields list.
func (g *OASSchemaGenerator) removeExcludedSpecFields(schema *Schema) []error {
	if len(g.resourceConfig.ExcludedSpecFields) == 0 {
		//log.Printf("No excludedSpecFields to remove.")
		return nil
	}

	//log.Printf("Removing excludedSpecFields: %v", g.resourceConfig.ExcludedSpecFields)

	var warnings []error
	for _, excludedField := range g.resourceConfig.ExcludedSpecFields {
		if !g.removeFieldAtPath(schema, strings.Split(excludedField, ".")) {
			warnings = append(warnings, SchemaGenerationError{Code: CodeFieldNotFound, Message: fmt.Sprintf("field '%s' set in excludedSpecFields not found in schema", excludedField)})
		}
	}
	return warnings
}

// removeFieldAtPath is the public entry point for removing a nested field from a schema.
// It sets up a recursion guard and calls the recursive implementation.
func (g *OASSchemaGenerator) removeFieldAtPath(schema *Schema, fields []string) bool {

	//log.Printf("Removing field at path: %v", fields)

	guard := safety.NewRecursionGuard(g.generatorConfig.MaxRecursionDepth, g.generatorConfig.MaxRecursionNodes, g.generatorConfig.RecursionTimeout)
	ctx, cancel := guard.WithContext()
	defer cancel()

	//log.Printf("Initial schema properties: %v", schema.Properties)

	return g.removeFieldAtPathRec(ctx, schema, fields, guard, 0)
}

// removeFieldAtPathRec recursively traverses the schema and removes the specified nested field.
func (g *OASSchemaGenerator) removeFieldAtPathRec(ctx context.Context, schema *Schema, fields []string, guard *safety.RecursionGuard, depth int) bool {
	if schema == nil || len(fields) == 0 {
		return false
	}

	if err := guard.Check(ctx, depth); err != nil {
		return false
	}

	fieldName := fields[0]
	//log.Printf("Processing field: %s", fieldName)
	remainingFields := fields[1:]
	//log.Printf("Remaining fields to process: %v", remainingFields)

	var newProperties []Property
	found := false
	for _, prop := range schema.Properties {
		if prop.Name == fieldName {
			if len(remainingFields) > 0 {
				// We are in a nested path, so we recurse.
				// We keep the parent property, but with a potentially modified schema.
				if g.removeFieldAtPathRec(ctx, prop.Schema, remainingFields, guard, depth+1) {
					found = true
				}
				newProperties = append(newProperties, prop)
			} else {
				// This is the field to remove since there are no more remaining fields (last element in the "dot" path).
				// We just don't add it to the new list.
				found = true
			}
		} else {
			newProperties = append(newProperties, prop)
		}
	}
	schema.Properties = newProperties
	//log.Printf("After processing, schema properties are now: %v", schema.Properties)

	// If the field was found and removed, also remove it from the required list
	if found && len(remainingFields) == 0 {
		// Remove from required list
		var newRequired []string
		for _, req := range schema.Required {
			if req != fieldName {
				newRequired = append(newRequired, req)
			}
		}
		schema.Required = newRequired
	}

	return found
}
