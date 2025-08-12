package oas2jsonschema

import (
	"fmt"
	"testing"

	definitionv1alpha1 "github.com/krateoplatformops/oasgen-provider/apis/restdefinitions/v1alpha1"
	"github.com/stretchr/testify/assert"
)

// TestAreTypesCompatible
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

// TestCompareSchemas
func TestCompareSchemas(t *testing.T) {
	testCases := []struct {
		name        string
		schema1     *Schema
		schema2     *Schema
		expectErr   bool
		errCount    int
		errContains string
	}{
		{
			name: "Compatible simple schemas",
			schema1: &Schema{
				Properties: []Property{
					{Name: "id", Schema: &Schema{Type: []string{"string"}}},
					{Name: "value", Schema: &Schema{Type: []string{"integer"}}},
				},
			},
			schema2: &Schema{
				Properties: []Property{
					{Name: "id", Schema: &Schema{Type: []string{"string"}}},
					{Name: "value", Schema: &Schema{Type: []string{"integer"}}},
					{Name: "extra", Schema: &Schema{Type: []string{"boolean"}}}, // Extra field in schema2 is ignored
				},
			},
			expectErr: false,
		},
		{
			name: "Incompatible simple schemas (type mismatch)",
			schema1: &Schema{
				Properties: []Property{
					{Name: "id", Schema: &Schema{Type: []string{"string"}}},
					{Name: "value", Schema: &Schema{Type: []string{"integer"}}},
				},
			},
			schema2: &Schema{
				Properties: []Property{
					{Name: "id", Schema: &Schema{Type: []string{"string"}}},
					{Name: "value", Schema: &Schema{Type: []string{"string"}}}, // Mismatch here
				},
			},
			expectErr: true,
			errCount:  1,
		},
		{
			name: "Compatible nested schemas",
			schema1: &Schema{
				Properties: []Property{
					{Name: "user", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "id", Schema: &Schema{Type: []string{"integer"}}},
						},
					}},
				},
			},
			schema2: &Schema{
				Properties: []Property{
					{Name: "user", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "id", Schema: &Schema{Type: []string{"integer"}}},
							{Name: "name", Schema: &Schema{Type: []string{"string"}}},
						},
					}},
				},
			},
			expectErr: false,
		},
		{
			name: "Incompatible nested schemas",
			schema1: &Schema{
				Properties: []Property{
					{Name: "user", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "id", Schema: &Schema{Type: []string{"string"}}}, // Mismatch
						},
					}},
				},
			},
			schema2: &Schema{
				Properties: []Property{
					{Name: "user", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "id", Schema: &Schema{Type: []string{"integer"}}},
						},
					}},
				},
			},
			expectErr: true,
			errCount:  1,
		},
		{
			name: "Compatible array of objects",
			schema1: &Schema{
				Properties: []Property{
					{Name: "items", Schema: &Schema{
						Type: []string{"array"},
						Items: &Schema{
							Type: []string{"object"},
							Properties: []Property{
								{Name: "key", Schema: &Schema{Type: []string{"string"}}},
							},
						},
					}},
				},
			},
			schema2: &Schema{
				Properties: []Property{
					{Name: "items", Schema: &Schema{
						Type: []string{"array"},
						Items: &Schema{
							Type: []string{"object"},
							Properties: []Property{
								{Name: "key", Schema: &Schema{Type: []string{"string"}}},
							},
						},
					}},
				},
			},
			expectErr: false,
		},
		{
			name: "Incompatible array of objects",
			schema1: &Schema{
				Properties: []Property{
					{Name: "items", Schema: &Schema{
						Type: []string{"array"},
						Items: &Schema{
							Type: []string{"object"},
							Properties: []Property{
								{Name: "key", Schema: &Schema{Type: []string{"boolean"}}}, // Mismatch
							},
						},
					}},
				},
			},
			schema2: &Schema{
				Properties: []Property{
					{Name: "items", Schema: &Schema{
						Type: []string{"array"},
						Items: &Schema{
							Type: []string{"object"},
							Properties: []Property{
								{Name: "key", Schema: &Schema{Type: []string{"string"}}},
							},
						},
					}},
				},
			},
			expectErr: true,
			errCount:  1,
		},
		{
			name: "Property mismatch (one has props, other does not)",
			schema1: &Schema{
				Properties: []Property{
					{Name: "id", Schema: &Schema{Type: []string{"string"}}},
				},
			},
			schema2: &Schema{
				Type: []string{"string"},
			},
			expectErr:   true,
			errCount:    1,
			errContains: "schema mismatch: response for action 'action1' has properties but response for action 'action2' does not",
		},
		{
			name:      "Incompatible property-less schemas",
			schema1:   &Schema{Type: []string{"string"}},
			schema2:   &Schema{Type: []string{"integer"}},
			expectErr: true,
			errCount:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := compareSchemas(".", tc.schema1, tc.schema2, "action1", "action2")
			if tc.expectErr {
				assert.NotEmpty(t, errs, "Expected errors, but got none")
				assert.Len(t, errs, tc.errCount, "Unexpected number of errors")
				if tc.errContains != "" {
					assert.Contains(t, errs[0].Error(), tc.errContains)
				}
			} else {
				assert.Empty(t, errs, fmt.Sprintf("Expected no errors, but got: %v", errs))
			}
		})
	}
}

