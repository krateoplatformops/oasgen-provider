package oas2jsonschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
