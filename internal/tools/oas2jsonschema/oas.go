package oas2jsonschema

import (
	"fmt"
	"net/http"
	"strings"

	definitionv1alpha1 "github.com/krateoplatformops/oasgen-provider/apis/restdefinitions/v1alpha1"

	"github.com/krateoplatformops/crdgen"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/generation"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/text"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

type OASSchemaGenerator struct {
	specByteSchema   []byte
	statusByteSchema []byte
	secByteSchema    map[string][]byte
}

// GenerateByteSchemas generates the byte schemas for the spec, status and auth schemas.
// Could return a fatal error and a list of generic errors (non-fatal).
func GenerateByteSchemas(doc *libopenapi.DocumentModel[v3.Document], resource definitionv1alpha1.Resource, identifiers []string) (g *OASSchemaGenerator, errors []error, fatalError error) {

	// Initialization and first validation checks

	secByteSchema := make(map[string][]byte)
	var schema *base.Schema
	bodySchema := base.CreateSchemaProxy(&base.Schema{Properties: orderedmap.New[string, *base.SchemaProxy]()})
	if bodySchema == nil {
		return nil, errors, fmt.Errorf("schemaproxy is nil")
	}
	schema, err := bodySchema.BuildSchema()
	if err != nil {
		return nil, errors, fmt.Errorf("building schema")
	}
	if doc.Model.Components == nil {
		return nil, errors, fmt.Errorf("components not found")
	}
	if doc.Model.Components.SecuritySchemes == nil {
		return nil, errors, fmt.Errorf("security schemes not found")
	}
	for secSchemaPair := doc.Model.Components.SecuritySchemes.First(); secSchemaPair != nil; secSchemaPair = secSchemaPair.Next() {
		authSchemaName, err := generation.GenerateAuthSchemaName(secSchemaPair.Value())
		if err != nil {
			return nil, errors, fmt.Errorf("auth schema name: %w", err)
		}

		secByteSchema[authSchemaName], err = generation.GenerateAuthSchemaFromSecuritySchema(secSchemaPair.Value())
		if err != nil {
			return nil, errors, fmt.Errorf("auth schema: %w", err)
		}
	}

	// Spec schema generation

	specByteSchema := make(map[string][]byte)
	for _, verb := range resource.VerbsDescription {

		// 1. Add 'create' request body
		if strings.EqualFold(verb.Action, "create") {
			path := doc.Model.Paths.PathItems.Value(verb.Path)
			if path == nil {
				return nil, errors, fmt.Errorf("path %s not found", verb.Path)
			}

			ops := path.GetOperations()
			if ops == nil {
				return nil, errors, fmt.Errorf("operations not found for %s", verb.Path)
			}

			op := ops.Value(strings.ToLower(verb.Method))
			if op == nil {
				return nil, errors, fmt.Errorf("operation not found for %s on path %s", verb.Method, verb.Path)
			}
			if op.RequestBody != nil {
				bodySchema = op.RequestBody.Content.Value("application/json").Schema
			}
			if bodySchema == nil {
				return nil, errors, fmt.Errorf("body schema not found for %s", verb.Path)
			}
			schema, err = bodySchema.BuildSchema()
			if err != nil {
				return nil, errors, fmt.Errorf("building schema for %s: %w", verb.Path, err)
			}
			if len(schema.Type) > 0 {
				if schema.Type[0] == "array" { // this assumes the shape is like: ["array", "null"] in this order
					schema.Properties = orderedmap.New[string, *base.SchemaProxy]()
					schema.Properties.Set("items", base.CreateSchemaProxy(
						&base.Schema{
							Type:  []string{"array"},
							Items: schema.Items,
						}))
					schema.Type = []string{"object"}
				}
			}

			populateFromAllOf(schema)
		}

		// 2. Add 'AuthenticationRefs'
		if len(secByteSchema) > 0 {
			authPair := orderedmap.NewPair("authenticationRefs", base.CreateSchemaProxy(&base.Schema{
				Type:        []string{"object"},
				Description: "AuthenticationRefs represent the reference to a CR containing the authentication information. One authentication method must be set."}))
			req := []string{
				"authenticationRefs",
			}

			if schema == nil {
				om := orderedmap.New[string, *base.SchemaProxy]()
				om.Set(authPair.Key(), authPair.Value())
				schemaproxy := base.CreateSchemaProxy(&base.Schema{
					Type:       []string{"object"},
					Properties: om,
					Required:   req,
				})
				schema, err = schemaproxy.BuildSchema()
				if err != nil {
					return nil, errors, fmt.Errorf("building schema for %s: %w", verb.Path, err)
				}

			} else {
				if schema.Properties == nil {
					schema.Properties = orderedmap.New[string, *base.SchemaProxy]()
				}
				schema.Properties.Set(authPair.Key(), authPair.Value())
				schema.Required = req
			}
		}
		for key := range secByteSchema {
			authSchemaProxy := schema.Properties.Value("authenticationRefs")
			if authSchemaProxy == nil {
				return nil, errors, fmt.Errorf("building schema for %s: %w", verb.Path, err)
			}

			sch, err := authSchemaProxy.BuildSchema()
			if err != nil {
				return nil, errors, fmt.Errorf("building schema for %s: %w", verb.Path, err)
			}

			if sch == nil {
				authSchemaProxy = base.CreateSchemaProxy(&base.Schema{
					Type:        []string{"object"},
					Description: "AuthenticationRefs represent the reference to a CR containing the authentication information. One authentication method must be set.",
					Properties:  orderedmap.New[string, *base.SchemaProxy](),
					Required:    []string{"authenticationRefs"},
				})
				sch, err = authSchemaProxy.BuildSchema()
				if err != nil {
					return nil, errors, fmt.Errorf("building schema for %s: %w", verb.Path, err)
				}
			}

			if sch.Properties == nil {
				sch.Properties = orderedmap.New[string, *base.SchemaProxy]()
			}
			sch.Properties.Set(fmt.Sprintf("%sRef", text.FirstToLower(key)),
				base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}}))
		}

		// 3. Add 'Parameters' to the schema (path, query, header, etc.)
		for _, verb := range resource.VerbsDescription {
			for el := doc.Model.Paths.PathItems.First(); el != nil; el = el.Next() {
				path := el.Value()
				ops := path.GetOperations()
				if ops == nil {
					continue
				}
			}
			path := doc.Model.Paths.PathItems.Value(verb.Path)
			if path == nil {
				return nil, errors, fmt.Errorf("path %s not found", verb.Path)
			}
			ops := path.GetOperations()
			if ops == nil {
				continue
			}
			for op := ops.First(); op != nil; op = op.Next() {
				for _, param := range op.Value().Parameters {
					if _, ok := schema.Properties.Get(param.Name); ok {
						errors = append(errors, fmt.Errorf("parameter %s already exists in schema", param.Name))
						continue
					}

					schema.Properties.Set(param.Name, param.Schema)
					schemaProxyParam := schema.Properties.Value(param.Name)
					if schemaProxyParam == nil {
						return nil, errors, fmt.Errorf("schema proxy for %s is nil", param.Name)
					}
					schemaParam, err := schemaProxyParam.BuildSchema()
					if err != nil {
						return nil, errors, fmt.Errorf("building schema for %s: %w", verb.Path, err)
					}
					schemaParam.Description = fmt.Sprintf("PARAMETER: %s, VERB: %s - %s", param.In, text.CapitaliseFirstLetter(op.Key()), param.Description)
				}
			}
		}

		// If at this point the schema is nil, a fatal error is returned
		if schema == nil {
			return nil, errors, fmt.Errorf("schema is nil for %s", verb.Path)
		}

		// 4. Add the identifiers to the properties map
		for _, identifier := range identifiers {
			_, ok := schema.Properties.Get(identifier)
			if !ok {
				schema.Properties.Set(identifier, base.CreateSchemaProxy(&base.Schema{
					Description: fmt.Sprintf("IDENTIFIER: %s", identifier),
					Type:        []string{"string"},
				}))
			}
		}

		byteSchema, err := generation.GenerateJsonSchemaFromSchemaProxy(base.CreateSchemaProxy(schema))
		if err != nil {
			// If there is an error generating the JSON schema, we return it as a fatal error
			return nil, errors, err
		}

		specByteSchema[resource.Kind] = byteSchema
	}

	// Status schema generation

	if len(resource.Identifiers) == 0 && len(resource.AdditionalStatusFields) == 0 {
		fmt.Println("No identifiers or additional status fields defined in RestDefinition (Empty status)")
	}

	var statusByteSchema []byte

	// Create an ordered property map
	propMap := orderedmap.New[string, *base.SchemaProxy]()

	allStatusFields := append(identifiers, resource.AdditionalStatusFields...)

	responseSchema, err := extractSchemaForAction(doc, resource.VerbsDescription, "get")
	if err != nil { // Note: extractSchemaForAction returns nil, nil if verb is not found
		errors = append(errors, fmt.Errorf("schema validation warning: %w", err))
	}

	// fallback to "findby" if "get" is not found
	if responseSchema == nil {
		responseSchema, err = extractSchemaForAction(doc, resource.VerbsDescription, "findby")
		if err != nil { // Note: extractSchemaForAction returns nil, nil if verb is not found
			errors = append(errors, fmt.Errorf("schema validation warning: %w", err))
		}
	}

	if responseSchema == nil && len(allStatusFields) > 0 {
		// It may be that the resource does not have a GET or FINDBY action defined in the OpenAPI spec
		// In addition, it may be that, in some cases, it make sense to not have a status for this resource
		errors = append(errors, fmt.Errorf("failed to find a GET or FINDBY response schema for status generation"))
	}

	// Add the identifiers and additional status fields to the properties map
	for _, fieldName := range allStatusFields {
		if responseSchema != nil && responseSchema.Properties != nil {
			fieldSchemaProxy := responseSchema.Properties.Value(fieldName)
			if fieldSchemaProxy != nil {
				propMap.Set(fieldName, fieldSchemaProxy)
				continue
			}
		}

		errors = append(errors, fmt.Errorf("status field '%s' defined in RestDefinition not found in GET or FINDBY response schema, defaulting to string", fieldName))
		// Here, instead, a fatal error could be returned (to be decided)
		propMap.Set(fieldName, base.CreateSchemaProxy(&base.Schema{
			Type: []string{"string"},
		}))
	}

	// Create a schema proxy with the properties map
	schemaProxy := base.CreateSchemaProxy(&base.Schema{
		Type:       []string{"object"},
		Properties: propMap,
	})

	statusSchema, err := schemaProxy.BuildSchema()
	if err != nil {
		return nil, errors, fmt.Errorf("building status schema for %s: %w", identifiers, err)
	}

	statusByteSchema, err = generation.GenerateJsonSchemaFromSchemaProxy(base.CreateSchemaProxy(statusSchema))
	if err != nil {
		return nil, errors, err
	}

	g = &OASSchemaGenerator{
		specByteSchema:   specByteSchema[resource.Kind],
		statusByteSchema: statusByteSchema,
		secByteSchema:    secByteSchema,
	}

	// could this validation be moved upper in the function? (to be decided)
	if len(allStatusFields) > 0 {
		validationErrs := validateSchemas(doc, resource.VerbsDescription)
		errors = append(errors, validationErrs...)

		// prints to be removed
		if len(validationErrs) > 0 {
			fmt.Printf("Schema validation completed with %d validation errors.\n", len(validationErrs))
			fmt.Println("Errors:")
			for _, err := range validationErrs {
				fmt.Println("-", err)
			}
		} else {
			fmt.Println("No schema validation errors found.")
		}
	}

	// At this point, g contains the generated schema and any validation errors/warnings but no fatal errors.
	return g, errors, nil
}