// TestValidateSchemas
func TestValidateSchemas(t *testing.T) {
	config := DefaultGeneratorConfig()

	// Schema for successful GET/FINDBY responses
	getSchema := &Schema{
		Type: []string{"object"},
		Properties: []Property{
			{Name: "id", Schema: &Schema{Type: []string{"integer"}}},
			{Name: "name", Schema: &Schema{Type: []string{"string"}}},
		},
	}

	// Schema for successful CREATE/UPDATE responses
	createSchema := &Schema{
		Type: []string{"object"},
		Properties: []Property{
			{Name: "id", Schema: &Schema{Type: []string{"integer"}}},
			{Name: "name", Schema: &Schema{Type: []string{"string"}}},
			{Name: "status", Schema: &Schema{Type: []string{"string"}}}, // Extra field is ok
		},
	}

	// Schema with a type mismatch
	mismatchedSchema := &Schema{
		Type: []string{"object"},
		Properties: []Property{
			{Name: "id", Schema: &Schema{Type: []string{"string"}}}, // Mismatch: should be integer
			{Name: "name", Schema: &Schema{Type: []string{"string"}}},
		},
	}

	testCases := []struct {
		name      string
		doc       OASDocument
		verbs     []definitionv1alpha1.VerbsDescription
		expectErr bool
		errCode   ValidationCode
	}{
		{
			name: "Compatible schemas",
			doc: &mockOASDocument{
				Paths: map[string]*mockPathItem{
					"/items": {
						Ops: map[string]Operation{
							"get": &mockOperation{
								Responses: map[int]ResponseInfo{
									200: {Content: map[string]*Schema{"application/json": getSchema}},
								},
							},
							"post": &mockOperation{
								Responses: map[int]ResponseInfo{
									201: {Content: map[string]*Schema{"application/json": createSchema}},
								},
							},
						},
					},
				},
			},
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: ActionGet, Path: "/items", Method: "get"},
				{Action: ActionCreate, Path: "/items", Method: "post"},
			},
			expectErr: false,
		},
		{
			name: "Incompatible schemas",
			doc: &mockOASDocument{
				Paths: map[string]*mockPathItem{
					"/items": {
						Ops: map[string]Operation{
							"get": &mockOperation{
								Responses: map[int]ResponseInfo{
									200: {Content: map[string]*Schema{"application/json": getSchema}},
								},
							},
							"post": &mockOperation{
								Responses: map[int]ResponseInfo{
									201: {Content: map[string]*Schema{"application/json": mismatchedSchema}},
								},
							},
						},
					},
				},
			},
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: ActionGet, Path: "/items", Method: "get"},
				{Action: ActionCreate, Path: "/items", Method: "post"},
			},
			expectErr: true,
			errCode:   CodeTypeMismatch,
		},
		{
			name: "Missing base action (get or findby)",
			doc: &mockOASDocument{
				Paths: map[string]*mockPathItem{
					"/items": {
						Ops: map[string]Operation{
							"post": &mockOperation{
								Responses: map[int]ResponseInfo{
									201: {Content: map[string]*Schema{"application/json": createSchema}},
								},
							},
						},
					},
				},
			},
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: ActionCreate, Path: "/items", Method: "post"},
			},
			expectErr: true,
			errCode:   CodeMissingBaseAction,
		},
		{
			name: "Base action schema is missing",
			doc: &mockOASDocument{
				Paths: map[string]*mockPathItem{
					"/items": {
						Ops: map[string]Operation{
							"get": &mockOperation{ // No 200 response
								Responses: map[int]ResponseInfo{},
							},
							"post": &mockOperation{
								Responses: map[int]ResponseInfo{
									201: {Content: map[string]*Schema{"application/json": createSchema}},
								},
							},
						},
					},
				},
			},
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: ActionGet, Path: "/items", Method: "get"},
				{Action: ActionCreate, Path: "/items", Method: "post"},
			},
			expectErr: true,
			errCode:   CodeActionSchemaMissing,
		},
		{
			name: "Compared action schema is missing",
			doc: &mockOASDocument{
				Paths: map[string]*mockPathItem{
					"/items": {
						Ops: map[string]Operation{
							"get": &mockOperation{
								Responses: map[int]ResponseInfo{
									200: {Content: map[string]*Schema{"application/json": getSchema}},
								},
							},
							// No "post" operation for the create action
						},
					},
				},
			},
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: ActionGet, Path: "/items", Method: "get"},
				{Action: ActionCreate, Path: "/items", Method: "post"}, // This action will fail to be extracted
			},
			expectErr: true,
			errCode:   CodeActionSchemaMissing,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := validateSchemas(tc.doc, tc.verbs, config)
			if tc.expectErr {
				assert.NotEmpty(t, errs, "Expected errors, but got none")
				validationErr, ok := errs[0].(SchemaValidationError)
				assert.True(t, ok, "Error is not of type SchemaValidationError")
				assert.Equal(t, tc.errCode, validationErr.Code)
			} else {
				assert.Empty(t, errs, fmt.Sprintf("Expected no errors, but got: %v", errs))
			}
		})
	}
}

