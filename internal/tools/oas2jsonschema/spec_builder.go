package oas2jsonschema

import (
	"fmt"

	"github.com/krateoplatformops/oasgen-provider/internal/tools/text"
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

	// Create a lookup set of configured fields for efficient filtering.
	configuredFieldsSet := g.getConfiguredFieldsSet()

	// Always add a reference to the Configuration CRD.
	addConfigurationRefToSpec(baseSchema)

	// Remove any properties from the base schema that are defined as configuration fields.
	// TODO: understand if this is actually needed
	// If we decide that a field in the request body cannot be a configuration field, then this is not needed.
	filterConfiguredProperties(baseSchema, configuredFieldsSet)

	// Add parameters to the spec schema, excluding those that are defined as configuration fields.
	warnings = append(warnings, g.addParametersToSpec(baseSchema, configuredFieldsSet)...)

	// Add identifiers to the spec schema, if configured.
	if g.generatorConfig.IncludeIdentifiersInSpec {
		addIdentifiersToSpec(baseSchema, g.resourceConfig.Identifiers)
	}

	// Schema preparation for CRD compatibility.
	if err := prepareSchemaForCRD(baseSchema); err != nil {
		return nil, warnings, fmt.Errorf("could not prepare spec schema for CRD: %w", err)
	}

	// Convert the schema to JSON schema format.
	byteSchema, err := GenerateJsonSchema(baseSchema)
	if err != nil {
		return nil, warnings, fmt.Errorf("could not generate final JSON schema: %w", err)
	}

	return byteSchema, warnings, nil
}

// getConfiguredFieldsSet creates a set of configured fields based on the resource configuration.
func (g *OASSchemaGenerator) getConfiguredFieldsSet() map[string]struct{} {
	configuredFields := make(map[string]struct{})
	for _, field := range g.resourceConfig.ConfigurationFields {
		in := field.FromOpenAPI.In
		//if in == "" {
		//	in = "body" // Normalize for consistency with request body properties
		//}
		key := fmt.Sprintf("%s-%s", in, field.FromOpenAPI.Name)
		fmt.Printf("Adding configured field: %s\n", key) // Debugging output
		configuredFields[key] = struct{}{}               // Use a struct for set-like behavior
	}
	return configuredFields
}

// addParametersToSpec adds the parameters from all verbs to the schema,
// excluding those that are defined as configuration fields.
func (g *OASSchemaGenerator) addParametersToSpec(schema *Schema, configuredFields map[string]struct{}) []error {
	var warnings []error

	// Use a map to track unique paths to avoid processing them multiple times
	uniquePaths := make(map[string]struct{})
	for _, verb := range g.resourceConfig.Verbs {
		uniquePaths[verb.Path] = struct{}{}
	}

	for pathStr := range uniquePaths {
		path, ok := g.doc.FindPath(pathStr)
		if !ok {
			warnings = append(warnings, SchemaGenerationError{Code: CodePathNotFound, Message: fmt.Sprintf("path '%s' set in RestDefinition not found in OAS", pathStr)})
			continue
		}
		ops := path.GetOperations()
		for opName, op := range ops {
			for _, param := range op.GetParameters() {
				key := fmt.Sprintf("%s-%s", param.In, param.Name)
				if _, isConfigured := configuredFields[key]; isConfigured {
					continue // Skip parameters that are part of the configuration
				}

				found := false
				for _, p := range schema.Properties {
					if p.Name == param.Name {
						warnings = append(warnings, SchemaGenerationError{Code: CodeDuplicateParameter, Message: fmt.Sprintf("parameter '%s' already exists, skipping", param.Name)})
						found = true
						break
					}
				}
				if !found {
					param.Schema.Description = fmt.Sprintf("PARAMETER: %s, VERB: %s - %s", param.In, text.CapitaliseFirstLetter(opName), param.Description)
					schema.Properties = append(schema.Properties, Property{Name: param.Name, Schema: param.Schema})
				}
			}
		}
	}
	return warnings
}

// addConfigurationRefToSpec adds the configurationRef property to the schema.
func addConfigurationRefToSpec(schema *Schema) {
	configRefSchema := &Schema{
		Type:        []string{"object"},
		Description: "A reference to the Configuration CR that holds all the needed configuration for this resource.",
		Properties: []Property{
			{Name: "name", Schema: &Schema{Type: []string{"string"}}},
			{Name: "namespace", Schema: &Schema{Type: []string{"string"}, Description: "Namespace of the referenced Configuration. If not provided, the same namespace will be used."}},
		},
		Required: []string{"name"}, // If namespace is not provided, it is RDC duty to use the same namespace as the resource.
	}
	schema.Properties = append(schema.Properties, Property{Name: "configurationRef", Schema: configRefSchema})
	schema.Required = append(schema.Required, "configurationRef")
}

// addIdentifiersToSpec adds the identifiers to the schema.
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

// TODO: understand if this is actually needed
// If we decide that a field in the request body cannot be a configuration field, then this is not needed.
func filterConfiguredProperties(schema *Schema, configuredFields map[string]struct{}) {
	var filteredProps []Property
	for _, prop := range schema.Properties {
		key := fmt.Sprintf("body-%s", prop.Name)
		if _, isConfigured := configuredFields[key]; !isConfigured {
			filteredProps = append(filteredProps, prop)
		}
	}
	schema.Properties = filteredProps
}
