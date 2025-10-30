package oas2jsonschema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveExcludedSpecFields(t *testing.T) {
	baseSchema := func() *Schema {
		return &Schema{
			Type: []string{"object"},
			Properties: []Property{
				{Name: "pullRequestId", Schema: &Schema{Type: []string{"integer"}}},
				{Name: "maxCommentLength", Schema: &Schema{Type: []string{"integer"}}},
				{Name: "searchCriteria.creatorId", Schema: &Schema{Type: []string{"string"}}},
				{Name: "completionOptions", Schema: &Schema{
					Type: []string{"object"},
					Properties: []Property{
						{Name: "triggeredByAutoComplete", Schema: &Schema{Type: []string{"boolean"}}},
						{Name: "timeout", Schema: &Schema{Type: []string{"integer"}}},
					},
					Required: []string{"timeout"},
				}},
			},
			Required: []string{"pullRequestId", "completionOptions"},
		}
	}

	testCases := []struct {
		name           string
		schema         *Schema
		excludedFields []string
		expectedSchema *Schema
		expectedWarns  int
	}{
		{
			name:           "should remove multiple top-level fields",
			schema:         baseSchema(),
			excludedFields: []string{"pullRequestId", "maxCommentLength"},
			expectedSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "searchCriteria.creatorId", Schema: &Schema{Type: []string{"string"}}},
					{Name: "completionOptions", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "triggeredByAutoComplete", Schema: &Schema{Type: []string{"boolean"}}},
							{Name: "timeout", Schema: &Schema{Type: []string{"integer"}}},
						},
						Required: []string{"timeout"},
					}},
				},
				Required: []string{"completionOptions"},
			},
		},
		{
			name:           "should remove a nested field using dot notation",
			schema:         baseSchema(),
			excludedFields: []string{"completionOptions.triggeredByAutoComplete"},
			expectedSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "pullRequestId", Schema: &Schema{Type: []string{"integer"}}},
					{Name: "maxCommentLength", Schema: &Schema{Type: []string{"integer"}}},
					{Name: "searchCriteria.creatorId", Schema: &Schema{Type: []string{"string"}}},
					{Name: "completionOptions", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "timeout", Schema: &Schema{Type: []string{"integer"}}},
						},
						Required: []string{"timeout"},
					}},
				},
				Required: []string{"pullRequestId", "completionOptions"},
			},
		},
		{
			name:           "should remove a field with a literal dot using bracket notation",
			schema:         baseSchema(),
			excludedFields: []string{"['searchCriteria.creatorId']"},
			expectedSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "pullRequestId", Schema: &Schema{Type: []string{"integer"}}},
					{Name: "maxCommentLength", Schema: &Schema{Type: []string{"integer"}}},
					{Name: "completionOptions", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "triggeredByAutoComplete", Schema: &Schema{Type: []string{"boolean"}}},
							{Name: "timeout", Schema: &Schema{Type: []string{"integer"}}},
						},
						Required: []string{"timeout"},
					}},
				},
				Required: []string{"pullRequestId", "completionOptions"},
			},
		},
		{
			name:           "should handle a mix of notations",
			schema:         baseSchema(),
			excludedFields: []string{"maxCommentLength", "completionOptions.triggeredByAutoComplete", "['searchCriteria.creatorId']"},
			expectedSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "pullRequestId", Schema: &Schema{Type: []string{"integer"}}},
					{Name: "completionOptions", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "timeout", Schema: &Schema{Type: []string{"integer"}}},
						},
						Required: []string{"timeout"},
					}},
				},
				Required: []string{"pullRequestId", "completionOptions"},
			},
		},
		{
			name:           "should do nothing if excludedFields is empty",
			schema:         baseSchema(),
			excludedFields: []string{},
			expectedSchema: baseSchema(),
		},
		{
			name:           "should generate warnings for fields that are not found",
			schema:         baseSchema(),
			excludedFields: []string{"nonExistentField", "completionOptions.nonExistent"},
			expectedSchema: baseSchema(),
			expectedWarns:  2,
		},
		{
			name:           "should generate a warning for invalid path syntax",
			schema:         baseSchema(),
			excludedFields: []string{"['unclosed.bracket"},
			expectedSchema: baseSchema(),
			expectedWarns:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			g := &OASSchemaGenerator{
				generatorConfig: DefaultGeneratorConfig(), // Initialize the config
				resourceConfig: &ResourceConfig{
					ExcludedSpecFields: tc.excludedFields,
				},
			}

			// Act
			warnings := g.removeExcludedSpecFields(tc.schema)

			// Assert
			assert.Equal(t, tc.expectedSchema, tc.schema)
			assert.Len(t, warnings, tc.expectedWarns)
		})
	}
}

