package oas2jsonschema

import (
	"strings"
	"testing"
)

func TestGenerateSpecSchema(t *testing.T) {
	t.Run("should generate a basic spec schema from the create action", func(t *testing.T) {
		// Arrange
		resourceConfig := &ResourceConfig{
			Verbs: []Verb{
				{Action: "create", Path: "/widgets", Method: "post"},
			},
		}
		mockDoc := &mockOASDocument{
			Paths: map[string]*mockPathItem{
				"/widgets": {
					Ops: map[string]Operation{
						"post": &mockOperation{
							RequestBody: RequestBodyInfo{
								Content: map[string]*Schema{
									"application/json": {
										Type: []string{"object"},
										Properties: []Property{
											{Name: "name", Schema: &Schema{Type: []string{"string"}}},
											{Name: "size", Schema: &Schema{Type: []string{"number"}}},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig(), resourceConfig)

		// Act
		result, err := generator.Generate()

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}

		schemaStr := string(result.SpecSchema)
		if !strings.Contains(schemaStr, `"name"`) {
			t.Error("Schema should contain 'name' property from request body")
		}
		if !strings.Contains(schemaStr, `"size"`) {
			t.Error("Schema should contain 'size' property from request body")
		}
		if !strings.Contains(schemaStr, `"type": "integer"`) {
			t.Error("Schema should have converted 'number' to 'integer'")
		}
	})

	t.Run("should add identifiers to the spec schema", func(t *testing.T) {
		// Arrange
		resourceConfig := &ResourceConfig{
			Verbs: []Verb{
				{Action: "create", Path: "/widgets", Method: "post"},
			},
			Identifiers: []string{"id"},
		}
		mockDoc := &mockOASDocument{
			Paths: map[string]*mockPathItem{
				"/widgets": {
					Ops: map[string]Operation{
						"post": &mockOperation{
							RequestBody: RequestBodyInfo{
								Content: map[string]*Schema{"application/json": {Type: []string{"object"}}},
							},
						},
					},
				},
			},
		}
		generatorConfig := DefaultGeneratorConfig()
		generatorConfig.IncludeIdentifiersInSpec = true
		generator := NewOASSchemaGenerator(mockDoc, generatorConfig, resourceConfig)

		// Act
		result, _ := generator.Generate()

		// Assert
		schemaStr := string(result.SpecSchema)
		if !strings.Contains(schemaStr, `"IDENTIFIER: id"`) {
			t.Error("Schema should contain the 'id' identifier with a description")
		}
	})

	t.Run("should add parameters from all verbs to the spec schema", func(t *testing.T) {
		// Arrange
		resourceConfig := &ResourceConfig{
			Verbs: []Verb{
				{Action: "create", Path: "/widgets", Method: "post"},
				{Action: "get", Path: "/widgets/{id}", Method: "get"},
			},
		}
		mockDoc := &mockOASDocument{
			Paths: map[string]*mockPathItem{
				"/widgets": {
					Ops: map[string]Operation{
						"post": &mockOperation{
							RequestBody: RequestBodyInfo{
								Content: map[string]*Schema{"application/json": {Type: []string{"object"}}},
							},
						},
					},
				},
				"/widgets/{id}": {
					Ops: map[string]Operation{
						"get": &mockOperation{
							Parameters: []ParameterInfo{
								{Name: "id", In: "path", Schema: &Schema{Type: []string{"string"}}},
								{Name: "verbose", In: "query", Schema: &Schema{Type: []string{"boolean"}}},
							},
						},
					},
				},
			},
		}
		generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig(), resourceConfig)

		// Act
		result, _ := generator.Generate()

		// Assert
		schemaStr := string(result.SpecSchema)
		if !strings.Contains(schemaStr, `"verbose"`) {
			t.Error("Schema should contain parameter 'verbose' from the get operation")
		}
		if !strings.Contains(schemaStr, `"PARAMETER: query - "`) {
			t.Error("Parameter description was not added correctly")
		}
	})

	t.Run("should exclude configured parameters from the spec schema", func(t *testing.T) {
		// Arrange
		resourceConfig := &ResourceConfig{
			Verbs: []Verb{
				{Action: "create", Path: "/widgets", Method: "post"},
			},
			ConfigurationFields: []ConfigurationField{
				{
					FromOpenAPI:        FromOpenAPI{Name: "api-version", In: "query"},
					FromRestDefinition: FromRestDefinition{Actions: []string{"create"}},
				},
			},
		}
		mockDoc := &mockOASDocument{
			Paths: map[string]*mockPathItem{
				"/widgets": {
					Ops: map[string]Operation{
						"post": &mockOperation{
							RequestBody: RequestBodyInfo{
								Content: map[string]*Schema{"application/json": {Type: []string{"object"}}},
							},
							Parameters: []ParameterInfo{
								{Name: "api-version", In: "query", Schema: &Schema{Type: []string{"string"}}},
							},
						},
					},
				},
			},
		}
		generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig(), resourceConfig)

		// Act
		result, err := generator.Generate()
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}

		// Assert
		schemaStr := string(result.SpecSchema)
		if strings.Contains(schemaStr, `"api-version"`) {
            t.Error("Schema should NOT contain 'api-version' as it is a configuration field")
        }
    })

    t.Run("should include enum validation for string properties", func(t *testing.T) {
        // Arrange
        resourceConfig := &ResourceConfig{
            Verbs: []Verb{
                {Action: "create", Path: "/widgets", Method: "post"},
            },
        }
        mockDoc := &mockOASDocument{
            Paths: map[string]*mockPathItem{
                "/widgets": {
                    Ops: map[string]Operation{
                        "post": &mockOperation{
                            RequestBody: RequestBodyInfo{
                                Content: map[string]*Schema{
                                    "application/json": {
                                        Type: []string{"object"},
                                        Properties: []Property{
                                            {
                                                Name: "status",
                                                Schema: &Schema{
                                                    Type: []string{"string"},
                                                    Enum: []interface{}{"active", "inactive"},
                                                },
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
            },
        }
        generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig(), resourceConfig)

        // Act
        result, err := generator.Generate()
        if err != nil {
            t.Fatalf("Expected no error, but got: %v", err)
        }

        // Assert
		schemaStr := string(result.SpecSchema)
		// Check for the key parts separately to be less sensitive to indentation and exact formatting.
		if !strings.Contains(schemaStr, `"status"`) {
			t.Error("Schema is missing 'status' property")
		}
		if !strings.Contains(schemaStr, `"enum"`) {
			t.Error("Schema is missing 'enum' key for status property")
		}
		if !strings.Contains(schemaStr, `"active"`) || !strings.Contains(schemaStr, `"inactive"`) {
			t.Error("Schema is missing enum values 'active' or 'inactive'")
		}
	})
}

func TestGenerateStatusSchema(t *testing.T) {
	t.Run("should generate a status schema from the get action response", func(t *testing.T) {
		// Arrange
		resourceConfig := &ResourceConfig{
			Verbs: []Verb{
				{Action: "get", Path: "/widgets/{id}", Method: "get"},
				{Action: "findby", Path: "/widgets", Method: "get"},
			},
			Identifiers:            []string{"id"},
			AdditionalStatusFields: []string{"last_updated", "version"},
		}
		mockDoc := &mockOASDocument{
			Paths: map[string]*mockPathItem{
				"/widgets/{id}": {
					Ops: map[string]Operation{
						"get": &mockOperation{
							Responses: map[int]ResponseInfo{
								200: {
									Content: map[string]*Schema{
										"application/json": {
											Type: []string{"object"},
											Properties: []Property{
												{Name: "id", Schema: &Schema{Type: []string{"string"}}},
												{Name: "version", Schema: &Schema{Type: []string{"number"}}},
												{Name: "last_updated", Schema: &Schema{Type: []string{"string"}}},
											},
										},
									},
								},
							},
						},
					},
				},
				"/widgets": {
					Ops: map[string]Operation{
						"get": &mockOperation{
							Responses: map[int]ResponseInfo{
								200: {
									Content: map[string]*Schema{
										"application/json": {
											Type: []string{"object"},
											Properties: []Property{
												{Name: "id", Schema: &Schema{Type: []string{"string"}}},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig(), resourceConfig)

		// Act
		result, err := generator.Generate()

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}
		if len(result.GenerationWarnings) > 0 {
			t.Errorf("Expected no generation warnings, but got: %v", result.GenerationWarnings)
		}
		if len(result.ValidationWarnings) > 0 {
			t.Errorf("Expected no validation warnings, but got: %v", result.ValidationWarnings)
		}

		schemaStr := string(result.StatusSchema)
		if !strings.Contains(schemaStr, `"id"`) || !strings.Contains(schemaStr, `"version"`) || !strings.Contains(schemaStr, `"last_updated"`) {
			t.Error("Status schema should contain all specified fields")
		}
		if !strings.Contains(schemaStr, `"type": "integer"`) {
			t.Error("Status schema should have converted 'number' to 'integer'")
		}
	})

	t.Run("should use findby as a fallback if get is not available", func(t *testing.T) {
		// Arrange
		resourceConfig := &ResourceConfig{
			Verbs: []Verb{
				{Action: "findby", Path: "/widgets", Method: "get"},
			},
			Identifiers:            []string{"id"},
			AdditionalStatusFields: []string{"status"},
		}
		mockDoc := &mockOASDocument{
			Paths: map[string]*mockPathItem{
				"/widgets": {
					Ops: map[string]Operation{
						"get": &mockOperation{
							Responses: map[int]ResponseInfo{
								200: {
									Content: map[string]*Schema{
										"application/json": {
											Type: []string{"array"},
											Items: &Schema{ // findby returns an array of items
												Type: []string{"object"},
												Properties: []Property{
													{Name: "id", Schema: &Schema{Type: []string{"string"}}},
													{Name: "status", Schema: &Schema{Type: []string{"string"}}},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig(), resourceConfig)

		// Act
		result, _ := generator.Generate()

		// Assert
		schemaStr := string(result.StatusSchema)
		if !strings.Contains(schemaStr, `"id"`) || !strings.Contains(schemaStr, `"status"`) {
			t.Error("Status schema should contain fields from findby response")
		}
	})

	t.Run("should default to string and warn for fields not in the response", func(t *testing.T) {
		// Arrange
		resourceConfig := &ResourceConfig{
			Verbs: []Verb{
				{Action: "get", Path: "/widgets/{id}", Method: "get"},
			},
			Identifiers:            []string{"id"},
			AdditionalStatusFields: []string{"non_existent_field"},
		}
		mockDoc := &mockOASDocument{
			Paths: map[string]*mockPathItem{
				"/widgets/{id}": {
					Ops: map[string]Operation{
						"get": &mockOperation{
							Responses: map[int]ResponseInfo{
								200: {Content: map[string]*Schema{"application/json": {Type: []string{"object"}}}},
							},
						},
					},
				},
			},
		}
		generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig(), resourceConfig)

		// Act
		result, err := generator.Generate()

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}
		if len(result.GenerationWarnings) != 2 { // One for 'id', one for 'non_existent_field'
			t.Fatalf("Expected 2 generation warnings for missing fields, but got %d", len(result.GenerationWarnings))
		}
		if !strings.Contains(result.GenerationWarnings[1].Error(), "non_existent_field") {
			t.Errorf("Warning message should mention 'non_existent_field'")
		}

		schemaStr := string(result.StatusSchema)
		expected := `"non_existent_field": {
      "type": "string"
    }`
		if !strings.Contains(schemaStr, expected) {
			t.Errorf("Expected non_existent_field to default to string, but it didn't. Got:\n%s", schemaStr)
		}
	})

}

func TestPrepareSchemaForCRD(t *testing.T) {
	t.Run("should correctly merge properties from allOf", func(t *testing.T) {
		schema := &Schema{
			AllOf: []*Schema{
				{
					Properties: []Property{
						{Name: "prop1", Schema: &Schema{Type: []string{"string"}}},
					},
				},
				{
					Properties: []Property{
						{Name: "prop2", Schema: &Schema{Type: []string{"number"}}},
					},
				},
			},
		}

		err := prepareSchemaForCRD(schema, DefaultGeneratorConfig())

		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}

		if len(schema.Properties) != 2 {
			t.Fatalf("Expected 2 properties, but got %d", len(schema.Properties))
		}

		prop1Found := false
		prop2Found := false
		for _, p := range schema.Properties {
			if p.Name == "prop1" {
				prop1Found = true
			}
			if p.Name == "prop2" {
				prop2Found = true
				if p.Schema.Type[0] != "integer" {
					t.Errorf("Expected prop2 to be of type 'integer', but got '%s'", p.Schema.Type[0])
				}
			}
		}

		if !prop1Found || !prop2Found {
			t.Errorf("Expected to find both prop1 and prop2, but prop1Found=%v, prop2Found=%v", prop1Found, prop2Found)
		}
	})

	t.Run("should recursively prepare schemas in arrays", func(t *testing.T) {
		schema := &Schema{
			Type: []string{"array"},
			Items: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "nestedProp", Schema: &Schema{Type: []string{"number"}}},
				},
			},
		}

		err := prepareSchemaForCRD(schema, DefaultGeneratorConfig())

		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}

		if schema.Items.Properties[0].Schema.Type[0] != "integer" {
			t.Errorf("Expected nestedProp to be of type 'integer', but got '%s'", schema.Items.Properties[0].Schema.Type[0])
		}
	})

	t.Run("should handle empty schema", func(t *testing.T) {
		schema := &Schema{}

		err := prepareSchemaForCRD(schema, DefaultGeneratorConfig())

		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}
	})

	t.Run("should handle nil schema", func(t *testing.T) {
		err := prepareSchemaForCRD(nil, DefaultGeneratorConfig())

		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}
	})
}