func validateSchemas(doc *libopenapi.DocumentModel[v3.Document], verbs []definitionv1alpha1.VerbsDescription) []error {
	var errors []error

	// Step 1: Discover Available Actions
	availableActions := make(map[string]bool)
	for _, verb := range verbs {
		availableActions[verb.Action] = true
	}

	// Step 2: Determine the "Source of Truth" Schema (either "get" or "findby")
	baseAction := ""
	if availableActions["get"] {
		baseAction = "get"
	} else if availableActions["findby"] {
		baseAction = "findby"
	}

	if baseAction == "" {
		errors = append(errors, fmt.Errorf("schema validation warning: no 'get' or 'findby' action found to serve as a base for schema validation"))
		return errors
	}

	// Step 3: Compare Other Actions Against the Source of Truth
	for _, actionToCompare := range []string{"create", "update"} {
		if availableActions[actionToCompare] {
			errors = append(errors, compareActionResponseSchemas(doc, verbs, actionToCompare, baseAction)...)
		}
	}
	if baseAction == "get" && availableActions["findby"] {
		errors = append(errors, compareActionResponseSchemas(doc, verbs, "findby", baseAction)...)
	}

	return errors
}

func compareActionResponseSchemas(doc *libopenapi.DocumentModel[v3.Document], verbs []definitionv1alpha1.VerbsDescription, action1, action2 string) []error {
	var errors []error

	schema2, err := extractSchemaForAction(doc, verbs, action2)
	if err != nil {
		errors = append(errors, fmt.Errorf("schema validation warning: error when calling extractSchemaForAction for action %s: %w", action2, err))
	}
	if schema2 == nil {
		errors = append(errors, fmt.Errorf("Schema for action %s is nil, cannot compare.\n", action2))
	}

	schema1, err := extractSchemaForAction(doc, verbs, action1)
	if err != nil {
		errors = append(errors, fmt.Errorf("schema validation warning: error when calling extractSchemaForAction for action %s: %w", action1, err))
	}
	if schema1 == nil {
		errors = append(errors, fmt.Errorf("Schema for action %s is nil, cannot compare.\n", action1))
	}

	return compareSchemas(".", schema1, schema2)
}

