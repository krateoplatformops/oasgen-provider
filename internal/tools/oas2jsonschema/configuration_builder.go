package oas2jsonschema

import (
	"fmt"
	"reflect"

	"github.com/krateoplatformops/oasgen-provider/internal/tools/text"
)

// Note: currently the Configuration CRD has no status subresource.

// BuildConfigurationSchema builds the spec schema for the Configuration CRD.
func (g *OASSchemaGenerator) BuildConfigurationSchema() ([]byte, error) {
	// If there are no configuration fields and no security schemes, no configuration CRD is needed.
	if len(g.resourceConfig.ConfigurationFields) == 0 && len(g.doc.SecuritySchemes()) == 0 {
		// TODO: add logging / warning here
		return nil, nil
	}

	rootSchema := &Schema{
		Type:       []string{"object"},
		Properties: []Property{},
	}

	// A map to hold the schemas for each parameter type (path, query, etc.)
	paramTypeSchemas := make(map[string]*Schema)

	for _, field := range g.resourceConfig.ConfigurationFields {
		param, err := g.findParameterInOAS(field)
		if err != nil {
			// Consider logging a warning here
			continue
		}

		// Ensure the top-level schema for the parameter type (e.g., "query") exists.
		if _, ok := paramTypeSchemas[param.In]; !ok {
			paramTypeSchemas[param.In] = &Schema{Type: []string{"object"}, Properties: []Property{}}
		}
		paramTypeSchema := paramTypeSchemas[param.In]

		// Ensure the schema for the action (e.g., "get") exists.
		var actionSchema *Schema
		found := false
		for i := range paramTypeSchema.Properties {
			if paramTypeSchema.Properties[i].Name == field.FromRestDefinition.Action {
				actionSchema = paramTypeSchema.Properties[i].Schema
				found = true
				break
			}
		}
		if !found {
			actionSchema = &Schema{Type: []string{"object"}, Properties: []Property{}}
			paramTypeSchema.Properties = append(paramTypeSchema.Properties, Property{
				Name:   field.FromRestDefinition.Action,
				Schema: actionSchema,
			})
		}

		// Add the parameter's schema to the action schema.
		actionSchema.Properties = append(actionSchema.Properties, Property{Name: param.Name, Schema: param.Schema})
	}

	// Add the populated parameter type schemas to the root schema.
	for paramType, schema := range paramTypeSchemas {
		rootSchema.Properties = append(rootSchema.Properties, Property{Name: paramType, Schema: schema})
	}

	// Add authentication into this schema.
	authMethodsSchemas, err := g.buildAuthMethodsSchemaMap()
	if err != nil {
		return nil, fmt.Errorf("could not generate auth schemas for configuration: %w", err)
	}
	if len(authMethodsSchemas) > 0 {
		addAuthMethods(rootSchema, authMethodsSchemas)
	}

	return GenerateJsonSchema(rootSchema)
}

// buildAuthMethodsSchemaMap generates the JSON schemas for the authentication methods.
func (g *OASSchemaGenerator) buildAuthMethodsSchemaMap() (map[string]*Schema, error) {
	schemaMap := make(map[string]*Schema)
	for _, secScheme := range g.doc.SecuritySchemes() {
		authSchema, err := createSchemaForSecurityScheme(secScheme)
		if err != nil {
			// Skip unsupported security schemes
			// TODO: add logging here
			continue
		}
		schemaMap[secScheme.Name] = authSchema
	}
	return schemaMap, nil
}

// addAuthMethods adds the authenticationMethods property to the configuration schema.
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
	schema.Properties = append(schema.Properties, Property{Name: "authenticationMethods", Schema: authMethodsSchema})
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
