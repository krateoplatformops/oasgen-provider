package oas2jsonschema

import (
	"fmt"
	"reflect"

	"github.com/krateoplatformops/oasgen-provider/internal/tools/text"
)

// Note: currently the Configuration CRD has no status subresource.

// BuildConfigurationSchema builds the spec schema for the Configuration CRD.
func (g *OASSchemaGenerator) BuildConfigurationSchema() ([]byte, error) {
	if len(g.resourceConfig.ConfigurationFields) == 0 && len(g.doc.SecuritySchemes()) == 0 {
		return nil, nil
	}

	// Root schema for the entire configuration.
	rootSchema := &Schema{
		Type:       []string{"object"},
		Properties: []Property{},
	}

	paramTypeSchemas := make(map[string]*Schema)

	for _, field := range g.resourceConfig.ConfigurationFields {
		param, err := g.findParameterInOAS(field)
		if err != nil {
			// TODO: Consider logging a warning here.
			continue
		}

		// Ensure the top-level schema for the parameter's location (e.g., "query") already exists.
		paramIn := param.In
		if _, ok := paramTypeSchemas[paramIn]; !ok {
			paramTypeSchemas[paramIn] = &Schema{Type: []string{"object"}, Properties: []Property{}}
		}
		paramTypeSchema := paramTypeSchemas[paramIn]

		// Iterate over all actions this configuration field applies to.
		for _, action := range field.FromRestDefinition.Actions {
			var actionSchema *Schema
			found := false
			// Check if a schema for this action (e.g., "get") already exists
			for i := range paramTypeSchema.Properties {
				if paramTypeSchema.Properties[i].Name == action {
					actionSchema = paramTypeSchema.Properties[i].Schema
					found = true
					break
				}
			}
			// If not found, create a new schema for the action (e.g., "get").
			if !found {
				actionSchema = &Schema{Type: []string{"object"}, Properties: []Property{}}
				paramTypeSchema.Properties = append(paramTypeSchema.Properties, Property{
					Name:   action,
					Schema: actionSchema,
				})
			}

			// Add a deep copy of the parameter's schema to the action's schema
			// to prevent issues with shared schema references.
			actionSchema.Properties = append(actionSchema.Properties, Property{Name: param.Name, Schema: param.Schema.deepCopy()})
			if param.Required {
				actionSchema.Required = append(actionSchema.Required, param.Name)
			}
		}
	}

	if len(paramTypeSchemas) > 0 {
		configurationSchema := &Schema{
			Type:       []string{"object"},
			Properties: []Property{},
		}
		for paramType, schema := range paramTypeSchemas {
			configurationSchema.Properties = append(configurationSchema.Properties, Property{Name: paramType, Schema: schema})
		}
		rootSchema.Properties = append(rootSchema.Properties, Property{
			Name:   "configuration",
			Schema: configurationSchema,
		})
	}

	authMethodsSchemas, err := g.buildAuthMethodsSchemaMap()
	if err != nil {
		return nil, fmt.Errorf("could not generate auth schemas for configuration: %w", err)
	}
	if len(authMethodsSchemas) > 0 {
		addAuthMethods(rootSchema, authMethodsSchemas)
	}

	return GenerateJsonSchema(rootSchema, g.generatorConfig)
}

// buildAuthMethodsSchemaMap generates the JSON schemas for the authentication methods.
func (g *OASSchemaGenerator) buildAuthMethodsSchemaMap() (map[string]*Schema, error) {
	schemaMap := make(map[string]*Schema)
	for _, secScheme := range g.doc.SecuritySchemes() {
		authSchema, err := createSchemaForSecurityScheme(secScheme)
		if err != nil {
			// Skip unsupported security schemes
			// TODO: Consider logging a warning here.
			continue
		}
		schemaMap[secScheme.Scheme] = authSchema
	}
	return schemaMap, nil
}

// addAuthMethods adds the `authentication` property to the configuration schema.
func addAuthMethods(schema *Schema, authSchemas map[string]*Schema) {
	authMethodsProps := []Property{}
	for key, authSchema := range authSchemas {
		authMethodsProps = append(authMethodsProps, Property{Name: text.FirstToLower(key), Schema: authSchema})
	}

	authMethodsSchema := &Schema{
		Type:        []string{"object"},
		Description: "The authentication methods available for this API.",
		Properties:  authMethodsProps,
	}
	schema.Properties = append(schema.Properties, Property{Name: "authentication", Schema: authMethodsSchema})
}

// createSchemaForSecurityScheme generates the JSON schema for a given security scheme.
// Note: currently only supports HTTP Basic and Bearer authentication schemes.
// If the security scheme is not supported, it returns an error.
func createSchemaForSecurityScheme(info SecuritySchemeInfo) (*Schema, error) {
	if info.Type == SchemeTypeHTTP && info.Scheme == "basic" {
		return reflectSchema(reflect.TypeOf(BasicAuth{}))
	}

	if info.Type == SchemeTypeHTTP && info.Scheme == "bearer" {
		return reflectSchema(reflect.TypeOf(BearerAuth{}))
	}

	return nil, fmt.Errorf("unsupported security scheme type: %s", info.Type)
}