// to ensure that the fields in common are type-compatible
// we just return a list of errors (warnings) if there are any mismatches
func compareSchemas(path string, schema1, schema2 *base.Schema) []error {
	var errors []error

	// Handle cases where schemas themselves are nil
	if schema1 == nil && schema2 == nil {
		return nil // Both are nil, consider them compatible for this path
	}
	if schema1 == nil {
		return []error{fmt.Errorf("schema validation warning: first schema is nil at path '%s'", path)}
	}
	if schema2 == nil {
		return []error{fmt.Errorf("schema validation warning: second schema is nil at path '%s'", path)}
	}

	// Base case: If both schemas do not have properties, compare their types directly.
	// This handles primitives inside arrays or simple top-level primitives.
	schema1HasProps := schema1.Properties != nil && schema1.Properties.Len() > 0
	schema2HasProps := schema2.Properties != nil && schema2.Properties.Len() > 0

	// If both schemas do not have properties, we can compare their types directly.
	if !schema1HasProps && !schema2HasProps {
		if !areTypesCompatible(schema1.Type, schema2.Type) {
			errors = append(errors, fmt.Errorf("schema validation warning: type mismatch at path '%s'. First schema types are '%v', second are '%v'", path, schema1.Type, schema2.Type))
		}
		return errors
	}

	// If one has properties and the other doesn't, it's an incompatibility
	if schema1HasProps && !schema2HasProps {
		errors = append(errors, fmt.Errorf("schema validation warning: first schema has properties but second does not at path '%s'", path))
		// We can still try to compare common fields if we want, but for now, just report the error and return.
		return errors
	}
	if !schema1HasProps && schema2HasProps {
		errors = append(errors, fmt.Errorf("schema validation warning: second schema has properties but first does not at path '%s'", path))
		return errors
	}

	// If we reach here, both schemas have properties, so we can iterate.
	// Loop through properties of schema1 and compare with schema2
	for pair := schema1.Properties.First(); pair != nil; pair = pair.Next() {
		propName := pair.Key()
		propSchemaProxy1 := pair.Value()

		propSchemaProxy2 := schema2.Properties.Value(propName)
		if propSchemaProxy2 == nil {
			// Field from schema1 doesn't exist in schema2, so we skip it.
			continue
		}

		propSchema1, err1 := propSchemaProxy1.BuildSchema()
		propSchema2, err2 := propSchemaProxy2.BuildSchema()

		if err1 != nil {
			errors = append(errors, fmt.Errorf("schema validation warning: error building schema for property '%s' in first schema: %w", buildPath(path, propName), err1))
			continue
		}
		if err2 != nil {
			errors = append(errors, fmt.Errorf("schema validation warning: error building schema for property '%s' in second schema: %w", buildPath(path, propName), err2))
			continue
		}

		// currentPath is the path to the current property being compared
		// It is built by appending the property name to the current path.
		currentPath := buildPath(path, propName)

		if !areTypesCompatible(propSchema1.Type, propSchema2.Type) {
			errors = append(errors, fmt.Errorf("schema validation warning: type mismatch for field '%s'. First schema types are '%v', second are '%v'", currentPath, propSchema1.Type, propSchema2.Type))
			continue
		}

		// If we reached here, the types are compatible but it may be that they are complex types (object or array).
		// We need to check if they are objects or arrays and handle them accordingly.
		// So, at this point, they have the shape of:
		// - ["object", "null"] vs ["object", "null"] or ["object"] vs ["object"]
		// - ["array", "null"] vs ["array", "null"] or ["array"] vs ["array"]
		switch getPrimaryType(propSchema1.Type) {
		case "object":
			// recursive call to compareSchemas for nested objects
			errors = append(errors, compareSchemas(currentPath, propSchema1, propSchema2)...)
		case "array":
			if propSchema1.Items != nil && propSchema2.Items != nil {
				items1, err1 := propSchema1.Items.A.BuildSchema()
				items2, err2 := propSchema2.Items.A.BuildSchema()

				if err1 != nil {
					errors = append(errors, fmt.Errorf("schema validation warning: error building schema for array item at path '%s' in first schema: %w", currentPath, err1))
				}
				if err2 != nil {
					errors = append(errors, fmt.Errorf("schema validation warning: error building schema for array item at path '%s' in second schema: %w", currentPath, err2))
				}

				if items1 != nil && items2 != nil {
					errors = append(errors, compareSchemas(currentPath, items1, items2)...)
				} else if items1 == nil && items2 != nil {
					errors = append(errors, fmt.Errorf("schema validation warning: array item schema is nil for first schema at path '%s'", currentPath))
				} else if items1 != nil && items2 == nil {
					errors = append(errors, fmt.Errorf("schema validation warning: array item schema is nil for second schema at path '%s'", currentPath))
				}
			} else if propSchema1.Items != nil && propSchema2.Items == nil {
				errors = append(errors, fmt.Errorf("schema validation warning: second schema has no items for array at path '%s'", currentPath))
			} else if propSchema1.Items == nil && propSchema2.Items != nil {
				errors = append(errors, fmt.Errorf("schema validation warning: first schema has no items for array at path '%s'", currentPath))
			}
		}
	}

	return errors
}

