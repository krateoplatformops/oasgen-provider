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
		// Fatal error
		return nil, fmt.Errorf("failed to generate spec schema: %w", err)
	}
	generationWarnings = append(generationWarnings, warnings...)

	// Generate Status Schema
	statusSchema, warnings, err := g.BuildStatusSchema()
	if err != nil {
		// A failure to generate status schema is currently not considered a fatal error for compatibility reasons
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
			// Fatal error since configuration schema is required in these cases
			return nil, fmt.Errorf("failed to generate configuration schema: %w", err)
		}
	}

	// Annotate schemas to disambiguate duplicate field names.
	// This is necessary due to the underlying tool for generating CRDs.
	finalSpec, finalStatus, err := annotateSchemas(specSchema, statusSchema, "x-crdgen-identifier-name")
	if err != nil {
		return nil, fmt.Errorf("failed to annotate schemas with 'x-crdgen-identifier-name': %w", err)
	}

	// Annotate configuration schema if exists, to disambiguate duplicate field names.
	// This is necessary due to the underlying tool for generating CRDs.
	var finalConfig []byte
	if len(configurationSchema) > 0 {
		var err error
		finalConfig, _, err = annotateSchemas(configurationSchema, nil, "x-crdgen-identifier-name")
		if err != nil {
			return nil, fmt.Errorf("failed to annotate configuration schema with 'x-crdgen-identifier-name': %w", err)
		}
	}

	// TODO: consider to log the generated spec schema for debugging purposes (we need the logger setup)
	log.Print("======= Final Spec Schema =======")
	log.Print(string(finalSpec))
	log.Print("======= End Spec Schema =======")

	////// TODO: consider to log the generated status schema for debugging purposes (we need the logger setup)
	log.Print("======= Final Status Schema  =======")
	log.Print(string(finalStatus))
	log.Print("======= End Status Schema =======")

	////// TODO: consider to log the generated configuration schema for debugging purposes (we need the logger setup)
	log.Print("Final configuration schema")
	if finalConfig != nil {
		log.Print(string(finalConfig))
	}
	log.Print("======= End Configuration Schema =======")

	return &GenerationResult{
		SpecSchema:          finalSpec,
		StatusSchema:        finalStatus,
		ConfigurationSchema: configurationSchema,
		GenerationWarnings:  generationWarnings,
		ValidationWarnings:  validationWarnings,
	}, nil
}
