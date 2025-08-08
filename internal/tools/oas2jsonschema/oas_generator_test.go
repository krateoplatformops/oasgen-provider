package oas2jsonschema

import (
	"strings"
	"testing"

	definitionv1alpha1 "github.com/krateoplatformops/oasgen-provider/apis/restdefinitions/v1alpha1"
)

// --- Mock Implementations ---

// mockOperation implements the Operation interface for testing.
type mockOperation struct {
	Parameters  []ParameterInfo
	RequestBody RequestBodyInfo
	Responses   map[int]ResponseInfo
}

func (m *mockOperation) GetParameters() []ParameterInfo     { return m.Parameters }
func (m *mockOperation) GetRequestBody() RequestBodyInfo    { return m.RequestBody }
func (m *mockOperation) GetResponses() map[int]ResponseInfo { return m.Responses }

// mockPathItem implements the PathItem interface for testing.
type mockPathItem struct {
	Ops map[string]Operation
}

func (m *mockPathItem) GetOperations() map[string]Operation { return m.Ops }

// mockOASDocument implements the OASDocument interface for testing.
type mockOASDocument struct {
	Paths           map[string]*mockPathItem
	securitySchemes []SecuritySchemeInfo
}

func (m *mockOASDocument) FindPath(path string) (PathItem, bool) {
	p, ok := m.Paths[path]
	return p, ok
}

func (m *mockOASDocument) SecuritySchemes() []SecuritySchemeInfo {
	return m.securitySchemes
}

// --- Test Suite ---

func TestGenerateSpecSchema(t *testing.T) {
	// Arrange: Common setup for all spec schema tests
	resource := definitionv1alpha1.Resource{
		VerbsDescription: []definitionv1alpha1.VerbsDescription{
			{Action: "create", Path: "/widgets", Method: "post"},
			{Action: "get", Path: "/widgets/{id}", Method: "get"},
		},
	}
	identifiers := []string{"id"}

	t.Run("should generate a basic spec schema from the create action", func(t *testing.T) {
		// Arrange
		resource := definitionv1alpha1.Resource{
			VerbsDescription: []definitionv1alpha1.VerbsDescription{
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
		generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig())

		// Act
		result, err := generator.Generate(resource, identifiers)

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
		resource := definitionv1alpha1.Resource{
			VerbsDescription: []definitionv1alpha1.VerbsDescription{
				{Action: "create", Path: "/widgets", Method: "post"},
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
		generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig())

		// Act
		result, _ := generator.Generate(resource, identifiers)

		// Assert
		schemaStr := string(result.SpecSchema)
		if !strings.Contains(schemaStr, `"IDENTIFIER: id"`) {
			t.Error("Schema should contain the 'id' identifier with a description")
		}
	})

	t.Run("should add parameters from all verbs to the spec schema", func(t *testing.T) {
		// Arrange
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
		generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig())

		// Act
		result, _ := generator.Generate(resource, identifiers)

		// Assert
		schemaStr := string(result.SpecSchema)
		if !strings.Contains(schemaStr, `"verbose"`) {
			t.Error("Schema should contain parameter 'verbose' from the get operation")
		}
		if !strings.Contains(schemaStr, `"PARAMETER: query, VERB: Get - "`) {
			t.Error("Parameter description was not added correctly")
		}
	})

	t.Run("should add authenticationRefs for security schemes", func(t *testing.T) {
		// Arrange
		resource := definitionv1alpha1.Resource{
			VerbsDescription: []definitionv1alpha1.VerbsDescription{
				{Action: "create", Path: "/widgets", Method: "post"},
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
			securitySchemes: []SecuritySchemeInfo{
				{Name: "BasicAuth", Type: SchemeTypeHTTP, Scheme: "basic"},
			},
		}
		generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig())

		// Act
		result, _ := generator.Generate(resource, identifiers)

		// Assert
		schemaStr := string(result.SpecSchema)
		if !strings.Contains(schemaStr, `"authenticationRefs"`) {
			t.Error("Schema should contain 'authenticationRefs' property")
		}
		if !strings.Contains(schemaStr, `"basicAuthRef"`) {
			t.Error("Schema should contain 'basicAuthRef' within authenticationRefs")
		}
		if !strings.Contains(string(result.SpecSchema), `"required": [
    "authenticationRefs"
  ]`) {
			t.Error("authenticationRefs should be a required field")
		}
	})
}

func TestGenerateStatusSchema(t *testing.T) {
	// Arrange: Common setup for all status schema tests
	resource := definitionv1alpha1.Resource{
		VerbsDescription: []definitionv1alpha1.VerbsDescription{
			{Action: "get", Path: "/widgets/{id}", Method: "get"},
			{Action: "findby", Path: "/widgets", Method: "get"},
		},
		AdditionalStatusFields: []string{"last_updated", "version"},
	}
	identifiers := []string{"id"}

	t.Run("should generate a status schema from the get action response", func(t *testing.T) {
		// Arrange
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
			},
		}
		generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig())

		// Act
		result, err := generator.Generate(resource, identifiers)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}
		if len(result.Warnings) > 1 { // one warning is expected
			t.Errorf("Expected one warning, but got: %v", result.Warnings)
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
		resourceWithFindBy := definitionv1alpha1.Resource{
			VerbsDescription: []definitionv1alpha1.VerbsDescription{
				{Action: "findby", Path: "/widgets", Method: "get"},
			},
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
		generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig())

		// Act
		result, _ := generator.Generate(resourceWithFindBy, identifiers)

		// Assert
		schemaStr := string(result.StatusSchema)
		if !strings.Contains(schemaStr, `"id"`) || !strings.Contains(schemaStr, `"status"`) {
			t.Error("Status schema should contain fields from findby response")
		}
	})

	t.Run("should default to string and warn for fields not in the response", func(t *testing.T) {
		// Arrange
		resourceMissingField := definitionv1alpha1.Resource{
			VerbsDescription:       []definitionv1alpha1.VerbsDescription{{Action: "get", Path: "/widgets/{id}", Method: "get"}},
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
		generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig())

		// Act
		result, err := generator.Generate(resourceMissingField, identifiers)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}
		if len(result.Warnings) != 2 { // One for 'id', one for 'non_existent_field'
			t.Fatalf("Expected 2 warnings for missing fields, but got %d", len(result.Warnings))
		}
		if !strings.Contains(result.Warnings[1].Error(), "non_existent_field") {
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

		err := prepareSchemaForCRD(schema)

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

		err := prepareSchemaForCRD(schema)

		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}

		if schema.Items.Properties[0].Schema.Type[0] != "integer" {
			t.Errorf("Expected nestedProp to be of type 'integer', but got '%s'", schema.Items.Properties[0].Schema.Type[0])
		}
	})

	t.Run("should handle empty schema", func(t *testing.T) {
		schema := &Schema{}

		err := prepareSchemaForCRD(schema)

		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}
	})

	t.Run("should handle nil schema", func(t *testing.T) {
		err := prepareSchemaForCRD(nil)

		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}
	})
}