func buildPath(base, field string) string {
	if base == "." {
		return field
	}
	return fmt.Sprintf("%s.%s", base, field)
}

// getPrimaryType extracts the FIRST non-"null" type from a slice of types (can be also "object" and "array").
// If only "null" is present or the slice is empty, it returns an empty string.
// Note that in the context of OASGen we do not handle type like the following:
// type: ["string", "number", "null"]
// meaning that we do not support multiple types in the same field.
// in the case of multiple non-null types, we consider the first one as the primary type.
func getPrimaryType(types []string) string {
	for _, t := range types {
		if t != "null" {
			return t
		}
	}
	return ""
}

// areTypesCompatible checks if two slices of types are compatible based on their primary non-null type.
// It considers types compatible if:
//  1. Both have the same primary non-null type.
//  2. One has a primary non-null type and the other has no primary non-null type (i.e., only "null" or empty),
//     and the one with the primary type also explicitly allows "null".
//  3. Both have no primary non-null type (i.e., both are only "null" or empty).
func areTypesCompatible(types1, types2 []string) bool {
	primaryType1 := getPrimaryType(types1)
	primaryType2 := getPrimaryType(types2)

	// Case 1: Both have a primary type. They must be identical.
	if primaryType1 != "" && primaryType2 != "" {
		// to be removed
		fmt.Printf("Comparing primary types: '%s' and '%s'\n", primaryType1, primaryType2)
		if primaryType1 != primaryType2 {
			fmt.Printf("Primary types '%s' and '%s' are not compatible.\n", primaryType1, primaryType2)
		} else {
			fmt.Printf("Primary types '%s' and '%s' are compatible.\n", primaryType1, primaryType2)
		}
		return primaryType1 == primaryType2
	}

	// Case 2: One has a primary type, the other doesn't.
	// This is compatible if the one with the primary type also allows null,
	// or if the one without a primary type is effectively "any" (empty slice).
	if primaryType1 != "" && primaryType2 == "" {
		// Check if types1 explicitly allows null
		fmt.Printf("Primary type 1: '%s', checking if it allows null\n", primaryType1)
		for _, t := range types1 {
			if t == "null" {
				fmt.Printf("Primary type 1 '%s' allows null, compatible with empty types2\n", primaryType1)
				return true
			}
		}
		fmt.Printf("Primary type 1 '%s' does not allow null, incompatible with empty types2\n", primaryType1)
		return false // types1 has a primary type but doesn't allow null, and types2 is only null/empty
	}

	if primaryType1 == "" && primaryType2 != "" {
		// Check if types2 explicitly allows null
		fmt.Printf("Primary type 2: '%s', checking if it allows null\n", primaryType2)
		for _, t := range types2 {
			if t == "null" {
				fmt.Printf("Primary type 2 '%s' allows null, compatible with empty types1\n", primaryType2)
				return true
			}
		}
		fmt.Printf("Primary type 2 '%s' does not allow null, incompatible with empty types1\n", primaryType2)
		return false // types2 has a primary type but doesn't allow null, and types1 is only null/empty
	}

	// Case 3: Both have no primary type (i.e., both are only "null" or empty).
	fmt.Printf("Both types are empty or only 'null', they are compatible.\n")
	return true
}