// TestExtractSchemaForAction
func TestExtractSchemaForAction(t *testing.T) {
	defaultConfig := DefaultGeneratorConfig()
	itemSchema := &Schema{Type: []string{"object"}, Properties: []Property{{Name: "id", Schema: &Schema{Type: []string{"integer"}}}}}
	listSchema := &Schema{Type: []string{"array"}, Items: itemSchema}
	altSchema := &Schema{Type: []string{"object"}, Properties: []Property{{Name: "alt-id", Schema: &Schema{Type: []string{"string"}}}}}

	doc := &mockOASDocument{
		Paths: map[string]*mockPathItem{
			"/items": {
				Ops: map[string]Operation{
					"get": &mockOperation{
						Responses: map[int]ResponseInfo{
							200: {Content: map[string]*Schema{"application/json": itemSchema}},
						},
					},
				},
			},
			"/items/search": {
				Ops: map[string]Operation{
					"get": &mockOperation{
						Responses: map[int]ResponseInfo{
							200: {Content: map[string]*Schema{"application/json": listSchema}},
						},
					},
				},
			},
			"/items/alt": {
				Ops: map[string]Operation{
					"get": &mockOperation{
						Responses: map[int]ResponseInfo{
							200: {Content: map[string]*Schema{"application/vnd.api+json": altSchema}},
						},
					},
				},
			},
		},
	}

	testCases := []struct {
		name           string
		config         *GeneratorConfig
		verbs          []definitionv1alpha1.VerbsDescription
		targetAction   string
		expectErr      bool
		expectNil      bool
		expectArray    bool // True if we expect the .Items schema from a list
		expectedError  string
		expectedSchema *Schema
	}{
		{
			name:   "Extract schema for 'get' action",
			config: defaultConfig,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: ActionGet, Path: "/items", Method: "get"},
			},
			targetAction:   ActionGet,
			expectNil:      false,
			expectedSchema: itemSchema,
		},
		{
			name:   "Extract schema for 'findby' action (unwraps array)",
			config: defaultConfig,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: ActionFindBy, Path: "/items/search", Method: "get"},
			},
			targetAction:   ActionFindBy,
			expectNil:      false,
			expectArray:    true,
			expectedSchema: itemSchema,
		},
		{
			name:   "Action not found",
			config: defaultConfig,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: ActionGet, Path: "/items", Method: "get"},
			},
			targetAction:  ActionUpdate, // 'update' is not in the verbs list
			expectErr:     true,
			expectNil:     true,
			expectedError: "action 'update' not defined in resource verbs",
		},
		{
			name:   "Path not found in spec",
			config: defaultConfig,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: ActionGet, Path: "/nonexistent", Method: "get"},
			},
			targetAction:  ActionGet,
			expectErr:     true,
			expectNil:     true,
			expectedError: "path '/nonexistent' not found in OAS document",
		},
		{
			name: "Extract schema for alternative MIME type",
			config: &GeneratorConfig{
				SuccessCodes:      []int{200},
				AcceptedMIMETypes: []string{"application/json", "application/vnd.api+json"},
			},
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: ActionGet, Path: "/items/alt", Method: "get"},
			},
			targetAction:   ActionGet,
			expectNil:      false,
			expectedSchema: altSchema,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			schema, err := extractSchemaForAction(doc, tc.verbs, tc.targetAction, tc.config)

			if tc.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
			}

			if tc.expectNil {
				assert.Nil(t, schema)
			} else {
				assert.NotNil(t, schema)
				assert.Equal(t, tc.expectedSchema, schema)
			}
		})
	}
}

