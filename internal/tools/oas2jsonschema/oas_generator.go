package oas2jsonschema

import (
	"encoding/json"
	"fmt"

	definitionv1alpha1 "github.com/krateoplatformops/oasgen-provider/apis/restdefinitions/v1alpha1"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/text"
)

// OASSchemaGenerator orchestrates the generation of CRD schemas from an OpenAPI document.
type OASSchemaGenerator struct {
	config *GeneratorConfig
	doc    OASDocument
}

// NewOASSchemaGenerator creates a new, configured OASSchemaGenerator.
func NewOASSchemaGenerator(doc OASDocument, config *GeneratorConfig) *OASSchemaGenerator {
	return &OASSchemaGenerator{
		doc:    doc,
		config: config,
	}
}

// Generate orchestrates the full schema generation process.
func (g *OASSchemaGenerator) Generate(resource definitionv1alpha1.Resource, identifiers []string) (*GenerationResult, error) {
	var allWarnings []error

	specSchema, warnings, err := g.generateSpecSchema(resource, identifiers)
	if err != nil {
		return nil, fmt.Errorf("failed to generate spec schema: %w", err)
	}
	allWarnings = append(allWarnings, warnings...)

	statusSchema, warnings, err := g.generateStatusSchema(resource, identifiers)
	if err != nil {
		// A failure to generate status schema is not considered a fatal error.
		allWarnings = append(allWarnings, fmt.Errorf("failed to generate status schema: %w", err))
	}
	allWarnings = append(allWarnings, warnings...)

	authSchemas, err := g.generateSecuritySchemas()
	if err != nil {
		return nil, fmt.Errorf("failed to generate security schemas: %w", err)
	}

	return &GenerationResult{
		SpecSchema:   specSchema,
		StatusSchema: statusSchema,
		AuthSchemas:  authSchemas,
		Warnings:     allWarnings,
	}, nil
}

// generateSpecSchema generates the complete spec schema for a given resource.
func (g *OASSchemaGenerator) generateSpecSchema(resource definitionv1alpha1.Resource, identifiers []string) ([]byte, []error, error) {
	var warnings []error

	baseSchema, err := g.getBaseSchemaForSpec(resource)
	if err != nil {
		return nil, nil, fmt.Errorf("could not determine base schema for spec: %w", err)
	}

	secSchemas, err := g.generateSecuritySchemas()
	if err != nil {
		return nil, nil, fmt.Errorf("could not generate security schemas: %w", err)
	}
	if len(secSchemas) > 0 {
		addAuthenticationRefs(baseSchema, secSchemas)
	}

	warnings = append(warnings, g.addParametersToSpec(baseSchema, resource)...)
	addIdentifiersToSpec(baseSchema, identifiers)

	if err := prepareSchemaForCRD(baseSchema); err != nil {
		return nil, warnings, fmt.Errorf("could not prepare spec schema for CRD: %w", err)
	}

	byteSchema, err := GenerateJsonSchema(baseSchema)
	if err != nil {
		return nil, warnings, fmt.Errorf("could not generate final JSON schema: %w", err)
	}

	return byteSchema, warnings, nil
}

// generateStatusSchema generates the complete status schema for a given resource.
func (g *OASSchemaGenerator) generateStatusSchema(resource definitionv1alpha1.Resource, identifiers []string) ([]byte, []error, error) {
	var warnings []error

	allStatusFields := append(identifiers, resource.AdditionalStatusFields...)
	if len(allStatusFields) == 0 {
		return nil, []error{fmt.Errorf("no identifiers or additional status fields defined, skipping status schema generation")}, nil
	}

	responseSchema, err := g.getBaseSchemaForStatus(resource.VerbsDescription)
	if err != nil {
		warnings = append(warnings, fmt.Errorf("schema validation warning: %w", err))
	}
	if responseSchema == nil {
		warnings = append(warnings, fmt.Errorf("could not find a GET or FINDBY response schema for status generation"))
	}

	statusSchema, buildWarnings := g.buildStatusSchema(allStatusFields, responseSchema)
	warnings = append(warnings, buildWarnings...)

	if err := prepareSchemaForCRD(statusSchema); err != nil {
		return nil, warnings, fmt.Errorf("could not prepare status schema for CRD: %w", err)
	}

	byteSchema, err := GenerateJsonSchema(statusSchema)
	if err != nil {
		return nil, warnings, fmt.Errorf("could not generate final JSON schema for status: %w", err)
	}

	return byteSchema, warnings, nil
}

func (g *OASSchemaGenerator) getBaseSchemaForSpec(resource definitionv1alpha1.Resource) (*Schema, error) {
	for _, verb := range resource.VerbsDescription {
		if verb.Action != ActionCreate {
			continue
		}
		path, ok := g.doc.FindPath(verb.Path)
		if !ok {
			return nil, fmt.Errorf("path '%s' not found in OpenAPI spec", verb.Path)
		}
		ops := path.GetOperations()
		op, ok := ops[verb.Method]
		if !ok {
			return nil, fmt.Errorf("operation '%s' not found for path '%s'", verb.Method, verb.Path)
		}

		rb := op.GetRequestBody()
		for _, mimeType := range g.config.AcceptedMIMETypes {
			if schema, ok := rb.Content[mimeType]; ok {
				if getPrimaryType(schema.Type) == "array" {
					schema.Properties = append(schema.Properties, Property{Name: "items", Schema: &Schema{Type: []string{"array"}, Items: schema.Items}})
					schema.Type = []string{"object"}
				}
				return schema, nil
			}
		}
	}
	return &Schema{}, nil
}

