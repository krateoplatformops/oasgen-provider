package oas2jsonschema

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposeStatusSchema_WithDotNotation(t *testing.T) {
	// Arrange: Define a complex, nested response schema to test against.
	responseSchema := &Schema{
		Type: []string{"object"},
		Properties: []Property{
			{Name: "id", Schema: &Schema{Type: []string{"string"}}},
			{Name: "metadata", Schema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "creationTimestamp", Schema: &Schema{Type: []string{"string"}, Format: "date-time"}},
					{Name: "user", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "name", Schema: &Schema{Type: []string{"string"}}},
							{Name: "profile", Schema: &Schema{
								Type: []string{"object"},
								Properties: []Property{
									{Name: "email", Schema: &Schema{Type: []string{"string"}}},
								},
							}},
						},
					}},
				},
			}},
			{Name: "tags", Schema: &Schema{
				Type:  []string{"array"},
				Items: &Schema{Type: []string{"string"}},
			}},
			// Fields for testing field names with literal dots
			{Name: "field.with.dot", Schema: &Schema{Type: []string{"boolean"}}},
			{Name: "parent.with.dot", Schema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "child", Schema: &Schema{Type: []string{"integer"}}},
				},
			}},
		},
	}

	testCases := []struct {
		name               string
		statusFields       []string
		expectedSchemaJSON string
		expectedWarnings   int
	}{
		{
			name:         "should handle a single top-level field",
			statusFields: []string{"id"},
			expectedSchemaJSON: `{
				"properties": {
					"id": { "type": "string" }
				},
				"type": "object"
			}`,
			expectedWarnings: 0,
		},
		{
			name:         "should handle a single nested field",
			statusFields: []string{"metadata.creationTimestamp"},
			expectedSchemaJSON: `{
				"properties": {
					"metadata": {
						"properties": {
							"creationTimestamp": { "type": "string" }
						},
						"type": "object",
						"x-crdgen-identifier-name": "StatusMetadata"
					}
				},
				"type": "object"
			}`,
			expectedWarnings: 0,
		},
		{
			name:         "should handle a deeply nested field",
			statusFields: []string{"metadata.user.profile.email"},
			expectedSchemaJSON: `{
				"properties": {
					"metadata": {
						"properties": {
							"user": {
								"properties": {
									"profile": {
										"properties": {
											"email": { "type": "string" }
										},
										"type": "object"
									}
								},
								"type": "object"
							}
						},
						"type": "object",
						"x-crdgen-identifier-name": "StatusMetadata"
					}
				},
				"type": "object"
			}`,
			expectedWarnings: 0,
		},
		{
			name:         "should combine multiple nested and top-level fields correctly",
			statusFields: []string{"id", "metadata.user.name", "metadata.creationTimestamp"},
			expectedSchemaJSON: `{
				"properties": {
					"id": { "type": "string" },
					"metadata": {
						"properties": {
							"user": {
								"properties": {
									"name": { "type": "string" }
								},
								"type": "object"
							},
							"creationTimestamp": { "type": "string" }
						},
						"type": "object",
						"x-crdgen-identifier-name": "StatusMetadata"
					}
				},
				"type": "object"
			}`,
			expectedWarnings: 0,
		},
		{
			name:         "should warn and default to string for a non-existent nested field",
			statusFields: []string{"metadata.user.nonexistent"},
			expectedSchemaJSON: `{
				"properties": {
					"metadata": {
						"properties": {
							"user": {
								"properties": {
									"nonexistent": { "type": "string" }
								},
								"type": "object"
							}
						},
						"type": "object",
						"x-crdgen-identifier-name": "StatusMetadata"
					}
				},
				"type": "object"
			}`,
			expectedWarnings: 1,
		},
		{
			name:         "should warn and default to string for a non-existent top-level field",
			statusFields: []string{"nonexistent"},
			expectedSchemaJSON: `{
				"properties": {
					"nonexistent": { "type": "string" }
				},
				"type": "object"
			}`,
			expectedWarnings: 1,
		},
		{
			name:         "should handle a mix of existing and non-existent fields",
			statusFields: []string{"id", "metadata.user.profile.nonexistent"},
			expectedSchemaJSON: `{
				"properties": {
					"id": { "type": "string" },
					"metadata": {
						"properties": {
							"user": {
								"properties": {
									"profile": {
										"properties": {
											"nonexistent": { "type": "string" }
										},
										"type": "object"
									}
								},
								"type": "object"
							}
						},
						"type": "object",
						"x-crdgen-identifier-name": "StatusMetadata"
					}
				},
				"type": "object"
			}`,
			expectedWarnings: 1,
		},
		{
			name:         "should handle the case of an object field (non-primitive)",
			statusFields: []string{"metadata.user"},
			expectedSchemaJSON: `{
				"properties": {
					"metadata": {
						"properties": {
							"user": {
								"properties": {
									"name": { "type": "string" },
									"profile": {
										"properties": {
											"email": { "type": "string" }
										},
										"type": "object"
									}
								},
								"type": "object"
							}
						},
						"type": "object",
						"x-crdgen-identifier-name": "StatusMetadata"
					}
				},
				"type": "object"
			}`,
			expectedWarnings: 0,
		},
		{
			name:         "should handle arrays fields correctly",
			statusFields: []string{"tags"},
			expectedSchemaJSON: `{
				"properties": {
					"tags": {
						"type": "array",
						"items": { "type": "string" }
					}
				},
				"type": "object"
			}`,
			expectedWarnings: 0,
		},
		{
			name:         "should handle a field with a literal dot using bracket notation",
			statusFields: []string{"['field.with.dot']"},
			expectedSchemaJSON: `{
				"properties": {
					"field.with.dot": { "type": "boolean" }
				},
				"type": "object"
			}`,
			expectedWarnings: 0,
		},
		{
			name:         "should handle a nested field under a parent with a literal dot",
			statusFields: []string{"['parent.with.dot'].child"},
			expectedSchemaJSON: `{
				"properties": {
					"parent.with.dot": {
						"properties": {
							"child": { "type": "integer" }
						},
						"type": "object",
						"x-crdgen-identifier-name": "StatusParent.with.dot"
					}
				},
				"type": "object"
			}`,
			expectedWarnings: 0,
		},
		{
			name:         "should generate a warning for invalid path syntax",
			statusFields: []string{"['unclosed.bracket"},
			expectedSchemaJSON: `{
				"type": "object"
			}`,
			expectedWarnings: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			mockGenerator := &OASSchemaGenerator{generatorConfig: DefaultGeneratorConfig()}
			statusSchema, warnings := mockGenerator.composeStatusSchema(tc.statusFields, responseSchema)

			// Assert
			assert.Len(t, warnings, tc.expectedWarnings, "Unexpected number of warnings")

			generatedBytes, err := GenerateJsonSchema(statusSchema, DefaultGeneratorConfig())
			require.NoError(t, err, "Failed to marshal the generated schema")

			var expectedMap, actualMap map[string]interface{}
			err = json.Unmarshal([]byte(tc.expectedSchemaJSON), &expectedMap)
			require.NoError(t, err, "Failed to unmarshal expected JSON")
			err = json.Unmarshal(generatedBytes, &actualMap)
			require.NoError(t, err, "Failed to unmarshal actual JSON")

			assert.Equal(t, expectedMap, actualMap, "The generated schema does not match the expected structure")
		})
	}
}

func TestFindPropertyByPath(t *testing.T) {
	// Arrange
	schema := &Schema{
		Type: []string{"object"},
		Properties: []Property{
			{Name: "metadata", Schema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "name", Schema: &Schema{Type: []string{"string"}}},
				},
			}},
		},
	}
	mockGenerator := &OASSchemaGenerator{generatorConfig: DefaultGeneratorConfig()}

	t.Run("should find a nested property", func(t *testing.T) {
		// Act
		prop, found := mockGenerator.findPropertyByPath(schema, strings.Split("metadata.name", "."))

		// Assert
		assert.True(t, found)
		assert.Equal(t, "name", prop.Name)
		assert.Equal(t, []string{"string"}, prop.Schema.Type)
	})

	t.Run("should not find a non-existent property", func(t *testing.T) {
		// Act
		_, found := mockGenerator.findPropertyByPath(schema, strings.Split("metadata.nonexistent", "."))

		// Assert
		assert.False(t, found)
	})
}