func TestDetermineBaseAction(t *testing.T) {
	testCases := []struct {
		name           string
		verbs          []definitionv1alpha1.VerbsDescription
		expectedAction string
		expectErr      bool
	}{
		{
			name:           "should return 'get' when 'get' is available",
			verbs:          []definitionv1alpha1.VerbsDescription{{Action: ActionGet}, {Action: ActionCreate}},
			expectedAction: ActionGet,
			expectErr:      false,
		},
		{
			name:           "should return 'get' when both 'get' and 'findby' are available",
			verbs:          []definitionv1alpha1.VerbsDescription{{Action: ActionGet}, {Action: ActionFindBy}},
			expectedAction: ActionGet,
			expectErr:      false,
		},
		{
			name:           "should return 'findby' when only 'findby' is available",
			verbs:          []definitionv1alpha1.VerbsDescription{{Action: ActionFindBy}, {Action: ActionUpdate}},
			expectedAction: ActionFindBy,
			expectErr:      false,
		},
		{
			name:      "should return an error when no base action is available",
			verbs:     []definitionv1alpha1.VerbsDescription{{Action: ActionCreate}, {Action: ActionUpdate}},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			action, err := determineBaseAction(tc.verbs)

			if tc.expectErr {
				assert.Error(t, err)
				validationErr, ok := err.(SchemaValidationError)
				assert.True(t, ok)
				assert.Equal(t, CodeMissingBaseAction, validationErr.Code)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedAction, action)
			}
		})
	}
}

func TestCompareSchemas_NilCases(t *testing.T) {
	t.Run("should return no error if both schemas are nil", func(t *testing.T) {
		errs := compareSchemas(".", nil, nil, "action1", "action2")
		assert.Empty(t, errs)
	})

	t.Run("should return an error if first schema is nil", func(t *testing.T) {
		errs := compareSchemas(".", nil, &Schema{}, "action1", "action2")
		assert.NotEmpty(t, errs)
		assert.Contains(t, errs[0].Error(), "first schema is nil")
	})

	t.Run("should return an error if second schema is nil", func(t *testing.T) {
		errs := compareSchemas(".", &Schema{}, nil, "action1", "action2")
		assert.NotEmpty(t, errs)
		assert.Contains(t, errs[0].Error(), "second schema is nil")
	})
}

func TestCompareSchemas_ArrayCases(t *testing.T) {
	t.Run("should return error if one array schema is nil", func(t *testing.T) {
		schema1 := &Schema{
			Properties: []Property{
				{Name: "tags", Schema: &Schema{Type: []string{"array"}}},
			},
		}
		schema2 := &Schema{
			Properties: []Property{
				{Name: "tags", Schema: nil}, // Nil sub-schema
			},
		}

		errs := compareSchemas(".", schema1, schema2, "action1", "action2")
		assert.NotEmpty(t, errs)
		assert.Equal(t, CodePropertyMismatch, errs[0].(SchemaValidationError).Code)
		assert.Contains(t, errs[0].Error(), "schema for property 'tags' is nil")
	})

	t.Run("should return error if second array schema is missing items", func(t *testing.T) {
		schema1 := &Schema{
			Properties: []Property{
				{Name: "tags", Schema: &Schema{Type: []string{"array"}, Items: &Schema{Type: []string{"string"}}}},
			},
		}
		schema2 := &Schema{
			Properties: []Property{
				{Name: "tags", Schema: &Schema{Type: []string{"array"}, Items: nil}},
			},
		}

		errs := compareSchemas(".", schema1, schema2, "action1", "action2")
		assert.NotEmpty(t, errs)
		assert.Equal(t, CodeMissingArrayItems, errs[0].(SchemaValidationError).Code)
		assert.Contains(t, errs[0].Error(), "second schema has no items for array")
	})

	t.Run("should return error if first array schema is missing items", func(t *testing.T) {
		schema1 := &Schema{
			Properties: []Property{
				{Name: "tags", Schema: &Schema{Type: []string{"array"}, Items: nil}},
			},
		}
		schema2 := &Schema{
			Properties: []Property{
				{Name: "tags", Schema: &Schema{Type: []string{"array"}, Items: &Schema{Type: []string{"string"}}}},
			},
		}

		errs := compareSchemas(".", schema1, schema2, "action1", "action2")
		assert.NotEmpty(t, errs)
		assert.Equal(t, CodeMissingArrayItems, errs[0].(SchemaValidationError).Code)
		assert.Contains(t, errs[0].Error(), "first schema has no items for array")
	})
}

