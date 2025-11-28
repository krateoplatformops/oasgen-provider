package pathparsing

import (
	"fmt"
	"strings"

	"github.com/go-andiamo/splitter"
)

var dotSplitter = splitter.MustCreateSplitter('.', splitter.SquareBrackets, splitter.DoubleQuotes, splitter.SingleQuotes)

// ParsePath parses a path string into segments.
// Supports: dot notation (a.b.c) and bracket notation with both single and double quotes (a['b.c'], a["b.c"]).
func ParsePath(path string) ([]string, error) {
	if path == "" {
		return []string{""}, nil
	}

	if strings.Contains(path, " ") {
		return nil, fmt.Errorf("malformed path: contains spaces")
	}

	// Check for consecutive dots outside brackets (error).
	// E.g., a..b is invalid but ['a..b'] is valid.
	inBracket := false
	leadingDotsEnded := false
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '[':
			inBracket = true
			leadingDotsEnded = true
		case ']':
			inBracket = false
		case '.':
			if !inBracket && leadingDotsEnded && i+1 < len(path) && path[i+1] == '.' {
				return nil, fmt.Errorf("malformed path: consecutive dots")
			}
		default:
			leadingDotsEnded = true
		}
	}

	// Check for trailing dot (end of path)
	if len(path) > 0 && path[len(path)-1] == '.' {
		// Make sure it's not inside a bracket
		inBracket := false
		for i := 0; i < len(path)-1; i++ {
			switch path[i] {
			case '[':
				inBracket = true
			case ']':
				inBracket = false
			}
		}
		if !inBracket {
			return nil, fmt.Errorf("malformed path: trailing dot")
		}
	}

	// Split by dots outside brackets
	// Example: "a.b['c.d'].e" -> ["a", "b", "['c.d']", "e"]
	parts, err := dotSplitter.Split(path)
	if err != nil {
		return nil, fmt.Errorf("malformed path: %w", err)
	}

	// Handle leading dots - attach to first segment
	merged := make([]string, 0, len(parts))
	leadingDots := 0

	for _, p := range parts {
		if p == "" {
			leadingDots++
		} else {
			prefix := strings.Repeat(".", leadingDots)
			merged = append(merged, prefix+p)
			leadingDots = 0
		}
	}

	segments := make([]string, 0, len(merged))
	for _, p := range merged {
		seg, err := parseSegment(p)
		if err != nil {
			return nil, err
		}
		segments = append(segments, seg)
	}

	return segments, nil
}

// parseSegment parses a single segment, handling bracket notation and validation.
func parseSegment(s string) (string, error) {
	if s == "" {
		return "", fmt.Errorf("malformed path: empty segment")
	}

	// Strip leading dots for validation, but keep them in result
	leadingDots := 0
	for leadingDots < len(s) && s[leadingDots] == '.' {
		leadingDots++
	}

	if leadingDots == len(s) {
		// All dots, no content
		return "", fmt.Errorf("malformed path: segment has only dots")
	}

	rest := s[leadingDots:]

	// Plain segment (no brackets)
	if !strings.HasPrefix(rest, "[") {
		if strings.ContainsAny(rest, "[]'\"") { // If there is no opening bracket, these chars are invalid
			return "", fmt.Errorf("malformed path: invalid characters in segment")
		}
		return s, nil // return with leading dots
	}

	// Bracketed segment with leading dots is invalid, e.g., just .['field'] is not allowed
	if leadingDots > 0 {
		return "", fmt.Errorf("malformed path: dot before bracket")
	}

	// Bracketed segment: must be ['...'] or ["..."]
	if len(rest) < 4 {
		return "", fmt.Errorf("malformed path: bracket must contain quoted string")
	}
	if !strings.HasSuffix(rest, "]") {
		return "", fmt.Errorf("malformed path: unclosed bracket")
	}

	// Check for adjacent brackets like ['a']['b'] (without dot in between, invalid)
	closeIdx := strings.Index(rest, "]")
	if closeIdx != len(rest)-1 {
		return "", fmt.Errorf("malformed path: adjacent brackets must be separated by dot")
	}

	inner := rest[1 : len(rest)-1] // remove [ and ] at the ends
	if len(inner) < 2 {
		return "", fmt.Errorf("malformed path: empty bracket content")
	}

	// At this point, inner should be a quoted string like 'a.b' or "a.b"

	quote := inner[0]
	if quote != '\'' && quote != '"' {
		return "", fmt.Errorf("malformed path: bracket must contain quoted string")
	}
	if inner[len(inner)-1] != quote {
		return "", fmt.Errorf("malformed path: mismatched quotes")
	}

	content := inner[1 : len(inner)-1]
	if content == "" {
		return "", fmt.Errorf("malformed path: empty bracket content")
	}

	return content, nil
}
