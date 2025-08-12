package oas2jsonschema

import (
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
)

func TestSchemaConversion(t *testing.T) {
	testCases := []struct {
		name              string
		originalLibSchema *base.Schema
		assertDomain      func(t *testing.T, domainSchema *Schema)
		assertReconverted func(t *testing.T, reconvertedLibSchema *base.Schema)
	}{
		{
			name: "Simple Object",
			originalLibSchema: func() *base.Schema {
				s := &base.Schema{
					Type:        []string{"object"},
					Description: "A test schema",
					Properties:  orderedmap.New[string, *base.SchemaProxy](),
				}
				s.Properties.Set("id", base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}}))
				return s
			}(),
			assertDomain: func(t *testing.T, domainSchema *Schema) {
				assert.NotNil(t, domainSchema)
				assert.Equal(t, []string{"object"}, domainSchema.Type)
				assert.Equal(t, "A test schema", domainSchema.Description)
				assert.Len(t, domainSchema.Properties, 1)
				assert.Equal(t, "id", domainSchema.Properties[0].Name)
				assert.Equal(t, []string{"string"}, domainSchema.Properties[0].Schema.Type)
			},
			assertReconverted: func(t *testing.T, reconvertedLibSchema *base.Schema) {
				idProp, ok := reconvertedLibSchema.Properties.Get("id")
				assert.True(t, ok)
				idSchema, _ := idProp.BuildSchema()
				assert.Equal(t, []string{"string"}, idSchema.Type)
			},
		},
		{
			name: "Nested Object",
			originalLibSchema: func() *base.Schema {
				nested := &base.Schema{Type: []string{"object"}, Properties: orderedmap.New[string, *base.SchemaProxy]()}
				nested.Properties.Set("street", base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}}))
				s := &base.Schema{Type: []string{"object"}, Properties: orderedmap.New[string, *base.SchemaProxy]()}
				s.Properties.Set("address", base.CreateSchemaProxy(nested))
				return s
			}(),
			assertDomain: func(t *testing.T, domainSchema *Schema) {
				assert.Len(t, domainSchema.Properties, 1)
				addressProp := domainSchema.Properties[0]
				assert.Equal(t, "address", addressProp.Name)
				assert.Equal(t, []string{"object"}, addressProp.Schema.Type)
				assert.Len(t, addressProp.Schema.Properties, 1)
				streetProp := addressProp.Schema.Properties[0]
				assert.Equal(t, "street", streetProp.Name)
				assert.Equal(t, []string{"string"}, streetProp.Schema.Type)
			},
			assertReconverted: func(t *testing.T, reconvertedLibSchema *base.Schema) {
				addressProp, _ := reconvertedLibSchema.Properties.Get("address")
				addressSchema, _ := addressProp.BuildSchema()
				streetProp, _ := addressSchema.Properties.Get("street")
				streetSchema, _ := streetProp.BuildSchema()
				assert.Equal(t, []string{"string"}, streetSchema.Type)
			},
		},
		{
			name: "Array of Strings",
			originalLibSchema: &base.Schema{
				Type:  []string{"array"},
				Items: &base.DynamicValue[*base.SchemaProxy, bool]{A: base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}})},
			},
			assertDomain: func(t *testing.T, domainSchema *Schema) {
				assert.Equal(t, []string{"array"}, domainSchema.Type)
				assert.NotNil(t, domainSchema.Items)
				assert.Equal(t, []string{"string"}, domainSchema.Items.Type)
			},
			assertReconverted: func(t *testing.T, reconvertedLibSchema *base.Schema) {
				assert.NotNil(t, reconvertedLibSchema.Items)
				itemsSchema, _ := reconvertedLibSchema.Items.A.BuildSchema()
				assert.Equal(t, []string{"string"}, itemsSchema.Type)
			},
		},
		{
			name: "AllOf Composition",
			originalLibSchema: func() *base.Schema {
				part1 := &base.Schema{Properties: orderedmap.New[string, *base.SchemaProxy]()}
				part1.Properties.Set("id", base.CreateSchemaProxy(&base.Schema{Type: []string{"integer"}}))
				part2 := &base.Schema{Properties: orderedmap.New[string, *base.SchemaProxy]()}
				part2.Properties.Set("name", base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}}))
				return &base.Schema{
					AllOf: []*base.SchemaProxy{base.CreateSchemaProxy(part1), base.CreateSchemaProxy(part2)},
				}
			}(),
			assertDomain: func(t *testing.T, domainSchema *Schema) {
				assert.Len(t, domainSchema.AllOf, 2)
				assert.Len(t, domainSchema.AllOf[0].Properties, 1)
				assert.Equal(t, "id", domainSchema.AllOf[0].Properties[0].Name)
				assert.Len(t, domainSchema.AllOf[1].Properties, 1)
				assert.Equal(t, "name", domainSchema.AllOf[1].Properties[0].Name)
			},
			assertReconverted: func(t *testing.T, reconvertedLibSchema *base.Schema) {
				assert.Len(t, reconvertedLibSchema.AllOf, 2)
			},
		},
		{
			name:              "Nil Schema",
			originalLibSchema: nil,
			assertDomain: func(t *testing.T, domainSchema *Schema) {
				assert.Nil(t, domainSchema)
			},
			assertReconverted: func(t *testing.T, reconvertedLibSchema *base.Schema) {
				assert.Nil(t, reconvertedLibSchema)
			},
		},
		{
			name: "Complex AllOf Composition",
			originalLibSchema: func() *base.Schema {
				// Top-level schema with a property and an allOf
				schema := &base.Schema{
					Type:       []string{"object"},
					Properties: orderedmap.New[string, *base.SchemaProxy](),
				}
				schema.Properties.Set("id", base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}}))

				// First part of top-level allOf
				allOfPart1 := &base.Schema{Properties: orderedmap.New[string, *base.SchemaProxy]()}
				allOfPart1.Properties.Set("name", base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}}))

				// Second part of top-level allOf, which contains a nested allOf
				nestedAllOfPart1 := &base.Schema{Properties: orderedmap.New[string, *base.SchemaProxy]()}
				nestedAllOfPart1.Properties.Set("street", base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}}))

				nestedAllOfPart2 := &base.Schema{Properties: orderedmap.New[string, *base.SchemaProxy]()}
				nestedAllOfPart2.Properties.Set("city", base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}}))

				allOfPart2 := &base.Schema{
					AllOf: []*base.SchemaProxy{
						base.CreateSchemaProxy(nestedAllOfPart1),
						base.CreateSchemaProxy(nestedAllOfPart2),
					},
				}

				// Third part of top-level allOf, containing an array of items with allOf
				tagAllOfPart1 := &base.Schema{Properties: orderedmap.New[string, *base.SchemaProxy]()}
				tagAllOfPart1.Properties.Set("tag_id", base.CreateSchemaProxy(&base.Schema{Type: []string{"integer"}}))

				tagAllOfPart2 := &base.Schema{Properties: orderedmap.New[string, *base.SchemaProxy]()}
				tagAllOfPart2.Properties.Set("tag_name", base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}}))

				arrayItemsSchema := &base.Schema{
					AllOf: []*base.SchemaProxy{
						base.CreateSchemaProxy(tagAllOfPart1),
						base.CreateSchemaProxy(tagAllOfPart2),
					},
				}

				allOfPart3 := &base.Schema{Properties: orderedmap.New[string, *base.SchemaProxy]()}
				allOfPart3.Properties.Set("tags", base.CreateSchemaProxy(&base.Schema{
					Type:  []string{"array"},
					Items: &base.DynamicValue[*base.SchemaProxy, bool]{A: base.CreateSchemaProxy(arrayItemsSchema)},
				}))

				schema.AllOf = []*base.SchemaProxy{
					base.CreateSchemaProxy(allOfPart1),
					base.CreateSchemaProxy(allOfPart2),
					base.CreateSchemaProxy(allOfPart3),
				}
				return schema
			}(),
			assertDomain: func(t *testing.T, domainSchema *Schema) {
				assert.NotNil(t, domainSchema)
				// Check top-level property
				assert.Len(t, domainSchema.Properties, 1)
				assert.Equal(t, "id", domainSchema.Properties[0].Name)

				// Check top-level allOf
				assert.Len(t, domainSchema.AllOf, 3)

				// Check first allOf item
				allOf1 := domainSchema.AllOf[0]
				assert.Len(t, allOf1.Properties, 1)
				assert.Equal(t, "name", allOf1.Properties[0].Name)

				// Check second allOf item (with nested allOf)
				allOf2 := domainSchema.AllOf[1]
				assert.Len(t, allOf2.AllOf, 2)
				assert.Equal(t, "street", allOf2.AllOf[0].Properties[0].Name)
				assert.Equal(t, "city", allOf2.AllOf[1].Properties[0].Name)

				// Check third allOf item (with array of allOf items)
				allOf3 := domainSchema.AllOf[2]
				assert.Len(t, allOf3.Properties, 1)
				tagsProp := allOf3.Properties[0]
				assert.Equal(t, "tags", tagsProp.Name)
				assert.Equal(t, []string{"array"}, tagsProp.Schema.Type)
				assert.NotNil(t, tagsProp.Schema.Items)
				assert.Len(t, tagsProp.Schema.Items.AllOf, 2)
				assert.Equal(t, "tag_id", tagsProp.Schema.Items.AllOf[0].Properties[0].Name)
				assert.Equal(t, "tag_name", tagsProp.Schema.Items.AllOf[1].Properties[0].Name)
			},
			assertReconverted: func(t *testing.T, reconvertedLibSchema *base.Schema) {
				assert.NotNil(t, reconvertedLibSchema)
				assert.Len(t, reconvertedLibSchema.AllOf, 3)
				idProp, ok := reconvertedLibSchema.Properties.Get("id")
				assert.True(t, ok)
				idSchema, _ := idProp.BuildSchema()
				assert.Equal(t, []string{"string"}, idSchema.Type)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Convert to our domain model
			var proxy *base.SchemaProxy
			if tc.originalLibSchema != nil {
				proxy = base.CreateSchemaProxy(tc.originalLibSchema)
			}
			domainSchema := convertLibopenapiSchema(proxy)

			// 2. Assertions on the domain model
			tc.assertDomain(t, domainSchema)

			// 3. Convert back to libopenapi model
			reconvertedLibSchema := convertToLibopenapiSchema(domainSchema)

			// 4. Assertions on the reconverted model
			tc.assertReconverted(t, reconvertedLibSchema)
		})
	}
}