func TestGenerateAuthCRDSchemas(t *testing.T) {
	t.Run("should generate correct auth schemas and refs", func(t *testing.T) {
		// Arrange
		mockDoc := &mockOASDocument{
			securitySchemes: []SecuritySchemeInfo{
				{Name: "BasicAuth", Type: SchemeTypeHTTP, Scheme: "basic"},
				{Name: "BearerAuth", Type: SchemeTypeHTTP, Scheme: "bearer"},
			},
		}
		generator := NewOASSchemaGenerator(mockDoc, DefaultGeneratorConfig())

		// Empty spec schema to add authentication refs
		specSchema := &Schema{}

		authCRDSchemas, err := generator.generateAuthCRDSchemas()
		if err != nil {
			t.Fatalf("Expected no error from generateAuthCRDSchemas, but got: %v", err)
		}
		addAuthenticationRefs(specSchema, authCRDSchemas)

		if len(authCRDSchemas) != 2 {
			t.Fatalf("Expected 2 auth schemas, but got %d", len(authCRDSchemas))
		}

		basicAuthSchema, ok := authCRDSchemas["BasicAuth"]
		if !ok {
			t.Fatal("Expected to find BasicAuth schema")
		}
		if !strings.Contains(string(basicAuthSchema), `"username"`) {
			t.Error("BasicAuth schema should contain 'username' property")
		}

		bearerAuthSchema, ok := authCRDSchemas["BearerAuth"]
		if !ok {
			t.Fatal("Expected to find BearerAuth schema")
		}
		if !strings.Contains(string(bearerAuthSchema), `"tokenRef"`) {
			t.Error("BearerAuth schema should contain 'tokenRef' property")
		}

		if len(specSchema.Properties) != 1 || specSchema.Properties[0].Name != "authenticationRefs" {
			t.Fatal("specSchema should have one property named 'authenticationRefs'")
		}

		authRefs := specSchema.Properties[0].Schema
		if len(authRefs.Properties) != 2 {
			t.Fatalf("Expected 2 auth refs, but got %d", len(authRefs.Properties))
		}

		basicRefFound := false
		bearerRefFound := false
		for _, p := range authRefs.Properties {
			if p.Name == "basicAuthRef" {
				basicRefFound = true
			}
			if p.Name == "bearerAuthRef" {
				bearerRefFound = true
			}
		}

		if !basicRefFound || !bearerRefFound {
			t.Errorf("Expected to find both basicAuthRef and bearerAuthRef, but basicRefFound=%v, bearerRefFound=%v", basicRefFound, bearerRefFound)
		}
	})
}
