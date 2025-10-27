package oas2jsonschema

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAreTypesCompatible(t *testing.T) {
	testCases := []struct {
		name     string
		types1   []string
		types2   []string
		expected bool
	}{
		{
			name:     "Identical primary types",
			types1:   []string{"string"},
			types2:   []string{"string"},
			expected: true,
		},
		{
			name:     "Identical primary types with null",
			types1:   []string{"object", "null"},
			types2:   []string{"object", "null"},
			expected: true,
		},
		{
			name:     "Different primary types",
			types1:   []string{"string"},
			types2:   []string{"integer"},
			expected: false,
		},
		{
			name:     "One primary type, one null",
			types1:   []string{"string", "null"},
			types2:   []string{"null"},
			expected: false,
		},
		{
			name:     "One primary type (not nullable), one null",
			types1:   []string{"string"},
			types2:   []string{"null"},
			expected: false,
		},
		{
			name:     "One null, one primary type (not nullable)",
			types1:   []string{"null"},
			types2:   []string{"boolean"},
			expected: false,
		},
		{
			name:     "Both empty",
			types1:   []string{},
			types2:   []string{},
			expected: true,
		},
		{
			name:     "One empty, one with primary type",
			types1:   []string{},
			types2:   []string{"string"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := areTypesCompatible(tc.types1, tc.types2)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestPrepareSchemaForCRD_WithRecursionGuard(t *testing.T) {
	// Get the default config to make the test robust against changes in default values.
	defaultConfig := DefaultGeneratorConfig()
	limit := defaultConfig.MaxRecursionDepth

	testCases := []struct {
		name                  string
		depth                 int
		expectError           bool
		expectedErrorContains string
	}{
		{
			name:                  "should return an error when max recursion depth is exceeded",
			depth:                 limit + 1,
			expectError:           true,
			expectedErrorContains: "recursion limit exceeded",
		},
		{
			name:        "should correctly prepare a deeply nested schema well within the recursion limit",
			depth:       limit / 2,
			expectError: false,
		},
		{
			name:        "should succeed at the exact recursion depth limit",
			depth:       limit,
			expectError: false,
		},
		{
			name:        "should succeed with a single level of nesting",
			depth:       1,
			expectError: false,
		},
		{
			name:        "should succeed with no nesting",
			depth:       0,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange: Create a schema with the specified depth.
			root := &Schema{Type: []string{"object"}}
			current := root

			for i := 0; i < tc.depth; i++ {
				next := &Schema{Type: []string{"object"}}
				current.Properties = []Property{
					{Name: "nested", Schema: next},
				}
				current = next
			}

			// Act
			err := prepareSchemaForCRD(root, defaultConfig)

			// Assert
			if tc.expectError {
				require.Error(t, err, "Expected an error but got none")
				assert.True(t, strings.Contains(err.Error(), tc.expectedErrorContains),
					"Error message should contain '%s'", tc.expectedErrorContains)
			} else {
				require.NoError(t, err, "Expected no error but got one")

				// Also, verify that the schema structure is preserved and wasn't truncated.
				var finalDepth int
				current = root
				for len(current.Properties) > 0 && current.Properties[0].Schema != nil {
					finalDepth++
					current = current.Properties[0].Schema
				}
				assert.Equal(t, tc.depth, finalDepth, "The final schema depth should be preserved")
			}
		})
	}
}

func TestPrepareSchemaForCRDWithVisited_MergeAllOfProperties(t *testing.T) {
	schema := &Schema{
		Type: []string{"object"},
		AllOf: []*Schema{
			{
				Properties: []Property{
					{Name: "prop1", Schema: &Schema{Type: []string{"string"}}},
					{Name: "prop2", Schema: &Schema{Type: []string{"integer"}}},
				},
				Required: []string{"prop1"},
			},
			{
				Properties: []Property{
					{Name: "prop2", Schema: &Schema{Type: []string{"number"}}}, // Duplicate property
					{Name: "prop3", Schema: &Schema{Type: []string{"boolean"}}},
				},
				Required: []string{"prop3"},
			},
		},
	}

	err := prepareSchemaForCRD(schema, DefaultGeneratorConfig())
	require.NoError(t, err, "prepareSchemaForCRD should not return an error")

	// Verify that properties are merged correctly
	expectedProperties := map[string]string{
		"prop1": "string",
		"prop2": "integer", // First occurrence should be kept
		"prop3": "boolean",
	}

	require.Equal(t, len(expectedProperties), len(schema.Properties), "Number of properties should match")

	for _, prop := range schema.Properties {
		expectedType, exists := expectedProperties[prop.Name]
		require.True(t, exists, "Unexpected property: %s", prop.Name)
		require.Equal(t, expectedType, prop.Schema.Type[0], "Property %s should have type %s", prop.Name, expectedType)
	}

	// Verify that required fields are merged correctly
	expectedRequired := []string{"prop1", "prop3"}
	require.Equal(t, len(expectedRequired), len(schema.Required), "Number of required fields should match")
	for _, req := range expectedRequired {
		require.Contains(t, schema.Required, req, "Required fields should contain %s", req)
	}
}

// Example reference: SubnetType in ArubaCloud Subnet schema
func TestPrepareSchemaForCRDWithVisited_MergeAllOfEnumOnlySchemas(t *testing.T) {
	schema := &Schema{
		Type: []string{"string"},
		AllOf: []*Schema{
			{
				Enum: []interface{}{"value1", "value2"},
			},
			{
				Enum: []interface{}{"value2", "value3"},
			},
		},
	}

	err := prepareSchemaForCRD(schema, DefaultGeneratorConfig())
	require.NoError(t, err, "prepareSchemaForCRD should not return an error")

	// Verify that enum values are merged correctly
	expectedEnum := []interface{}{"value1", "value2", "value3"}
	require.Equal(t, len(expectedEnum), len(schema.Enum), "Number of enum values should match")

	enumMap := make(map[interface{}]bool)
	for _, val := range schema.Enum {
		enumMap[val] = true
	}

	for _, expectedVal := range expectedEnum {
		require.True(t, enumMap[expectedVal], "Enum values should contain %v", expectedVal)
	}
}