func TestLibOASDocumentAdapter(t *testing.T) {
	t.Run("FindPath should return correct PathItem", func(t *testing.T) {
		// Mock high-level libopenapi structures
		pathItems := orderedmap.New[string, *v3.PathItem]()
		pathItems.Set("/users", &v3.PathItem{})
		docModel := &v3.Document{
			Paths: &v3.Paths{PathItems: pathItems},
		}
		libDoc := &libopenapi.DocumentModel[v3.Document]{Model: *docModel}

		// Create adapter
		adapter := NewLibOASDocumentAdapter(libDoc)

		// Test FindPath
		pathItem, found := adapter.FindPath("/users")
		assert.True(t, found)
		assert.NotNil(t, pathItem)

		_, found = adapter.FindPath("/nonexistent")
		assert.False(t, found)
	})

	t.Run("SecuritySchemes should return correct SecuritySchemeInfo", func(t *testing.T) {
		// Mock high-level libopenapi structures
		securitySchemes := orderedmap.New[string, *v3.SecurityScheme]()
		securitySchemes.Set("BasicAuth", &v3.SecurityScheme{Type: "http", Scheme: "basic"})
		securitySchemes.Set("ApiKeyAuth", &v3.SecurityScheme{Type: "apiKey", In: "header", Name: "X-API-Key"})

		docModel := &v3.Document{
			Components: &v3.Components{SecuritySchemes: securitySchemes},
		}
		libDoc := &libopenapi.DocumentModel[v3.Document]{Model: *docModel}

		// Create adapter
		adapter := NewLibOASDocumentAdapter(libDoc)

		// Test SecuritySchemes
		schemes := adapter.SecuritySchemes()
		assert.Len(t, schemes, 2)

		// Note: Order is not guaranteed in maps, so we check for existence and correctness.
		foundBasic, foundAPIKey := false, false
		for _, s := range schemes {
			if s.Name == "BasicAuth" {
				assert.Equal(t, SchemeTypeHTTP, s.Type)
				assert.Equal(t, "basic", s.Scheme)
				foundBasic = true
			}
			if s.Name == "ApiKeyAuth" {
				assert.Equal(t, SchemeTypeAPIKey, s.Type)
				assert.Equal(t, "header", s.In)
				assert.Equal(t, "X-API-Key", s.ParamName)
				foundAPIKey = true
			}
		}
		assert.True(t, foundBasic, "BasicAuth scheme not found")
		assert.True(t, foundAPIKey, "ApiKeyAuth scheme not found")
	})
}

