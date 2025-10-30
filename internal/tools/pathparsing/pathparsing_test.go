package pathparsing

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
			segments, err := ParsePath(tc.path)

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
