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
		// Basic dot notation
		{
			name:     "Simple dot notation",
			path:     "a.b.c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Single field",
			path:     "pullRequestId",
			expected: []string{"pullRequestId"},
		},
		{
			name:     "Empty path",
			path:     "",
			expected: []string{""},
		},

		// Bracket notation
		{
			name:     "Field with literal dot in brackets",
			path:     "['searchCriteria.creatorId']",
			expected: []string{"searchCriteria.creatorId"},
		},
		{
			name:     "Double quotes in brackets",
			path:     `["field.with.quotes"]`,
			expected: []string{"field.with.quotes"},
		},
		{
			name:     "Triple dots escaped",
			path:     "['a...b']",
			expected: []string{"a...b"},
		},

		// Mixed notation
		{
			name:     "Mixed notation",
			path:     "completionOptions.['option.name'].value",
			expected: []string{"completionOptions", "option.name", "value"},
		},
		{
			name:     "Literal dot field at end",
			path:     "some.nested.['field.with.dot']",
			expected: []string{"some", "nested", "field.with.dot"},
		},
		{
			name:     "Useless bracket",
			path:     "a.['b'].c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Literal dot field in middle",
			path:     "user.['address.city'].street",
			expected: []string{"user", "address.city", "street"},
		},

		// Leading dots (attach to next segment)
		{
			name:     "Leading dot",
			path:     ".a.b",
			expected: []string{".a", "b"},
		},
		{
			name:     "Single segment with leading dot",
			path:     ".a",
			expected: []string{".a"},
		},
		{
			name:     "Multiple leading dots",
			path:     "..a.b",
			expected: []string{"..a", "b"},
		},

		// Trailing dots (error)
		{
			name:          "Trailing dot",
			path:          "a.b.",
			expectError:   true,
			errorContains: "trailing dot",
		},
		{
			name:          "Single segment trailing dot",
			path:          "a.",
			expectError:   true,
			errorContains: "trailing dot",
		},

		// Consecutive dots (error)
		{
			name:          "Consecutive dots in middle",
			path:          "a..b",
			expectError:   true,
			errorContains: "consecutive dots",
		},

		// Dot + bracket combinations
		{
			name:          "Dot before bracket",
			path:          ".['a']",
			expectError:   true,
			errorContains: "dot before bracket",
		},
		{
			name:          "Dot after bracket at end",
			path:          "['a'].",
			expectError:   true,
			errorContains: "trailing dot",
		},
		{
			name:          "Adjacent brackets without dot",
			path:          "a.['b']['c']",
			expectError:   true,
			errorContains: "adjacent brackets",
		},

		// Invalid bracket syntax
		{
			name:          "Unclosed bracket",
			path:          "['a.b.c",
			expectError:   true,
			errorContains: "unclosed",
		},
		{
			name:          "Missing quotes in bracket",
			path:          "a.[b].c",
			expectError:   true,
			errorContains: "bracket must contain quoted string",
		},
		{
			name:          "Empty bracket content",
			path:          "['']",
			expectError:   true,
			errorContains: "empty bracket content",
		},

		// Spaces
		{
			name:          "Spaces in path",
			path:          "a. b.c",
			expectError:   true,
			errorContains: "spaces",
		},
		{
			name:          "Spaces around brackets",
			path:          "[ 'field' ]",
			expectError:   true,
			errorContains: "spaces",
		},

		// Special characters in segments
		{
			name:     "Numeric field names",
			path:     "user.['address.123'].street",
			expected: []string{"user", "address.123", "street"},
		},
		{
			name:     "Underscores and numbers",
			path:     "user.['address_123'].street",
			expected: []string{"user", "address_123", "street"},
		},
		{
			name:     "Underscores and dashes",
			path:     "user.['address_123-456'].street",
			expected: []string{"user", "address_123-456", "street"},
		},

		// Escaped leading/trailing dots in brackets
		{
			name:     "Leading dot escaped in bracket",
			path:     "['.leading'].dot",
			expected: []string{".leading", "dot"},
		},
		{
			name:     "Trailing dot escaped in bracket",
			path:     "trailing.['dot.']",
			expected: []string{"trailing", "dot."},
		},

		// Only dots (error)
		{
			name:          "Only dots",
			path:          "...",
			expectError:   true,
			errorContains: "trailing dot",
		},

		// Double quotes in brackets
		{
			name:     "Double quotes in brackets mixed",
			path:     "user.[\"address.city\"].street",
			expected: []string{"user", "address.city", "street"},
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