func TestLibOASPathItemAdapter(t *testing.T) {
	t.Run("GetOperations should return correct Operations", func(t *testing.T) {
		// Mock high-level libopenapi structures
		pathItem := &v3.PathItem{
			Get:  &v3.Operation{Summary: "Get User"},
			Post: &v3.Operation{Summary: "Create User"},
		}

		// Create adapter
		adapter := &libOASPathItemAdapter{path: pathItem}

		// Test GetOperations
		ops := adapter.GetOperations()
		assert.Len(t, ops, 2)

		getOp, ok := ops["get"]
		assert.True(t, ok)
		assert.NotNil(t, getOp)

		postOp, ok := ops["post"]
		assert.True(t, ok)
		assert.NotNil(t, postOp)
	})
}

func TestLibOASOperationAdapter(t *testing.T) {
	t.Run("GetParameters should return correct ParameterInfo", func(t *testing.T) {
		// Mock high-level libopenapi structures
		params := []*v3.Parameter{
			{Name: "id", In: "path", Description: "User ID", Schema: base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}})},
			{Name: "token", In: "query", Description: "Auth Token", Schema: base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}})},
		}
		op := &v3.Operation{Parameters: params}

		// Create adapter
		adapter := &libOASOperationAdapter{op: op}

		// Test GetParameters
		paramInfos := adapter.GetParameters()
		assert.Len(t, paramInfos, 2)

		assert.Equal(t, "id", paramInfos[0].Name)
		assert.Equal(t, "path", paramInfos[0].In)
		assert.Equal(t, []string{"string"}, paramInfos[0].Schema.Type)

		assert.Equal(t, "token", paramInfos[1].Name)
		assert.Equal(t, "query", paramInfos[1].In)
		assert.Equal(t, []string{"string"}, paramInfos[1].Schema.Type)
	})

	t.Run("GetRequestBody should return correct RequestBodyInfo", func(t *testing.T) {
		// Mock high-level libopenapi structures
		content := orderedmap.New[string, *v3.MediaType]()
		content.Set("application/json", &v3.MediaType{Schema: base.CreateSchemaProxy(&base.Schema{Type: []string{"object"}})})
		rb := &v3.RequestBody{Content: content}
		op := &v3.Operation{RequestBody: rb}

		// Create adapter
		adapter := &libOASOperationAdapter{op: op}

		// Test GetRequestBody
		requestBodyInfo := adapter.GetRequestBody()
		assert.NotNil(t, requestBodyInfo.Content)
		jsonSchema, ok := requestBodyInfo.Content["application/json"]
		assert.True(t, ok)
		assert.Equal(t, []string{"object"}, jsonSchema.Type)
	})

	t.Run("GetResponses should return correct ResponseInfo", func(t *testing.T) {
		// Mock high-level libopenapi structures
		codes := orderedmap.New[string, *v3.Response]()
		jsonContent := orderedmap.New[string, *v3.MediaType]()
		jsonContent.Set("application/json", &v3.MediaType{Schema: base.CreateSchemaProxy(&base.Schema{Type: []string{"object"}})})
		codes.Set("200", &v3.Response{Content: jsonContent})

		op := &v3.Operation{Responses: &v3.Responses{Codes: codes}}

		// Create adapter
		adapter := &libOASOperationAdapter{op: op}

		// Test GetResponses
		responseInfos := adapter.GetResponses()
		assert.Len(t, responseInfos, 1)

		resp200, ok := responseInfos[200]
		assert.True(t, ok)
		assert.NotNil(t, resp200.Content)
		jsonSchema, ok := resp200.Content["application/json"]
		assert.True(t, ok)
		assert.Equal(t, []string{"object"}, jsonSchema.Type)
	})
}