func TestValidateSchemas_Complex(t *testing.T) {
	// Define a complex, nested schema to serve as the base
	complexSchemaBase := &Schema{
		Type: []string{"object"},
		Properties: []Property{
			{Name: "id", Schema: &Schema{Type: []string{"integer"}}},
			{Name: "status", Schema: &Schema{Type: []string{"string"}}},
			{Name: "user", Schema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "name", Schema: &Schema{Type: []string{"string"}}},
					{Name: "profile", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "email", Schema: &Schema{Type: []string{"string"}}},
						},
					}},
				},
			}},
			{Name: "tags", Schema: &Schema{
				Type: []string{"array"},
				Items: &Schema{
					Type: []string{"object"},
					Properties: []Property{
						{Name: "key", Schema: &Schema{Type: []string{"string"}}},
						{Name: "value", Schema: &Schema{Type: []string{"string"}}},
					},
				},
			}},
		},
	}

	// Define a second schema with multiple, deeply nested errors
	complexSchemaWithErrors := &Schema{
		Type: []string{"object"},
		Properties: []Property{
			{Name: "id", Schema: &Schema{Type: []string{"integer"}}},
			{Name: "status", Schema: &Schema{Type: []string{"boolean"}}}, // Error 1: Mismatch at root
			{Name: "user", Schema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "name", Schema: &Schema{Type: []string{"string"}}},
					{Name: "profile", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "email", Schema: &Schema{Type: []string{"integer"}}}, // Error 2: Mismatch in nested object
						},
					}},
				},
			}},
			{Name: "tags", Schema: &Schema{
				Type: []string{"array"},
				Items: &Schema{
					Type: []string{"object"},
					Properties: []Property{
						{Name: "key", Schema: &Schema{Type: []string{"string"}}},
						{Name: "value", Schema: &Schema{Type: []string{"boolean"}}}, // Error 3: Mismatch in array of objects
					},
				},
			}},
		},
	}

	doc := &mockOASDocument{
		Paths: map[string]*mockPathItem{
			"/complex": {
				Ops: map[string]Operation{
					"get": &mockOperation{
						Responses: map[int]ResponseInfo{200: {Content: map[string]*Schema{"application/json": complexSchemaBase}}},
					},
					"post": &mockOperation{
						Responses: map[int]ResponseInfo{201: {Content: map[string]*Schema{"application/json": complexSchemaWithErrors}}},
					},
				},
			},
		},
	}

	verbs := []definitionv1alpha1.VerbsDescription{
		{Action: ActionGet, Path: "/complex", Method: "get"},
		{Action: ActionCreate, Path: "/complex", Method: "post"},
	}

	// Act
	errs := validateSchemas(doc, verbs, DefaultGeneratorConfig())

	// Assert
	assert.NotEmpty(t, errs, "Expected validation errors, but got none")
	assert.Len(t, errs, 3, "Expected exactly 3 validation errors")

	// Check for specific errors
	expectedErrors := map[string]ValidationCode{
		"status":             CodeTypeMismatch,
		"user.profile.email": CodeTypeMismatch,
		"tags.value":         CodeTypeMismatch,
	}

	foundErrors := make(map[string]bool)
	for _, err := range errs {
		validationErr, ok := err.(SchemaValidationError)
		assert.True(t, ok, "Error is not of type SchemaValidationError")

		for path, code := range expectedErrors {
			if validationErr.Path == path && validationErr.Code == code {
				foundErrors[path] = true
			}
		}
	}

	assert.Len(t, foundErrors, 3, "Did not find all expected errors")
}