func TestRemoveConfiguredFields(t *testing.T) {
	baseSchema := func() *Schema {
		return &Schema{
			Type: []string{"object"},
			Properties: []Property{
				{Name: "apiKey", Schema: &Schema{Type: []string{"string"}}},
				{Name: "auth.token", Schema: &Schema{Type: []string{"string"}}}, // note the dot in the name (literal dot)
				{Name: "credentials", Schema: &Schema{
					Type: []string{"object"},
					Properties: []Property{
						{Name: "user", Schema: &Schema{Type: []string{"string"}}},
						{Name: "pass", Schema: &Schema{Type: []string{"string"}}},
					},
				}},
			},
			Required: []string{"apiKey"},
		}
	}

	testCases := []struct {
		name             string
		schema           *Schema
		configuredFields []ConfigurationField
		expectedSchema   *Schema
		expectedWarns    int
	}{
		{
			name:   "should remove a simple configured field",
			schema: baseSchema(),
			configuredFields: []ConfigurationField{
				{FromOpenAPI: FromOpenAPI{Name: "apiKey"}},
			},
			expectedSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "auth.token", Schema: &Schema{Type: []string{"string"}}},
					{Name: "credentials", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "user", Schema: &Schema{Type: []string{"string"}}},
							{Name: "pass", Schema: &Schema{Type: []string{"string"}}},
						},
					}},
				},
				Required: []string{},
			},
		},
		{
			name:   "should remove a nested configured field",
			schema: baseSchema(),
			configuredFields: []ConfigurationField{
				{FromOpenAPI: FromOpenAPI{Name: "credentials.user"}},
			},
			expectedSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "apiKey", Schema: &Schema{Type: []string{"string"}}},
					{Name: "auth.token", Schema: &Schema{Type: []string{"string"}}},
					{Name: "credentials", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "pass", Schema: &Schema{Type: []string{"string"}}},
						},
					}},
				},
				Required: []string{"apiKey"},
			},
		},
		{
			name:   "should remove a configured field with a literal dot",
			schema: baseSchema(),
			configuredFields: []ConfigurationField{
				{FromOpenAPI: FromOpenAPI{Name: "['auth.token']"}},
			},
			expectedSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "apiKey", Schema: &Schema{Type: []string{"string"}}},
					{Name: "credentials", Schema: &Schema{
						Type: []string{"object"},
						Properties: []Property{
							{Name: "user", Schema: &Schema{Type: []string{"string"}}},
							{Name: "pass", Schema: &Schema{Type: []string{"string"}}},
						},
					}},
				},
				Required: []string{"apiKey"},
			},
		},
		{
			name:             "should do nothing if configuredFields is empty",
			schema:           baseSchema(),
			configuredFields: []ConfigurationField{},
			expectedSchema:   baseSchema(),
		},
		{
			name:   "should generate a warning for a configured field that is not found",
			schema: baseSchema(),
			configuredFields: []ConfigurationField{
				{FromOpenAPI: FromOpenAPI{Name: "nonExistent"}},
			},
			expectedSchema: baseSchema(),
			expectedWarns:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			g := &OASSchemaGenerator{
				generatorConfig: DefaultGeneratorConfig(),
				resourceConfig: &ResourceConfig{
					ConfigurationFields: tc.configuredFields,
				},
			}

			// Act
			warnings := g.removeConfiguredFields(tc.schema)

			// Assert
			require.Equal(t, tc.expectedSchema, tc.schema, "The schema was not modified as expected.")
			assert.Len(t, warnings, tc.expectedWarns)
		})
	}
}

func TestAddParametersToSpec(t *testing.T) {
	baseSchema := &Schema{
		Type:       []string{"object"},
		Properties: []Property{{Name: "existingProp", Schema: &Schema{Type: []string{"string"}}}},
	}

	mockDoc := &mockOASDocument{
		Paths: map[string]*mockPathItem{
			"/path1": {Ops: map[string]Operation{
				"get": &mockOperation{Parameters: []ParameterInfo{
					{Name: "param1", In: "query", Schema: &Schema{Type: []string{"string"}}, Required: true},
					{Name: "commonParam", In: "query", Schema: &Schema{Type: []string{"string"}}},
				}},
			}},
			"/path2": {Ops: map[string]Operation{
				"post": &mockOperation{Parameters: []ParameterInfo{
					{Name: "param2", In: "header", Schema: &Schema{Type: []string{"integer"}}},
					{Name: "commonParam", In: "query", Schema: &Schema{Type: []string{"string"}}},  // Duplicate
					{Name: "existingProp", In: "query", Schema: &Schema{Type: []string{"string"}}}, // Name collision
					{Name: "Authorization", In: "header", Schema: &Schema{Type: []string{"string"}}},
				}},
			}},
		},
	}

	g := &OASSchemaGenerator{
		doc: mockDoc, // Inject the mock document
		resourceConfig: &ResourceConfig{
			Verbs: []Verb{
				{Path: "/path1", Method: "get"},
				{Path: "/path2", Method: "post"},
			},
		},
	}

	schema := baseSchema.deepCopy()
	g.addParametersToSpec(schema)

	// Assertions
	props := schema.Properties
	propMap := make(map[string]Property)
	for _, p := range props {
		propMap[p.Name] = p
	}

	assert.Len(t, props, 4, "Should have existing prop + 3 new params")
	assert.Contains(t, propMap, "param1", "Should have added param1")
	assert.Contains(t, propMap, "param2", "Should have added param2")
	assert.Contains(t, propMap, "commonParam", "Should have added commonParam")
	assert.NotContains(t, propMap, "Authorization", "Should have skipped Authorization header")

	assert.Equal(t, "PARAMETER: query", propMap["param1"].Schema.Description, "Description should be set correctly")
	assert.Equal(t, "PARAMETER: header", propMap["param2"].Schema.Description, "Description should be set correctly")

	assert.Len(t, schema.Required, 1, "Should have 1 required field")
	assert.Equal(t, "param1", schema.Required[0])
}