func TestAdapterEdgeCases(t *testing.T) {
	t.Run("SecuritySchemes returns nil if components are nil", func(t *testing.T) {
		docModel := &v3.Document{Components: nil}
		libDoc := &libopenapi.DocumentModel[v3.Document]{Model: *docModel}
		adapter := NewLibOASDocumentAdapter(libDoc)
		assert.Nil(t, adapter.SecuritySchemes())
	})

	t.Run("SecuritySchemes returns nil if security schemes are nil", func(t *testing.T) {
		docModel := &v3.Document{
			Components: &v3.Components{SecuritySchemes: nil},
		}
		libDoc := &libopenapi.DocumentModel[v3.Document]{Model: *docModel}
		adapter := NewLibOASDocumentAdapter(libDoc)
		assert.Nil(t, adapter.SecuritySchemes())
	})

	t.Run("GetRequestBody returns empty if request body is nil", func(t *testing.T) {
		op := &v3.Operation{RequestBody: nil}
		adapter := &libOASOperationAdapter{op: op}
		body := adapter.GetRequestBody()
		assert.Empty(t, body.Content)
	})

	t.Run("GetRequestBody returns empty if content is nil", func(t *testing.T) {
		op := &v3.Operation{RequestBody: &v3.RequestBody{Content: nil}}
		adapter := &libOASOperationAdapter{op: op}
		body := adapter.GetRequestBody()
		assert.Empty(t, body.Content)
	})

	t.Run("GetResponses returns nil if responses are nil", func(t *testing.T) {
		op := &v3.Operation{Responses: nil}
		adapter := &libOASOperationAdapter{op: op}
		responses := adapter.GetResponses()
		assert.Nil(t, responses)
	})

	t.Run("GetResponses skips non-integer response codes", func(t *testing.T) {
		codes := orderedmap.New[string, *v3.Response]()
		codes.Set("200", &v3.Response{})
		codes.Set("default", &v3.Response{}) // Invalid for Atoi, should be skipped
		op := &v3.Operation{Responses: &v3.Responses{Codes: codes}}
		adapter := &libOASOperationAdapter{op: op}
		responses := adapter.GetResponses()
		assert.Len(t, responses, 1)
		_, ok := responses[200]
		assert.True(t, ok)
	})

	t.Run("convertLibopenapiSchema returns nil on schema build panic", func(t *testing.T) {
		// This test confirms that our adapter's defer/recover logic correctly
		// handles a panic from the underlying library.

		// 1. Create a schema proxy that is known to cause a panic when BuildSchema is called
		// on it without a valid document context.
		proxy := base.CreateSchemaProxyRef("#/components/schemas/NonExistent")

		// 2. Call the conversion function. The defer/recover should catch the panic.
		schema := convertLibopenapiSchema(proxy)

		// 3. Assert that the function returned nil instead of crashing.
		assert.Nil(t, schema)
	})
}
