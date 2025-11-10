package oas2jsonschema

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var (
	nonAlphaNum = regexp.MustCompile(`[^a-zA-Z0-9]+`)
	camelCaseRE = regexp.MustCompile(`([a-z0-9])([A-Z])`)
)

// exportedName converts a field name into an exported Go struct field name with PascalCase.
func exportedName(name string) string {
	name = strings.TrimLeft(name, "_")
	if name == "" {
		return ""
	}

	// Normalize underscores and non-alphanumeric
	name = nonAlphaNum.ReplaceAllString(name, "_")

	// Split camelCase into snake_case before splitting
	name = camelCaseRE.ReplaceAllString(name, "${1}_${2}")

	// Split into parts and capitalize each
	parts := strings.Split(name, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}

	return strings.Join(parts, "")
}

// countNames traverses the JSON-like data structure and counts occurrences of normalized field names.
// parentKey tracks the context to know if we're inside a "properties" map.
func countNames(data interface{}, counts map[string]int, parentKey string) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			// Only count this key if we're inside a "properties" map
			// because those are the actual property names that will become struct fields
			if parentKey == "properties" {
				base := exportedName(key)
				if base != "" {
					counts[base]++
				}
			}

			// Recurse with the current key as context
			countNames(val, counts, key)
		}
	case []interface{}:
		for _, item := range v {
			countNames(item, counts, parentKey)
		}
	}
}

// annotate traverses the JSON-like data structure and adds annotations for field names
// that appear more than once globally.
// parentKey tracks the context to know if we're inside a "properties" map.
// The double pass approach (count then annotate) ensures we can check if we have more than one occurrence.
func annotate(data interface{}, totalCount, currentIndex map[string]int, annotationKey, parentKey string) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			// Skip early if not under "properties"
			if parentKey != "properties" {
				// Recurse with the current key as context
				annotate(val, totalCount, currentIndex, annotationKey, key)
				continue
			}

			base := exportedName(key)
			// Skip if base name empty or not duplicated
			if base == "" || totalCount[base] <= 1 {
				// Recurse with the current key as context
				annotate(val, totalCount, currentIndex, annotationKey, key)
				continue
			}

			currentIndex[base]++

			// Annotate only if the value is a schema object of type "object"
			schemaObj, ok := val.(map[string]interface{})
			if ok && schemaObj["type"] == "object" {
				schemaObj[annotationKey] = fmt.Sprintf("%s%d", base, currentIndex[base])
			}

			// Recurse with the current key as context
			annotate(val, totalCount, currentIndex, annotationKey, key)
		}

	case []interface{}:
		for _, item := range v {
			// Recurse with the current key as context
			annotate(item, totalCount, currentIndex, annotationKey, parentKey)
		}
	}
}

// annotateSchemas adds an annotation (customizable key) to both schemas (spec and status)
// for any normalized field name appearing more than once globally.
// This is necessary due to the underlying `crdgen` tool that needs to create Go struct from field names
func annotateSchemas(specSchema, statusSchema []byte, annotationKey string) ([]byte, []byte, error) {
	var specData, statusData interface{}

	// Unmarshal spec if non-nil
	if len(specSchema) > 0 {
		if err := json.Unmarshal(specSchema, &specData); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal spec: %w", err)
		}
	}

	// Unmarshal status if non-nil
	if len(statusSchema) > 0 {
		if err := json.Unmarshal(statusSchema, &statusData); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal status: %w", err)
		}
	}

	// Local maps for this invocation of annotateSchemas
	nameTotalCount := make(map[string]int)
	nameCurrentIndex := make(map[string]int)

	// Count globally across both inputs (safe even if nil)
	// Start with empty parent key since we're at the root
	countNames(specData, nameTotalCount, "")
	countNames(statusData, nameTotalCount, "")

	// Annotate both inputs, with shared counters
	// Start with empty parent key since we're at the root
	annotate(specData, nameTotalCount, nameCurrentIndex, annotationKey, "")
	annotate(statusData, nameTotalCount, nameCurrentIndex, annotationKey, "")

	// Marshal back
	var (
		specOut   []byte
		statusOut []byte
		err       error
	)

	if specData == nil {
		specOut = nil
	} else if specOut, err = json.MarshalIndent(specData, "", "  "); err != nil {
		return nil, nil, err
	}

	if statusData == nil {
		statusOut = nil
	} else if statusOut, err = json.MarshalIndent(statusData, "", "  "); err != nil {
		return nil, nil, err
	}

	return specOut, statusOut, nil
}