func TestAddConfigurationRefToSpec(t *testing.T) {
	// Arrange
	schema := &Schema{
		Type:       []string{"object"},
		Properties: []Property{},
		Required:   []string{},
	}

	// Act
	addConfigurationRefToSpec(schema)

	// Assert
	require.Len(t, schema.Properties, 1)
	prop := schema.Properties[0]
	assert.Equal(t, "configurationRef", prop.Name)
	assert.Equal(t, "object", prop.Schema.Type[0])
	assert.Contains(t, prop.Schema.Description, "A reference to the Configuration CR")

	require.Len(t, prop.Schema.Properties, 2)
	assert.Equal(t, "name", prop.Schema.Properties[0].Name)
	assert.Equal(t, "namespace", prop.Schema.Properties[1].Name)

	assert.Equal(t, []string{"name"}, prop.Schema.Required)
	assert.Equal(t, []string{"configurationRef"}, schema.Required)
}

func TestAddIdentifiersToSpec(t *testing.T) {
	testCases := []struct {
		name           string
		initialSchema  *Schema
		identifiers    []string
		expectedSchema *Schema
	}{
		{
			name: "Add a new identifier",
			initialSchema: &Schema{
				Type:       []string{"object"},
				Properties: []Property{{Name: "prop1", Schema: &Schema{Type: []string{"string"}}}},
			},
			identifiers: []string{"id"},
			expectedSchema: &Schema{
				Type: []string{"object"},
				Properties: []Property{
					{Name: "prop1", Schema: &Schema{Type: []string{"string"}}},
					{Name: "id", Schema: &Schema{Description: "IDENTIFIER: id", Type: []string{"string"}}},
				},
			},
		},
		{
			name: "Identifier already exists",
			initialSchema: &Schema{
				Type:       []string{"object"},
				Properties: []Property{{Name: "id", Schema: &Schema{Type: []string{"string"}, Description: "Original Description."}}},
			},
			identifiers: []string{"id"},
			expectedSchema: &Schema{
				Type:       []string{"object"},
				Properties: []Property{{Name: "id", Schema: &Schema{Type: []string{"string"}, Description: "Original Description. (IDENTIFIER: id)"}}},
			},
		},
		{
			name:           "Empty identifiers list",
			initialSchema:  &Schema{Type: []string{"object"}, Properties: []Property{}},
			identifiers:    []string{},
			expectedSchema: &Schema{Type: []string{"object"}, Properties: []Property{}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			addIdentifiersToSpec(tc.initialSchema, tc.identifiers)
			assert.Equal(t, tc.expectedSchema, tc.initialSchema)
		})
	}
}

func TestIsAuthorizationHeader(t *testing.T) {
	testCases := []struct {
		name     string
		param    ParameterInfo
		expected bool
	}{
		{name: "Exact match", param: ParameterInfo{In: "header", Name: "Authorization"}, expected: true},
		{name: "Lowercase", param: ParameterInfo{In: "header", Name: "authorization"}, expected: true},
		{name: "With prefix", param: ParameterInfo{In: "header", Name: "X-Authorization"}, expected: true},
		{name: "Not in header", param: ParameterInfo{In: "query", Name: "Authorization"}, expected: false},
		{name: "Different header", param: ParameterInfo{In: "header", Name: "Content-Type"}, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isAuthorizationHeader(tc.param))
		})
	}
}

