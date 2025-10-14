package oas2jsonschema

import (
	"fmt"
	"log"
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

// Generate orchestrates the full schema (spec + status) generation process along with configuration schema if needed.
func (g *OASSchemaGenerator) Generate() (*GenerationResult, error) {
	var generationWarnings []error

	// Generate Spec Schema
	specSchema, warnings, err := g.BuildSpecSchema()
	if err != nil {
		// fatal error
		return nil, fmt.Errorf("failed to generate spec schema: %w", err)
	}
	generationWarnings = append(generationWarnings, warnings...)

	// Generate Status Schema
	statusSchema, warnings, err := g.BuildStatusSchema()
	if err != nil {
		// A failure to generate status schema is currently not considered a fatal error.
		generationWarnings = append(generationWarnings, fmt.Errorf("failed to generate status schema: %w", err))
	}
	generationWarnings = append(generationWarnings, warnings...)

	// Validate Status Schema
	validationWarnings := ValidateSchemas(g.doc, g.resourceConfig.Verbs, g.generatorConfig)

	// Generate Configuration Schema if needed
	var configurationSchema []byte
	if len(g.resourceConfig.ConfigurationFields) > 0 || len(g.doc.SecuritySchemes()) > 0 {
		var err error
		configurationSchema, err = g.BuildConfigurationSchema()
		if err != nil {
			// fatal error
			return nil, fmt.Errorf("failed to generate configuration schema: %w", err)
		}
	}

	// consider to log the generated spec schema for debugging purposes
	log.Print("======= Final Spec Schema =======")
	log.Print(string(specSchema))
	log.Print("======= End Spec Schema =======")

	// consider to log the generated status schema for debugging purposes
	log.Print("======= Final Status Schema  =======")
	log.Print(string(statusSchema))
	log.Print("======= End Status Schema =======")

	log.Printf("Final configuration schema")
	if configurationSchema != nil {
		log.Print(string(configurationSchema))
	}
	log.Print("======= End Configuration Schema =======")

	return &GenerationResult{
		SpecSchema:          specSchema,
		StatusSchema:        statusSchema,
		ConfigurationSchema: configurationSchema,
		GenerationWarnings:  generationWarnings,
		ValidationWarnings:  validationWarnings,
	}, nil
}
