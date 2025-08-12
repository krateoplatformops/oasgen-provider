package oas2jsonschema

// getPrimaryType returns the primary type from a slice of types introuduced in OpenAPI 3.1.
// which allows multiple types including "null".
// Source: https://www.openapis.org/blog/2021/02/16/migrating-from-openapi-3-0-to-3-1-0
func getPrimaryType(types []string) string {
	for _, t := range types {
		if t != "null" {
			return t
		}
	}
	return ""
}

// areTypesCompatible checks if two slices of types are compatible based on their primary non-null type (OAS 3.1).
// The opinionated compatibility rules are:
// 1. If both have a primary type (e.g., "string", "object"), they must be identical.
// 2. If one has a primary type and the other does not (i.e., is only "null" or empty), they are incompatible.
// 3. If neither has a primary type, they are compatible (e.g., ["null"] vs []).
func areTypesCompatible(types1, types2 []string) bool {
	primaryType1 := getPrimaryType(types1)
	primaryType2 := getPrimaryType(types2)

	// If both have a primary type, they must be the same.
	// If one has a primary type and the other doesn't, they are not compatible.
	return primaryType1 == primaryType2
}