// TestBuildSpecSchema ensures all its internal steps (adding parameters, config refs, and removing fields)
// work together correctly to produce a final, valid spec schema.
func TestBuildSpecSchema(t *testing.T) {
	// Arrange
	mockDoc := &mockOASDocument{
		Paths: map[string]*mockPathItem{
			"/items": {
				Ops: map[string]Operation{
					"post": &mockOperation{ // This will be the base schema
						RequestBody: RequestBodyInfo{
							Content: map[string]*Schema{
								"application/json": {
									Type: []string{"object"},
									Properties: []Property{
										{Name: "name", Schema: &Schema{Type: []string{"string"}}},
										{Name: "details", Schema: &Schema{
											Type: []string{"object"},
											Properties: []Property{
												{Name: "color", Schema: &Schema{Type: []string{"string"}}},
												{Name: "weight", Schema: &Schema{Type: []string{"number"}}},
											},
										}},
									},
									Required: []string{"name"},
								},
							},
						},
						Parameters: []ParameterInfo{
							{Name: "X-Tenant-ID", In: "header", Schema: &Schema{Type: []string{"string"}}}, // This will be a configured field
						},
					},
				},
			},
			"/items/{id}": {
				Ops: map[string]Operation{
					"get": &mockOperation{
						Parameters: []ParameterInfo{
							{Name: "id", In: "path", Schema: &Schema{Type: []string{"string"}}}, // This will be an identifier
							{Name: "verbose", In: "query", Schema: &Schema{Type: []string{"boolean"}}},
							{Name: "Authorization", In: "header", Schema: &Schema{Type: []string{"string"}}}, // Should be skipped
						},
					},
				},
			},
		},
		securitySchemes: []SecuritySchemeInfo{
			{Name: "BearerAuth", Type: SchemeTypeHTTP, Scheme: "bearer"},
		},
	}

	generatorConfig := DefaultGeneratorConfig()

	resourceConfig := &ResourceConfig{
		Verbs: []Verb{
			{Action: "create", Method: "post", Path: "/items"},
			{Action: "get", Method: "get", Path: "/items/{id}"},
		},
		Identifiers: []string{"fieldOnlyInResponseAndShouldNotBeAdded", "name"},
		ConfigurationFields: []ConfigurationField{
			{FromOpenAPI: FromOpenAPI{Name: "X-Tenant-ID", In: "header"}},
		},
		ExcludedSpecFields: []string{"details.weight"}, // Exclude a nested field
	}

	g := NewOASSchemaGenerator(mockDoc, generatorConfig, resourceConfig)

	// Act
	specBytes, warnings, err := g.BuildSpecSchema()

	// Assert
	require.NoError(t, err)
	require.Empty(t, warnings)
	require.NotNil(t, specBytes)

	var schemaMap map[string]interface{}
	err = json.Unmarshal(specBytes, &schemaMap)
	require.NoError(t, err, "Generated schema should be valid JSON")

	// 1. Check for base schema properties
	properties := schemaMap["properties"].(map[string]interface{})
	assert.Contains(t, properties, "name", "Should contain 'name' from base schema")
	assert.Contains(t, properties, "details", "Should contain 'details' object from base schema")

	// 2. Check for added parameters
	assert.Contains(t, properties, "verbose", "Should contain 'verbose' parameter from 'get' operation")
	verboseProp := properties["verbose"].(map[string]interface{})
	assert.Equal(t, "boolean", verboseProp["type"])
	assert.Contains(t, verboseProp["description"], "PARAMETER: query")

	// 3. Check for non-added identifiers (since by default they are not added to spec) and for field that is both identifier and part of base schema
	assert.NotContains(t, properties, "fieldOnlyInResponseAndShouldNotBeAdded", "Should NOT contain 'fieldOnlyInResponseAndShouldNotBeAdded' identifier")
	assert.Contains(t, properties, "name", "Should contain 'name' since it is an identifer but it is added since it is a part of the base schema")

	// 4. Check for added configurationRef
	assert.Contains(t, properties, "configurationRef", "Should contain 'configurationRef'")
	configRef := properties["configurationRef"].(map[string]interface{})
	assert.Equal(t, "object", configRef["type"])

	// 5. Check for removed configured fields
	assert.NotContains(t, properties, "X-Tenant-ID", "Should NOT contain configured field 'X-Tenant-ID'")

	// 6. Check for removed excluded fields
	detailsProp := properties["details"].(map[string]interface{})
	detailsSubProps := detailsProp["properties"].(map[string]interface{})
	assert.Contains(t, detailsSubProps, "color", "Details object should still contain 'color'")
	assert.NotContains(t, detailsSubProps, "weight", "Should NOT contain excluded field 'details.weight'")

	// 7. Check required fields
	required := schemaMap["required"].([]interface{})
	assert.ElementsMatch(t, []interface{}{"name", "configurationRef"}, required, "Required fields should be correct")

	// 8. Check that Authorization header was skipped
	assert.NotContains(t, properties, "Authorization", "Should NOT contain 'Authorization' header")
}
