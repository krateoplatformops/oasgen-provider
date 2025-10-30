package oas2jsonschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePath(t *testing.T) {
	testCases := []struct {
		name          string
		path          string
		expected      []string
		expectError   bool
		errorContains string
	}{
		{
			name:     "Simple dot notation",
			path:     "a.b.c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Field with literal dot in brackets",
			path:     "['searchCriteria.creatorId']",
			expected: []string{"searchCriteria.creatorId"},
		},
		{
			name:     "Mixed notation",
			path:     "completionOptions.['option.name'].value",
			expected: []string{"completionOptions", "option.name", "value"},
		},
		{
			name:     "Literal dot field at the end",
			path:     "some.nested.['field.with.dot']",
			expected: []string{"some", "nested", "field.with.dot"},
		},
		{
			name:     "Single field, no dots",
			path:     "pullRequestId",
			expected: []string{"pullRequestId"},
		},
		{
			name:        "Field with spaces in brackets",
			path:        "[ 'field with spaces' ].another",
			expectError: true,
		},
		{
			name:     "Field with double quotes inside brackets",
			path:     `["field.with.quotes"]`,
			expected: []string{"field.with.quotes"},
		},
		{
			name:        "Invalid path with unclosed bracket",
			path:        "['a.b.c",
			expectError: true,
		},
		{
			name:        "Invalid path with invalid bracket", // missing quotes around b
			path:        "a.[b].c",
			expectError: true,
		},
		{
			name:     "Valid path with useless bracket", // useless bracket but still valid
			path:     "a.['b'].c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:        "Invalid path with extra dots",
			path:        "a..b",
			expectError: true,
		},
		{
			name:        "Invalid path with even more dots",
			path:        "a...b",
			expectError: true,
		},
		{
			name:        "Empty path",
			path:        "",
			expected:    []string{""},
			expectError: false,
		},
		{
			name:        "Path with only brackets",
			path:        "['']",
			expectError: true,
		},
		{
			name:     "Associative array style with literal dot",
			path:     "user['address.city'].street",
			expected: []string{"user", "address.city", "street"},
		},
		{
			name:        "Start with dot as first character",
			path:        ".leading.dot",
			expectError: true,
		},
		{
			name:        "End with dot as last character",
			path:        "trailing.dot.",
			expectError: true,
		},
		{
			name:        "Path with spaces around brackets",
			path:        "  [ ' spaced.field ' ]  . next ",
			expectError: true,
		},
		{
			name:     "Path with numeric field names",
			path:     "user['address.123'].street",
			expected: []string{"user", "address.123", "street"},
		},
		{
			name:     "Path with underscores and numbers",
			path:     "user['address_123'].street",
			expected: []string{"user", "address_123", "street"},
		},
		{
			name:     "Path with underscores and dashes",
			path:     "user['address_123-456'].street",
			expected: []string{"user", "address_123-456", "street"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			segments, err := parsePath(tc.path)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, segments)
			}
		})
	}
}

func TestRemoveFieldAtPath(t *testing.T) {
	gen := &OASSchemaGenerator{
		generatorConfig: DefaultGeneratorConfig(),
	}

	testCases := []struct {
		name             string
		initialSchema    *Schema
		pathToRemove     string
		expectedSchema   *Schema
		expectedFound    bool
		expectParseError bool
	}{
		{
			name: "Remove simple nested field",
			initialSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "a", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "b", Schema: &Schema{Type: []string{"string"}}},
							{Name: "c", Schema: &Schema{Type: []string{"string"}}},
						},
						Required: []string{"b"},
					}},
				},
			},
			pathToRemove: "a.b",
			expectedSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "a", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "c", Schema: &Schema{Type: []string{"string"}}},
						},
						Required: []string{},
					}},
				},
			},
			expectedFound: true,
		},
		{
			name: "Remove field with literal dot in name",
			initialSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "searchCriteria.creatorId", Schema: &Schema{Type: []string{"string"}}},
					{Name: "otherField", Schema: &Schema{Type: []string{"string"}}},
				},
				Required: []string{"searchCriteria.creatorId"},
			},
			pathToRemove: "['searchCriteria.creatorId']",
			expectedSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "otherField", Schema: &Schema{Type: []string{"string"}}},
				},
				Required: []string{},
			},
			expectedFound: true,
		},
		{
			name: "Remove nested field under a field with a literal dot",
			initialSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "a.b", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "leaf", Schema: &Schema{Type: []string{"string"}}},
						},
					}},
					{Name: "other", Schema: &Schema{Type: []string{"string"}}},
				},
			},
			pathToRemove: "['a.b'].leaf",
			expectedSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "a.b", Schema: &Schema{
						Type:       []string{"object"},
						Properties: []Property{},
					}},
					{Name: "other", Schema: &Schema{Type: []string{"string"}}},
				},
			},
			expectedFound: true,
		},
		{
			name: "Path not found - simple",
			initialSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "a", Schema: &Schema{Type: []string{"string"}}},
				},
			},
			pathToRemove: "a.b",
			expectedSchema: &Schema{ // Unchanged
				Type: []string{"object"},
				Properties: []Property{
					{Name: "a", Schema: &Schema{Type: []string{"string"}}},
				},
			},
			expectedFound: false,
		},
		{
			name: "Path not found - literal dot name does not exist",
			initialSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "a.b", Schema: &Schema{Type: []string{"string"}}},
				},
			},
			pathToRemove: "['c.d']",
			expectedSchema: &Schema{ // Unchanged
				Type: []string{"object"},
				Properties: []Property{
					{Name: "a.b", Schema: &Schema{Type: []string{"string"}}},
				},
			},
			expectedFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pathSegments, err := parsePath(tc.pathToRemove)
			if tc.expectParseError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Use a deep copy to avoid modifying the original test case data
			schemaToModify := tc.initialSchema.deepCopy()

			found := gen.removeFieldAtPath(schemaToModify, pathSegments)

			assert.Equal(t, tc.expectedFound, found)
			assert.Equal(t, tc.expectedSchema, schemaToModify)
		})
	}
}