func (g *OASSchemaGenerator) generateSecuritySchemas() (map[string][]byte, error) {
	secByteSchema := make(map[string][]byte)
	for _, secScheme := range g.doc.SecuritySchemes() {
		authSchemaName, err := g.generateAuthSchemaName(secScheme)
		if err != nil {
			// Skip unsupported security schemes
			continue
		}

		authSchema, err := g.generateAuthSchema(secScheme)
		if err != nil {
			// Skip unsupported security schemes
			continue
		}

		byteSchema, err := GenerateJsonSchema(authSchema)
		if err != nil {
			return nil, fmt.Errorf("generating auth schema for '%s': %w", secScheme.Name, err)
		}
		secByteSchema[authSchemaName] = byteSchema
	}
	return secByteSchema, nil
}

func (g *OASSchemaGenerator) generateAuthSchemaName(info SecuritySchemeInfo) (string, error) {
	if info.Type == SchemeTypeHTTP && info.Scheme == "basic" {
		return "BasicAuth", nil
	}
	if info.Type == SchemeTypeHTTP && info.Scheme == "bearer" {
		return "BearerAuth", nil
	}
	return "", fmt.Errorf("unsupported security scheme type: %s", info.Type)
}

func (g *OASSchemaGenerator) generateAuthSchema(info SecuritySchemeInfo) (*Schema, error) {
	secretKeySelectorSchema := &Schema{
		Type: []string{"object"},
		Properties: []Property{
			{Name: "name", Schema: &Schema{Type: []string{"string"}}},
			{Name: "key", Schema: &Schema{Type: []string{"string"}}},
		},
		Required: []string{"name", "key"},
	}

	if info.Type == SchemeTypeHTTP && info.Scheme == "basic" {
		return &Schema{
			Type: []string{"object"},
			Properties: []Property{
				{Name: "username", Schema: &Schema{Type: []string{"string"}}},
				{Name: "passwordRef", Schema: secretKeySelectorSchema},
			},
			Required: []string{"username", "passwordRef"},
		}, nil
	}

	if info.Type == SchemeTypeHTTP && info.Scheme == "bearer" {
		return &Schema{
			Type: []string{"object"},
			Properties: []Property{
				{Name: "tokenRef", Schema: secretKeySelectorSchema},
			},
			Required: []string{"tokenRef"},
		}, nil
	}

	return nil, fmt.Errorf("unsupported security scheme type: %s", info.Type)
}

func addAuthenticationRefs(schema *Schema, secSchemas map[string][]byte) {
	var authRefsProps []Property
	for key := range secSchemas {
		authRefsProps = append(authRefsProps, Property{Name: fmt.Sprintf("%sRef", text.FirstToLower(key)), Schema: &Schema{Type: []string{"string"}}})
	}
	authRefsSchema := &Schema{
		Type:        []string{"object"},
		Description: "AuthenticationRefs represent the reference to a CR containing the authentication information. One authentication method must be set.",
		Properties:  authRefsProps,
	}
	schema.Properties = append(schema.Properties, Property{Name: "authenticationRefs", Schema: authRefsSchema})
	schema.Required = append(schema.Required, "authenticationRefs")
}

func (g *OASSchemaGenerator) addParametersToSpec(schema *Schema, resource definitionv1alpha1.Resource) []error {
	var warnings []error
	for _, verb := range resource.VerbsDescription {
		path, ok := g.doc.FindPath(verb.Path)
		if !ok {
			warnings = append(warnings, fmt.Errorf("path '%s' in RestDefinition not found", verb.Path))
			continue
		}
		ops := path.GetOperations()
		for opName, op := range ops {
			for _, param := range op.GetParameters() {
				found := false
				for _, p := range schema.Properties {
					if p.Name == param.Name {
						warnings = append(warnings, fmt.Errorf("parameter '%s' already exists, skipping", param.Name))
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

func addIdentifiersToSpec(schema *Schema, identifiers []string) {
	for _, identifier := range identifiers {
		found := false
		for _, p := range schema.Properties {
			if p.Name == identifier {
				found = true
				break
			}
		}
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

func (g *OASSchemaGenerator) getBaseSchemaForStatus(verbs []definitionv1alpha1.VerbsDescription) (*Schema, error) {
	actions := []string{ActionGet, ActionFindBy}
	for _, action := range actions {
		schema, err := extractSchemaForAction(g.doc, verbs, action, g.config)
		if err != nil {
			return nil, err
		}
		if schema != nil {
			return schema, nil
		}
	}
	return nil, nil
}

func (g *OASSchemaGenerator) buildStatusSchema(allStatusFields []string, responseSchema *Schema) (*Schema, []error) {
	var warnings []error
	var props []Property
	for _, fieldName := range allStatusFields {
		found := false
		if responseSchema != nil {
			for _, p := range responseSchema.Properties {
				if p.Name == fieldName {
					props = append(props, p)
					found = true
					break
				}
			}
		}
		if !found {
			warnings = append(warnings, fmt.Errorf("status field '%s' not found in response, defaulting to string", fieldName))
			props = append(props, Property{Name: fieldName, Schema: &Schema{Type: []string{"string"}}})
		}
	}
	return &Schema{Type: []string{"object"}, Properties: props}, warnings
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
		for _, prop := range allOfSchema.Properties {
			schema.Properties = append(schema.Properties, prop)
		}
	}

	return nil
}

func convertNumberToInteger(schema *Schema) {
	for i, t := range schema.Type {
		if t == "number" {
			schema.Type[i] = "integer"
		}
	}
}
