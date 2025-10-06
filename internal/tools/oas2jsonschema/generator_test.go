package oas2jsonschema

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
									"application/json": {Type: []string{"object"},
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
		if !strings.Contains(schemaStr, `"PARAMETER: query"`) {
			t.Error("Parameter description was not added correctly")
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

	t.Run("should exclude a top-level field from the spec schema", func(t *testing.T) {
		// Arrange
		resourceConfig := &ResourceConfig{
			Verbs: []Verb{
				{Action: "create", Path: "/widgets", Method: "post"},
			},
			ExcludedSpecFields: []string{"size"},
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
		if strings.Contains(schemaStr, `"size"`) {
			t.Error("Schema should NOT contain 'size' as it is an excluded field")
		}
	})

	t.Run("should exclude a nested field from the spec schema", func(t *testing.T) {
		// Arrange
		resourceConfig := &ResourceConfig{
			Verbs: []Verb{
				{Action: "create", Path: "/widgets", Method: "post"},
			},
			ExcludedSpecFields: []string{"dimensions.width"},
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
											{Name: "dimensions", Schema: &Schema{
												Type: []string{"object"},
												Properties: []Property{
													{Name: "width", Schema: &Schema{Type: []string{"number"}}},
													{Name: "height", Schema: &Schema{Type: []string{"number"}}},
												},
											}},
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
		if strings.Contains(schemaStr, `"width"`) {
			t.Error("Schema should NOT contain 'width' as it is an excluded field")
		}
	})

	t.Run("should exclude many fields from the spec schema", func(t *testing.T) {
		// Arrange
		resourceConfig := &ResourceConfig{
			Verbs: []Verb{
				{Action: "create", Path: "/widgets", Method: "post"},
			},
			ExcludedSpecFields: []string{"name", "dimensions.height"},
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
											{Name: "dimensions", Schema: &Schema{
												Type: []string{"object"},
												Properties: []Property{
													{Name: "width", Schema: &Schema{Type: []string{"number"}}},
													{Name: "height", Schema: &Schema{Type: []string{"number"}}},
												},
											}},
											{Name: "color", Schema: &Schema{Type: []string{"string"}}},
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
		if strings.Contains(schemaStr, `"name"`) {
			t.Error("Schema should NOT contain 'name' as it is an excluded field")
		}
		if strings.Contains(schemaStr, `"height"`) {
			t.Error("Schema should NOT contain 'height' as it is an excluded field")
		}
		if !strings.Contains(schemaStr, `"width"`) {
			t.Error("Schema should contain 'width' as it is NOT an excluded field")
		}
		if !strings.Contains(schemaStr, `"color"`) {
			t.Error("Schema should contain 'color' as it is NOT an excluded field")
		}
	})

	t.Run("should generate a warning when an excluded field is not found", func(t *testing.T) {
		// Arrange
		resourceConfig := &ResourceConfig{
			Verbs: []Verb{
				{Action: "create", Path: "/widgets", Method: "post"},
			},
			ExcludedSpecFields: []string{"non_existent_field"},
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

		if len(result.GenerationWarnings) != 2 {
			t.Fatalf("Expected 2 generation warnings for missing fields, but got %d", len(result.GenerationWarnings))
		}
		if !strings.Contains(result.GenerationWarnings[0].Error(), "non_existent_field") {
			t.Errorf("Warning message should mention 'non_existent_field'")
		}
	})

	t.Run("should exclude a configured field from the spec schema", func(t *testing.T) {
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

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}

		schemaStr := string(result.SpecSchema)
		if strings.Contains(schemaStr, `"api-version"`) {
			t.Error("Schema should NOT contain 'api-version' as it is a configuration field")
		}
	})

	t.Run("should exclude a nested configured field from the spec schema", func(t *testing.T) {
		// Arrange
		resourceConfig := &ResourceConfig{
			Verbs: []Verb{
				{Action: "create", Path: "/widgets", Method: "post"},
			},
			ConfigurationFields: []ConfigurationField{
				{
					FromOpenAPI:        FromOpenAPI{Name: "credentials.username", In: "header"},
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
								Content: map[string]*Schema{
									"application/json": {
										Type: []string{"object"},
										Properties: []Property{
											{Name: "credentials", Schema: &Schema{
												Type: []string{"object"},
												Properties: []Property{
													{Name: "username", Schema: &Schema{Type: []string{"string"}}},
													{Name: "password", Schema: &Schema{Type: []string{"string"}}},
												},
											}},
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
		if strings.Contains(schemaStr, `"username"`) {
			t.Error("Schema should NOT contain 'username' as it is a configuration field")
		}
	})

	t.Run("should generate a warning when a configured field is not found", func(t *testing.T) {
		// Arrange
		resourceConfig := &ResourceConfig{
			Verbs: []Verb{
				{Action: "create", Path: "/widgets", Method: "post"},
			},
			ConfigurationFields: []ConfigurationField{
				{
					FromOpenAPI:        FromOpenAPI{Name: "non_existent_field", In: "query"},
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

		if len(result.GenerationWarnings) != 2 {
			t.Fatalf("Expected 2 generation warning for missing field, but got %d", len(result.GenerationWarnings))
		}
		if !strings.Contains(result.GenerationWarnings[0].Error(), "non_existent_field") {
			t.Errorf("Warning message should mention 'non_existent_field'")
		}
	})

	t.Run("should handle required fields in the spec schema", func(t *testing.T) {
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
										Type:     []string{"object"},
										Required: []string{"name"},
										Properties: []Property{
											{Name: "name", Schema: &Schema{Type: []string{"string"}}},
											{Name: "size", Schema: &Schema{Type: []string{"integer"}}},
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
		if !strings.Contains(schemaStr, `"required": [ "name" ]`) && strings.Contains(schemaStr, `"required": [ "size" ]`) {
			t.Error("Schema should mark 'name' as a required property and not 'size'")
		}
	})

	t.Run("should correctly handle allOf with an empty referenced schema", func(t *testing.T) {
		// Arrange
		resourceConfig := &ResourceConfig{
			Verbs: []Verb{
				{Action: "create", Path: "/widgets", Method: "post"},
			},
		}

		// Define the empty schema that will be "referenced"
		emptySchema := &Schema{
			Type:       []string{"object"},
			Properties: []Property{}, // Explicitly empty properties
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
										},
										AllOf: []*Schema{
											emptySchema, // Point directly to the schema instance
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

		// 1. Verify the parent's own property is still there.
		if !strings.Contains(schemaStr, `"name"`) {
			t.Error("Schema should still contain 'name' property from the parent")
		}

		// 2. Verify the allOf keyword has been successfully removed.
		if strings.Contains(schemaStr, `allOf`) {
			t.Error("Schema should not contain 'allOf' after processing")
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

	// TestGenerator_SpecCorruption specifically targets the bug where generating a status schema
	// with a nested identifier (e.g., "metadata.name") would corrupt the spec schema by removing
	// sibling fields from the "metadata" object.
	t.Run("should not corrupt spec schema when using nested identifiers for status", func(t *testing.T) {
		// This shared schema will be used for both the request body (spec) and the response (status).
		// This is the key to reproducing the bug, as it creates the possibility of pointer aliasing.
		sharedSchema := &Schema{
			Type: []string{"object"},
			Properties: []Property{
				{Name: "id", Schema: &Schema{Type: []string{"string"}}},
				{
					Name: "metadata",
					Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "name", Schema: &Schema{Type: []string{"string"}}},
							{Name: "location", Schema: &Schema{Type: []string{"string"}}},
							{Name: "tags", Schema: &Schema{Type: []string{"array"}, Items: &Schema{Type: []string{"string"}}}},
						},
					},
				},
			},
		}

		// 1. Arrange
		mockDoc := &mockOASDocument{
			Paths: map[string]*mockPathItem{
				"/widgets": {
					Ops: map[string]Operation{
						"post": &mockOperation{
							RequestBody: RequestBodyInfo{
								Content: map[string]*Schema{"application/json": sharedSchema},
							},
						},
					},
				},
				"/widgets/{id}": {
					Ops: map[string]Operation{
						"get": &mockOperation{
							Responses: map[int]ResponseInfo{200: {Content: map[string]*Schema{"application/json": sharedSchema}}},
						},
					},
				},
			},
		}

		resourceConfig := &ResourceConfig{
			Verbs: []Verb{
				{Action: "create", Path: "/widgets", Method: "post"},
				{Action: "get", Path: "/widgets/{id}", Method: "get"},
			},
			// Use a nested identifier
			Identifiers: []string{"metadata.name"},
			// Also include a top-level field
			AdditionalStatusFields: []string{"id"},
		}

		generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig(), resourceConfig)

		// 2. Act
		result, err := generator.Generate()
		require.NoError(t, err)
		require.NotNil(t, result)

		// 3. Assert
		// Unmarshal both schemas for inspection.
		var specSchema map[string]interface{}
		err = json.Unmarshal(result.SpecSchema, &specSchema)
		require.NoError(t, err, "Failed to unmarshal spec schema")

		var statusSchema map[string]interface{}
		err = json.Unmarshal(result.StatusSchema, &statusSchema)
		require.NoError(t, err, "Failed to unmarshal status schema")

		// Assert Status Schema
		statusProps := statusSchema["properties"].(map[string]interface{})
		assert.Contains(t, statusProps, "id", "Status schema should have 'id'")
		statusMetadata := statusProps["metadata"].(map[string]interface{})
		statusMetadataProps := statusMetadata["properties"].(map[string]interface{})
		assert.Len(t, statusMetadataProps, 1, "Status metadata should only have one property")
		assert.Contains(t, statusMetadataProps, "name", "Status metadata should only contain 'name'")

		// Assert Spec Schema (it should NOT be corrupted)
		specProps := specSchema["properties"].(map[string]interface{})
		assert.Contains(t, specProps, "id", "Spec schema should have 'id'")
		specMetadata := specProps["metadata"].(map[string]interface{})
		specMetadataProps := specMetadata["properties"].(map[string]interface{})
		assert.Len(t, specMetadataProps, 3, "Spec metadata SHOULD have all 3 original properties")
		assert.Contains(t, specMetadataProps, "name", "Spec metadata should contain 'name'")
		assert.Contains(t, specMetadataProps, "location", "Spec metadata should still contain 'location'")
		assert.Contains(t, specMetadataProps, "tags", "Spec metadata should still contain 'tags'")
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
