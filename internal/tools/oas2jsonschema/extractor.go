package oas2jsonschema

import (
	"fmt"
	"strings"
)

func (g *OASSchemaGenerator) findParameterInOAS(field ConfigurationField) (*ParameterInfo, error) {
	for _, verb := range g.resourceConfig.Verbs {
		if verb.Action == field.FromRestDefinition.Action {
			path, ok := g.doc.FindPath(verb.Path)
			if !ok {
				continue
			}
			ops := path.GetOperations()
			op, ok := ops[strings.ToLower(verb.Method)]
			if !ok {
				continue
			}
			for _, param := range op.GetParameters() {
				if param.Name == field.FromOpenAPI.Name && param.In == field.FromOpenAPI.In {
					return &param, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("parameter '%s' in '%s' not found for action '%s'", field.FromOpenAPI.Name, field.FromOpenAPI.In, field.FromRestDefinition.Action)
}

// getBaseSchemaForSpec returns the base schema for the spec, which is the request body of the 'create' action.
// TODO: what about no create action but only update? TO BE DISCUSSED
// maybe this could be configured in the GeneratorConfig
func (g *OASSchemaGenerator) getBaseSchemaForSpec() (*Schema, error) {
	for _, verb := range g.resourceConfig.Verbs {
		if verb.Action != ActionCreate { // Right now we hardcode the action to 'create'
			continue
		}
		path, ok := g.doc.FindPath(verb.Path)
		if !ok {
			return nil, fmt.Errorf("path '%s' not found in OpenAPI spec", verb.Path)
		}
		ops := path.GetOperations()
		op, ok := ops[strings.ToLower(verb.Method)]
		if !ok {
			return nil, fmt.Errorf("operation '%s' not found for path '%s'", verb.Method, verb.Path)
		}

		rb := op.GetRequestBody()
		for _, mimeType := range g.generatorConfig.AcceptedMIMETypes {
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

// getBaseSchemaForStatus returns the base schema for the status, which is the response body of the 'get' or 'findby' action.
// TODO: what about no get/findby action but only update? TO BE DISCUSSED
// maybe this could be configured in the GeneratorConfig
func (g *OASSchemaGenerator) getBaseSchemaForStatus() (*Schema, error) {
	actions := []string{ActionGet, ActionFindBy}
	for _, action := range actions {
		schema, err := ExtractSchemaForAction(g.doc, g.resourceConfig.Verbs, action, g.generatorConfig)
		if err != nil {
			return nil, err
		}
		if schema != nil {
			return schema, nil
		}
	}
	return nil, nil
}

func ExtractSchemaForAction(doc OASDocument, verbs []Verb, targetAction string, config *GeneratorConfig) (*Schema, error) {
	var verbFound bool
	for _, verb := range verbs {
		if !strings.EqualFold(verb.Action, targetAction) {
			continue
		}
		verbFound = true

		path, ok := doc.FindPath(verb.Path)
		if !ok {
			return nil, fmt.Errorf("path '%s' not found in OAS document", verb.Path)
		}

		ops := path.GetOperations()
		op, ok := ops[strings.ToLower(verb.Method)]
		if !ok {
			return nil, fmt.Errorf("method '%s' not found for path '%s'", verb.Method, verb.Path)
		}

		responses := op.GetResponses()
		if responses == nil {
			continue // Or return an error if responses are expected
		}

		for _, code := range config.SuccessCodes {
			resp, ok := responses[code]
			if !ok {
				continue
			}

			for _, mimeType := range config.AcceptedMIMETypes {
				schema, ok := resp.Content[mimeType]
				if !ok || schema == nil {
					continue
				}

				// If a schema is found, return it immediately.
				if strings.EqualFold(targetAction, ActionFindBy) && schema.Items != nil {
					return schema.Items, nil
				}
				return schema, nil
			}
		}
	}

	if !verbFound {
		return nil, fmt.Errorf("action '%s' not defined in resource verbs", targetAction)
	}

	return nil, fmt.Errorf("no suitable response schema found for action '%s'", targetAction)
}
