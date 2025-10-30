package pathparsing

import (
	"fmt"
	"regexp"
	"strings"

	"log"
)

// ParsePath parses a path string into a slice of segments.
// It supports:
// - standard dot notation for nested fields in a path (e.g., "a.b.c")
// - bracket notation for field names that contain literal dots (e.g., "['a.b']")
// - mixed notation (e.g., "a.b['c.d'].e")
// It returns an error if the path is malformed, such as having mismatched brackets
// or invalid characters outside of valid segments (e.g., "a..b").
func ParsePath(path string) ([]string, error) {

	// pathSegmentRegex is used to parse a path string into segments.
	// It matches either a bracketed segment `['...']` or `["..."]` or a standard dot-separated path for dot notation.
	var pathSegmentRegex = regexp.MustCompile(`(?:\[\s*['"]([^'"]+)['"]\s*\]|([^.\[\]]+))`)

	if path == "" {
		return []string{""}, nil
	}

	// Verify that brackets are properly matched
	if strings.Count(path, "[") != strings.Count(path, "]") {
		return nil, fmt.Errorf("mismatched brackets in path")
	}

	// Verify no spaces are present in the string
	if strings.Contains(path, " ") {
		return nil, fmt.Errorf("malformed path: contains spaces")
	}

	matches := pathSegmentRegex.FindAllStringSubmatch(path, -1)
	if matches == nil {
		return nil, fmt.Errorf("path could not be parsed")
	}
	// Array of matches, each match is an array of strings:
	// match[0] = full match text
	// match[1] = first capture group (if any) which is the bracketed value
	// match[2] = second capture group (if any) which is the plain dot-segment

	// The structure of the results from FindAllStringSubmatch is as follows:
	// - The full match (e.g., "a", "b", "c")
	// - The bracketed value (if any) (e.g., "['a.b']", "['option.name']")
	// - The standard dot-segment (if any) (e.g., "a", "b", "c")

	// Examples of the regex matches:
	// For path "a.b.c", matches will be:
	//   [ ["a", "", "a"], ["b", "", "b"], ["c", "", "c"] ]
	//
	// For path "['a.b'].c", matches will be:
	//   [ ["['a.b']", "a.b", ""], [".c", "", "c"] ]
	//
	// For path "completionOptions.['option.name'].value", matches will be:
	//   [ ["completionOptions", "", "completionOptions"], [".['option.name']", "option.name", ""], [".value", "", "value"] ]

	// Additional validation to ensure the entire path is valid.
	// We create a "skeleton" of the path by replacing valid segments with a placeholder.
	const placeholder = "§"
	skeleton := pathSegmentRegex.ReplaceAllString(path, placeholder)

	// Then, we check the skeleton for invalid dot sequences.
	if strings.Contains(skeleton, "..") {
		return nil, fmt.Errorf("malformed path: contains consecutive dots")
	}
	if strings.HasPrefix(skeleton, ".") {
		return nil, fmt.Errorf("malformed path: contains leading dot")
	}
	if strings.HasSuffix(skeleton, ".") {
		return nil, fmt.Errorf("malformed path: contains trailing dot")
	}

	// Finally, we check for any characters that are not part of a valid segment or a dot,
	// which would indicate a syntax error.
	remainingChars := strings.ReplaceAll(skeleton, placeholder, "")
	remainingChars = strings.ReplaceAll(remainingChars, ".", "")
	if len(remainingChars) > 0 {
		// check if remainingChars contains the square brackets
		if strings.Contains(remainingChars, "[") || strings.Contains(remainingChars, "]") {
			return nil, fmt.Errorf("malformed path: mismatched or invalid brackets")
		}
		return nil, fmt.Errorf("malformed path contains invalid characters: %s", remainingChars)
	}

	segments := make([]string, 0, len(matches))
	for _, match := range matches {
		// We are not interested in match[0] (the full match), only in match[1] and match[2].
		// At this point, either match[1] or match[2] will be non-empty so either:
		// - match[1] contains the bracketed segment (literal dot field)
		// - match[2] contains the standard dot-segment
		if match[1] != "" {
			// This is a bracketed segment and so a literal dot field like "['a.b']" in "['a.b'].c"
			segments = append(segments, match[1])
		} else {
			// This is a standard segment, like the 'c' in "['a.b'].c" or the 'c' in "a.b.c"
			segments = append(segments, match[2])
		}
	}
	log.Printf("[parsePath] Parsed segments: %v", segments)
	return segments, nil
}
