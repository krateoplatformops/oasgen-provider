package oas2jsonschema

import (
	"fmt"
)

// OASSchemaGenerator orchestrates the generation of CRD schemas from an OpenAPI document.
type OASSchemaGenerator struct {
	generatorConfig *GeneratorConfig
	resourceConfig  *ResourceConfig
	doc             OASDocument
}

// NewOASSchemaGenerator creates a new, configured OASSchemaGenerator.
func NewOASSchemaGenerator(doc OASDocument, config *GeneratorConfig, resourceConfig *ResourceConfig) *OASSchemaGenerator {
	return &OASSchemaGenerator{
		generatorConfig: config,
		resourceConfig:  resourceConfig,
		doc:             doc,
	}
}

// Note on convention used in this package:
// - Methods: stateful operations that use the generator's state
// - Functions: stateless operations that do not rely on the generator's state

// Generate orchestrates the full schema (spec + status) generation process along with configuration schema if needed.
func (g *OASSchemaGenerator) Generate() (*GenerationResult, error) {
	var generationWarnings []error

	specSchema, warnings, err := g.BuildSpecSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to generate spec schema: %w", err)
	}
	generationWarnings = append(generationWarnings, warnings...)

	statusSchema, warnings, err := g.BuildStatusSchema()
	if err != nil {
		// A failure to generate status schema is not considered a fatal error. (TO BE DISCUSSED)
		generationWarnings = append(generationWarnings, fmt.Errorf("failed to generate status schema: %w", err))
	}
	generationWarnings = append(generationWarnings, warnings...)

	validationWarnings := ValidateSchemas(g.doc, g.resourceConfig.Verbs, g.generatorConfig)

	var configurationSchema []byte
	if len(g.resourceConfig.ConfigurationFields) > 0 || len(g.doc.SecuritySchemes()) > 0 {
		var err error
		configurationSchema, err = g.BuildConfigurationSchema()
		if err != nil {
			return nil, fmt.Errorf("failed to generate configuration schema: %w", err)
		}
	}

	return &GenerationResult{
		SpecSchema:          specSchema,
		StatusSchema:        statusSchema,
		ConfigurationSchema: configurationSchema,
		GenerationWarnings:  generationWarnings,
		ValidationWarnings:  validationWarnings,
	}, nil
}
