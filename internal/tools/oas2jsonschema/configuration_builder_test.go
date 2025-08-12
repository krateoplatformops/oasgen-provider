package oas2jsonschema

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateConfigurationSchema(t *testing.T) {
	// Common mock document with various parameters
	mockDoc := &mockOASDocument{
		Paths: map[string]*mockPathItem{
			"/items": {
				Ops: map[string]Operation{
					"get": &mockOperation{
						Parameters: []ParameterInfo{
							{Name: "api-version", In: "query", Schema: &Schema{Type: []string{"string"}}},
							{Name: "X-Request-ID", In: "header", Schema: &Schema{Type: []string{"string"}}},
						},
					},
				},
			},
			"/items/{id}": {
				Ops: map[string]Operation{
					"put": &mockOperation{
						Parameters: []ParameterInfo{
							{Name: "id", In: "path", Schema: &Schema{Type: []string{"integer"}}},
						},
					},
				},
			},
		},
		securitySchemes: []SecuritySchemeInfo{
			{Name: "BearerAuth", Type: SchemeTypeHTTP, Scheme: "bearer"},
		},
	}

	testCases := []struct {
		name                string
		resourceConfig      *ResourceConfig
		doc                 OASDocument
		expectError         bool
		expectedSchemaPaths map[string]string // map of JSON path to expected type
	}{
		{
			name: "Parameters Only",
			doc:  mockDoc,
			resourceConfig: &ResourceConfig{
				Verbs: []Verb{
					{Action: "get", Path: "/items", Method: "get"},
					{Action: "put", Path: "/items/{id}", Method: "put"},
				},
				ConfigurationFields: []ConfigurationField{
					{
						FromOpenAPI:        FromOpenAPI{Name: "api-version", In: "query"},
						FromRestDefinition: FromRestDefinition{Action: "get"},
					},
					{
						FromOpenAPI:        FromOpenAPI{Name: "X-Request-ID", In: "header"},
						FromRestDefinition: FromRestDefinition{Action: "get"},
					},
					{
						FromOpenAPI:        FromOpenAPI{Name: "id", In: "path"},
						FromRestDefinition: FromRestDefinition{Action: "put"},
					},
				},
			},
			expectedSchemaPaths: map[string]string{
				"properties.query.properties.get.properties.api-version.type":   "string",
				"properties.header.properties.get.properties.X-Request-ID.type": "string",
				"properties.path.properties.put.properties.id.type":             "integer",
			},
		},
		{
			name: "Authentication Only",
			doc:  mockDoc,
			resourceConfig: &ResourceConfig{
				Verbs:               []Verb{}, // No verbs needed if only testing auth
				ConfigurationFields: []ConfigurationField{},
			},
			expectedSchemaPaths: map[string]string{
				"properties.authenticationMethods.properties.bearerAuth.type": "object",
			},
		},
		{
			name: "Combined Parameters and Authentication",
			doc:  mockDoc,
			resourceConfig: &ResourceConfig{
				Verbs: []Verb{
					{Action: "get", Path: "/items", Method: "get"},
				},
				ConfigurationFields: []ConfigurationField{
					{
						FromOpenAPI:        FromOpenAPI{Name: "api-version", In: "query"},
						FromRestDefinition: FromRestDefinition{Action: "get"},
					},
				},
			},
			expectedSchemaPaths: map[string]string{
				"properties.query.properties.get.properties.api-version.type": "string",
				"properties.authenticationMethods.properties.bearerAuth.type": "object",
			},
		},
		{
			name: "Empty Case - No Fields and No Auth",
			doc:  &mockOASDocument{}, // Doc with no security schemes
			resourceConfig: &ResourceConfig{
				Verbs:               []Verb{},
				ConfigurationFields: []ConfigurationField{},
			},
			expectedSchemaPaths: nil, // Expect nil schema
		},
		{
			name: "Gracefully Skips Invalid Fields",
			doc:  mockDoc,
			resourceConfig: &ResourceConfig{
				Verbs: []Verb{
					{Action: "get", Path: "/items", Method: "get"},
				},
				ConfigurationFields: []ConfigurationField{
					{
						FromOpenAPI:        FromOpenAPI{Name: "non-existent-param", In: "query"},
						FromRestDefinition: FromRestDefinition{Action: "get"},
					},
					{
						FromOpenAPI:        FromOpenAPI{Name: "api-version", In: "query"},
						FromRestDefinition: FromRestDefinition{Action: "get"},
					},
				},
			},
			expectedSchemaPaths: map[string]string{
				"properties.query.properties.get.properties.api-version.type": "string",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			generator := NewOASSchemaGenerator(tc.doc, DefaultGeneratorConfig(), tc.resourceConfig)

			// Act
			schemaBytes, err := generator.GenerateConfigurationSchema()

			// Assert
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tc.expectedSchemaPaths == nil {
				assert.Nil(t, schemaBytes, "Schema bytes should be nil for this test case")
				return
			}

			require.NotNil(t, schemaBytes, "Schema bytes should not be nil for this test case")

			var schemaMap map[string]interface{}
			err = json.Unmarshal(schemaBytes, &schemaMap)
			require.NoError(t, err, "Generated schema should be valid JSON")

			for path, expectedType := range tc.expectedSchemaPaths {
				keys := strings.Split(path, ".")
				val, ok := getNestedValue(schemaMap, keys...)
				assert.True(t, ok, "Expected path should exist in schema: %s", path)
				assert.Equal(t, expectedType, val, "Expected type mismatch at path: %s", path)
			}
		})
	}
}

// getNestedValue is a helper to traverse a nested map[string]interface{}
func getNestedValue(data map[string]interface{}, path ...string) (interface{}, bool) {
	var current interface{} = data
	for _, key := range path {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = m[key]
		if !ok {
			return nil, false
		}
	}
	return current, true
}