func extractSchemaForAction(doc *libopenapi.DocumentModel[v3.Document], verbs []definitionv1alpha1.VerbsDescription, targetAction string) (*base.Schema, error) {
	for _, verb := range verbs {
		if !strings.EqualFold(verb.Action, targetAction) {
			continue
		}

		fmt.Printf("==============\n")
		fmt.Printf("Processing verb: %s %s\n", verb.Method, verb.Path)
		fmt.Printf("Target action: %s\n", targetAction)

		path := doc.Model.Paths.PathItems.Value(verb.Path)
		if path == nil {
			continue
		}

		ops := path.GetOperations()
		if ops == nil {
			continue
		}

		op := ops.Value(strings.ToLower(verb.Method))
		if op == nil {
			continue
		}

		if op.Responses == nil {
			continue
		}

		// Check for 200 OK or 201 Created
		for _, code := range []int{http.StatusOK, http.StatusCreated} {
			resp := op.Responses.Codes.Value(fmt.Sprintf("%d", code))
			if resp == nil {
				continue
			}

			if resp.Content == nil {
				continue
			}

			mediaType := resp.Content.Value("application/json")
			if mediaType == nil || mediaType.Schema == nil {
				continue
			}

			schemaProxy := mediaType.Schema
			s, err := schemaProxy.BuildSchema()
			if err != nil {
				return nil, err
			}

			fmt.Printf("Found schema for action %s\n", targetAction)

			if strings.EqualFold(targetAction, "findby") && s.Items != nil {
				return s.Items.A.BuildSchema()
			}
			return s, nil
		}
	}
	return nil, nil // Verb not found, but not an error
}

