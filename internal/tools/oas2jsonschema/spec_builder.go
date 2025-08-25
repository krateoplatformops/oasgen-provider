package oas2jsonschema

import (
	"fmt"
	"strings"
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

	// If the resource has configuration fields, add a reference to the configuration schema.
	// This is done only if there are configuration fields or security schemes defined.
	if len(g.resourceConfig.ConfigurationFields) > 0 || len(g.doc.SecuritySchemes()) > 0 {
		addConfigurationRefToSpec(baseSchema)
	}

	// Create a lookup set of configured fields for efficient filtering.
	configuredFieldsSet := g.getConfiguredFieldsSet()

	// Remove any properties from the base schema that are defined as configuration fields.
	// TODO: understand if this is actually needed
	// If we decide that a field in the request body cannot be a configuration field, then this is not needed.
	//filterConfiguredProperties(baseSchema, configuredFieldsSet)

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

// getConfiguredFieldsSet creates a map where keys are configured field identifiers
// and values are a set of actions they apply to.
func (g *OASSchemaGenerator) getConfiguredFieldsSet() map[string]map[string]struct{} {
	configuredFields := make(map[string]map[string]struct{})
	for _, field := range g.resourceConfig.ConfigurationFields {
		key := fmt.Sprintf("%s-%s", field.FromOpenAPI.In, field.FromOpenAPI.Name)
		if _, ok := configuredFields[key]; !ok {
			configuredFields[key] = make(map[string]struct{})
		}
		for _, action := range field.FromRestDefinition.Actions {
			configuredFields[key][action] = struct{}{}
		}
	}
	return configuredFields
}

// addParametersToSpec adds the parameters from all verbs to the schema,
// excluding those that are defined as configuration fields for that specific verb.
func (g *OASSchemaGenerator) addParametersToSpec(schema *Schema, configuredFields map[string]map[string]struct{}) []error {
	var warnings []error

	uniqueParams := make(map[string]struct{})

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

			// Check if this parameter is configured for the current action.
			key := fmt.Sprintf("%s-%s", param.In, param.Name)
			//fmt.Printf("Checking parameter: %s for action: %s\n", key, verb.Action)
			if actions, ok := configuredFields[key]; ok {
				if _, isConfiguredForAction := actions[verb.Action]; isConfiguredForAction {
					continue // Skip this parameter as it's a configuration field for this action.
				}
			}

			// if is authorizaion header, skip it as it is managed by the configuration CR withing the autehntication section.
			if isAuthorizationHeader(param) {
				//fmt.Printf("Skipping authorization header: %s\n", param.Name)
				continue
			}

			// Add parameter to spec only if not already present.
			if _, exists := uniqueParams[param.Name]; !exists {
				param.Schema.Description = fmt.Sprintf("PARAMETER: %s - %s", param.In, param.Description)
				schema.Properties = append(schema.Properties, Property{Name: param.Name, Schema: param.Schema})
				//fmt.Printf("Adding parameter: %s to spec schema\n", param.Name)
				uniqueParams[param.Name] = struct{}{}
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
//func filterConfiguredProperties(schema *Schema, configuredFields map[string]struct{}) {
//	var filteredProps []Property
//	for _, prop := range schema.Properties {
//		key := fmt.Sprintf("body-%s", prop.Name)
//		if _, isConfigured := configuredFields[key]; !isConfigured {
//			filteredProps = append(filteredProps, prop)
//		}
//	}
//	schema.Properties = filteredProps
//}

// isAuthorizationHeader checks if the given parameter is an authorization header (case-insensitive).
func isAuthorizationHeader(param ParameterInfo) bool {
	return strings.EqualFold(param.In, "header") && strings.Contains(strings.ToLower(param.Name), "authorization")
}
