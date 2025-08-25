package oas2jsonschema

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test Structures
type SimpleStruct struct {
	Name        string `json:"name"`
	Count       int    `json:"count"`
	Description string `json:"description,omitempty"`
}

type NestedStruct struct {
	ID      string       `json:"id"`
	Details SimpleStruct `json:"details"`
}

type EmbeddedFields struct {
	SimpleStruct
	IsEnabled bool `json:"isEnabled"`
}

func TestReflectSchema(t *testing.T) {
	testCases := []struct {
		name          string
		inputType     reflect.Type
		expected      *Schema
		expectError   bool
		errorContains string
	}{
		{
			name:      "Simple Struct",
			inputType: reflect.TypeOf(SimpleStruct{}),
			expected: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "name", Schema: &Schema{Type: []string{"string"}}},
					{Name: "count", Schema: &Schema{Type: []string{"int"}}},
					{Name: "description", Schema: &Schema{Type: []string{"string"}}},
				},
				Required: []string{"name", "count"},
			},
			expectError: false,
		},
		{
			name:      "Nested Struct",
			inputType: reflect.TypeOf(NestedStruct{}),
			expected: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "id", Schema: &Schema{Type: []string{"string"}}},
					{
						Name: "details",
						Schema: &Schema{
							Type: []string{"object"},
							Properties: []Property{
								{Name: "name", Schema: &Schema{Type: []string{"string"}}},
								{Name: "count", Schema: &Schema{Type: []string{"int"}}},
								{Name: "description", Schema: &Schema{Type: []string{"string"}}},
							},
							Required: []string{"name", "count"},
						},
					},
				},
				Required: []string{"id", "details"},
			},
			expectError: false,
		},
		{
			name:      "Struct with Embedded Fields",
			inputType: reflect.TypeOf(EmbeddedFields{}),
			expected: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "name", Schema: &Schema{Type: []string{"string"}}},
					{Name: "count", Schema: &Schema{Type: []string{"int"}}},
					{Name: "description", Schema: &Schema{Type: []string{"string"}}},
					{Name: "isEnabled", Schema: &Schema{Type: []string{"bool"}}},
				},
				Required: []string{"isEnabled", "name", "count"},
			},
			expectError: false,
		},
		{
			name:        "Nil Type",
			inputType:   nil,
			expected:    nil,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			schema, err := reflectSchema(tc.inputType)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, schema)
			}
		})
	}
}