// func PopulateFromAllOf() is a method that populates the schema with the properties from the allOf field.
// the recursive function to populate the schema with the properties from the allOf field.
func populateFromAllOf(schema *base.Schema) error {
	if len(schema.Type) > 0 && schema.Type[0] == "array" {
		if schema.Items != nil {
			if schema.Items.N == 0 {
				sch, err := schema.Items.A.BuildSchema()
				if err != nil {
					return err
				}

				populateFromAllOf(sch)
			}
		}
		return nil
	}
	for prop := schema.Properties.First(); prop != nil; prop = prop.Next() {
		sch, err := prop.Value().BuildSchema()
		if err != nil {
			return err
		}
		populateFromAllOf(sch)
	}
	for _, proxy := range schema.AllOf {
		propSchema, err := proxy.BuildSchema()
		populateFromAllOf(propSchema)
		if err != nil {
			return err
		}
		// Iterate over the properties of the schema with First() and Next()
		for prop := propSchema.Properties.First(); prop != nil; prop = prop.Next() {
			if schema.Properties == nil {
				schema.Properties = orderedmap.New[string, *base.SchemaProxy]()
			}
			// Add the property to the schema
			schema.Properties.Set(prop.Key(), prop.Value())
		}
	}
	return nil
}

func (g *OASSchemaGenerator) OASSpecJsonSchemaGetter() crdgen.JsonSchemaGetter {
	return &oasSpecJsonSchemaGetter{
		g: g,
	}
}

var _ crdgen.JsonSchemaGetter = (*oasSpecJsonSchemaGetter)(nil)

type oasSpecJsonSchemaGetter struct {
	g *OASSchemaGenerator
}

func (a *oasSpecJsonSchemaGetter) Get() ([]byte, error) {
	return a.g.specByteSchema, nil
}

func (g *OASSchemaGenerator) OASStatusJsonSchemaGetter() crdgen.JsonSchemaGetter {
	return &oasStatusJsonSchemaGetter{
		g: g,
	}
}

var _ crdgen.JsonSchemaGetter = (*oasStatusJsonSchemaGetter)(nil)

type oasStatusJsonSchemaGetter struct {
	g *OASSchemaGenerator
}

func (a *oasStatusJsonSchemaGetter) Get() ([]byte, error) {
	return a.g.statusByteSchema, nil
}

func (g *OASSchemaGenerator) OASAuthJsonSchemaGetter(secSchemaName string) crdgen.JsonSchemaGetter {
	return &oasAuthJsonSchemaGetter{
		g:             g,
		secSchemaName: secSchemaName,
	}
}

var _ crdgen.JsonSchemaGetter = (*oasAuthJsonSchemaGetter)(nil)

type oasAuthJsonSchemaGetter struct {
	g             *OASSchemaGenerator
	secSchemaName string
}

func (a *oasAuthJsonSchemaGetter) Get() ([]byte, error) {
	return a.g.secByteSchema[a.secSchemaName], nil
}

var _ crdgen.JsonSchemaGetter = (*staticJsonSchemaGetter)(nil)

func StaticJsonSchemaGetter() crdgen.JsonSchemaGetter {
	return &staticJsonSchemaGetter{}
}

type staticJsonSchemaGetter struct {
}

func (f *staticJsonSchemaGetter) Get() ([]byte, error) {
	return nil, nil
}
