package oas2jsonschema

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	_ "embed"

	"github.com/krateoplatformops/crdgen"
	definitionv1alpha1 "github.com/krateoplatformops/oasgen-provider/apis/restdefinitions/v1alpha1"
	"github.com/krateoplatformops/oasgen-provider/internal/tools/crd"
	"github.com/pb33f/libopenapi"
)

//go:embed tests/oas/*
var content embed.FS

func TestGenerateByteSchemas(t *testing.T) {
	// TestGenerateJsonSchema tests the generation of a JSON schema
	// from a YAML schema.
	// It does this by generating a JSON schema from the YAML schema
	// and then comparing the generated JSON schema with the expected
	// JSON schema.
	// The expected JSON schema is created by parsing a YAML file
	// that contains the expected JSON schema.
	// The expected JSON schema is then compared with the generated
	// JSON schema.

	ctx := context.Background()

	tempdir := os.TempDir()
	basePath := path.Join(tempdir, "oasgen-provider-test")

	os.MkdirAll(basePath, 0755)

	entries, err := fs.ReadDir(content, "tests/oas")
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		schemaRef, err := content.ReadFile(path.Join("tests/oas", entry.Name()))
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		contents := schemaRef //os.ReadFile(path.Join(basePath, path.Base(schemaRef)))
		d, err := libopenapi.NewDocument(contents)
		if err != nil {
			t.Errorf("failed to create document: %f", err)
		}

		doc, modelErrors := d.BuildV3Model()
		if len(modelErrors) > 0 {
			t.Errorf("failed to build model: %f", errors.Join(modelErrors...))
		}
		if doc == nil {
			t.Errorf("failed to build model")
		}

		if doc == nil {
			t.Errorf("failed to build model")
			return
		}
		if doc.Index == nil {
			t.Errorf("failed to build model index")
			return
		}
		// Resolve model references
		resolvingErrors := doc.Index.GetResolver().Resolve()
		errs := []error{}
		for i := range resolvingErrors {
			t.Log("Resolving error", "error", resolvingErrors[i].Error())
			errs = append(errs, resolvingErrors[i].ErrorRef)
		}
		if len(resolvingErrors) > 0 {
			t.Errorf("failed to resolve model references: %f", errors.Join(errs...))
		}

		// Get the first path item
		resource := definitionv1alpha1.Resource{
			Kind: "test-pet",
			VerbsDescription: []definitionv1alpha1.VerbsDescription{
				{
					Action: "create",
					Path:   "/pet",
					Method: "POST",
				},
				{
					Action: "get",
					Path:   "/pet/{petId}",
					Method: "GET",
				},
				{
					Action: "update",
					Path:   "/pet",
					Method: "PUT",
				},
				{
					Action: "delete",
					Path:   "/pet/{petId}",
					Method: "DELETE",
				},
			},
			AdditionalStatusFields: []string{"category", "status"},
		}

		identifiers := []string{"id"}

		gen, errors, fatalError := GenerateByteSchemas(doc, resource, identifiers)
		if fatalError != nil {
			t.Errorf("fatal error: %v", fatalError)
		}
		if len(errors) > 0 {
			break
		}

		gvk := schema.GroupVersionKind{
			Group:   "petstore.swagger.io",
			Version: "v1alpha1",
			Kind:    "Pet",
		}

		resourceResult := crdgen.Generate(ctx, crdgen.Options{
			Managed:                true,
			WorkDir:                "oasgen-provider-test",
			GVK:                    gvk,
			Categories:             []string{strings.ToLower(gvk.Kind)},
			SpecJsonSchemaGetter:   gen.OASSpecJsonSchemaGetter(),
			StatusJsonSchemaGetter: gen.OASStatusJsonSchemaGetter(),
		})
		if resourceResult.Err != nil {
			t.Errorf("error: %v", resourceResult.Err)
		}

		_, err = crd.Unmarshal(resourceResult.Manifest)
		if err != nil {
			t.Errorf("unmarshalling CRD: %f", err)
		}

		t.Log("CRD generated successfully")
	}

}

func TestGenerateByteSchemasAdditionalStatusFields(t *testing.T) {
	t.Run("AdditionalStatusFields are correctly included", func(t *testing.T) {
		doc, err := libopenapi.NewDocument([]byte(`
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  securitySchemes: {}
`))
		if err != nil {
			t.Fatalf("failed to create document: %v", err)
		}
		model, _ := doc.BuildV3Model()

		resource := definitionv1alpha1.Resource{
			Kind:                   "TestResource",
			AdditionalStatusFields: []string{"field1", "field2", "anotherField"},
		}
		identifiers := []string{"id"}

		gen, _, fatalError := GenerateByteSchemas(model, resource, identifiers)
		if fatalError != nil {
			t.Fatalf("fatal error: %v", fatalError)
		}

		statusSchemaBytes, err := gen.OASStatusJsonSchemaGetter().Get()
		if err != nil {
			t.Fatalf("failed to get status schema bytes: %v", err)
		}
		if statusSchemaBytes == nil {
			t.Fatal("status schema bytes are nil")
		}

		var statusSchema map[string]interface{}
		err = json.Unmarshal(statusSchemaBytes, &statusSchema)
		if err != nil {
			t.Fatalf("failed to unmarshal status schema: %v", err)
		}

		properties, ok := statusSchema["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("status schema does not contain properties map")
		}

		expectedFields := append(resource.AdditionalStatusFields, identifiers...)
		for _, field := range expectedFields {
			prop, ok := properties[field].(map[string]interface{})
			if !ok {
				t.Errorf("expected field '%s' not found in status properties", field)
				continue
			}
			fieldType, ok := prop["type"].(string)
			if !ok || fieldType != "string" {
				t.Errorf("expected field '%s' to be of type 'string', got '%v'", field, prop["type"])
			}
		}
	})

	t.Run("Empty AdditionalStatusFields array", func(t *testing.T) {
		doc, err := libopenapi.NewDocument([]byte(`
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  securitySchemes: {}
 `))

		if err != nil {
			t.Fatalf("failed to create document: %v", err)
		}
		model, _ := doc.BuildV3Model()

		resource := definitionv1alpha1.Resource{
			Kind:                   "TestResource",
			AdditionalStatusFields: []string{},
		}
		identifiers := []string{"id", "name"}

		gen, _, fatalError := GenerateByteSchemas(model, resource, identifiers)
		if fatalError != nil {
			t.Fatalf("fatal error: %v", fatalError)
		}

		statusSchemaBytes, err := gen.OASStatusJsonSchemaGetter().Get()
		if err != nil {
			t.Fatalf("failed to get status schema bytes: %v", err)
		}
		if statusSchemaBytes == nil {
			t.Fatal("status schema bytes are nil")
		}

		var statusSchema map[string]interface{}
		err = json.Unmarshal(statusSchemaBytes, &statusSchema)
		if err != nil {
			t.Fatalf("failed to unmarshal status schema: %v", err)
		}

		properties, ok := statusSchema["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("status schema does not contain properties map")
		}

		if len(properties) != len(identifiers) {
			// since AdditionalStatusFields is empty, we expect only identifiers
			t.Errorf("expected %d properties, got %d", len(identifiers), len(properties))
		}

		for _, field := range identifiers {
			prop, ok := properties[field].(map[string]interface{})
			if !ok {
				t.Errorf("expected identifier '%s' not found in status properties", field)
				continue
			}
			fieldType, ok := prop["type"].(string)
			if !ok || fieldType != "string" {
				t.Errorf("expected identifier '%s' to be of type 'string', got '%v'", field, prop["type"])
			}
		}
	})

	t.Run("AdditionalStatusFields with duplicate names", func(t *testing.T) {
		doc, err := libopenapi.NewDocument([]byte(`
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  securitySchemes: {}
 `))

		if err != nil {
			t.Fatalf("failed to create document: %v", err)
		}
		model, _ := doc.BuildV3Model()

		resource := definitionv1alpha1.Resource{
			Kind:                   "TestResource",
			AdditionalStatusFields: []string{"status", "category", "status", "anotherField"},
		}
		identifiers := []string{"id"}

		gen, _, fatalError := GenerateByteSchemas(model, resource, identifiers)
		if fatalError != nil {
			t.Fatalf("fatal error: %v", fatalError)
		}

		statusSchemaBytes, err := gen.OASStatusJsonSchemaGetter().Get()
		if err != nil {
			t.Fatalf("failed to get status schema bytes: %v", err)
		}
		if statusSchemaBytes == nil {
			t.Fatal("status schema bytes are nil")
		}

		var statusSchema map[string]interface{}
		err = json.Unmarshal(statusSchemaBytes, &statusSchema)
		if err != nil {
			t.Fatalf("failed to unmarshal status schema: %v", err)
		}

		properties, ok := statusSchema["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("status schema does not contain properties map")
		}

		// Expect unique fields: "id", "status", "category", "anotherField"
		// and not duplicates
		expectedUniqueFields := map[string]bool{
			"id":           true,
			"status":       true,
			"category":     true,
			"anotherField": true,
		}

		if len(properties) != len(expectedUniqueFields) {
			t.Errorf("expected %d unique properties, got %d", len(expectedUniqueFields), len(properties))
		}

		for field := range expectedUniqueFields {
			prop, ok := properties[field].(map[string]interface{})
			if !ok {
				t.Errorf("expected field '%s' not found in status properties", field)
				continue
			}
			fieldType, ok := prop["type"].(string)
			if !ok || fieldType != "string" {
				t.Errorf("expected field '%s' to be of type 'string', got '%v'", field, prop["type"])
			}
		}
	})
}

func TestGetPrimaryType(t *testing.T) {
	tests := []struct {
		name     string
		types    []string
		expected string
	}{
		{
			name:     "Primary type first",
			types:    []string{"string", "null"},
			expected: "string",
		},
		{
			name:     "Null first, then primary type",
			types:    []string{"null", "integer"},
			expected: "integer",
		},
		{
			name:     "Only null",
			types:    []string{"null"},
			expected: "",
		},
		{
			name:     "Empty slice",
			types:    []string{},
			expected: "",
		},
		{
			name:     "Multiple primary types (first one wins)",
			types:    []string{"string", "integer"},
			expected: "string",
		},
		{
			name:     "Multiple types with null",
			types:    []string{"null", "string", "integer"},
			expected: "string",
		},
		{
			name:     "Multiple nulls",
			types:    []string{"null", "null"},
			expected: "",
		},
		{
			name:     "Array type",
			types:    []string{"array", "null"},
			expected: "array",
		},
		{
			name:     "Object type",
			types:    []string{"object", "null"},
			expected: "object",
		},
		{
			name:     "Multiple types with object and null",
			types:    []string{"object", "null", "string"},
			expected: "object",
		},
		{
			name:     "Multiple types with array and null",
			types:    []string{"array", "null", "integer"},
			expected: "array",
		},
		{
			name:     "Multiple types with mixed nulls",
			types:    []string{"null", "string", "null", "integer"},
			expected: "string",
		},
		{
			name:     "Multiple types with mixed nulls and objects",
			types:    []string{"null", "object", "null", "string"},
			expected: "object",
		},
		{
			name:     "Multiple types with mixed nulls and arrays",
			types:    []string{"null", "array", "null", "integer"},
			expected: "array",
		},
		{
			name:     "Multiple types with mixed nulls and objects and arrays",
			types:    []string{"null", "object", "array", "null", "string"},
			expected: "object",
		},
		{
			name:     "Array and object only",
			types:    []string{"array", "object"},
			expected: "array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPrimaryType(tt.types)
			if result != tt.expected {
				t.Errorf("getPrimaryType(%v) = %s; want %s", tt.types, result, tt.expected)
			}
		})
	}
}

func TestExtractSchemaForAction(t *testing.T) {
	tests := []struct {
		name                     string
		oasContent               string
		verbs                    []definitionv1alpha1.VerbsDescription
		targetAction             string
		expectedErr              bool
		expectedSchemaProperties []string
	}{
		{
			name: "Successful GET schema extraction",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items/{id}:
    get:
      responses:
        '200':
          description: A single item
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  name:
                    type: string
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items/{id}", Method: "GET"},
			},
			targetAction:             "get",
			expectedErr:              false,
			expectedSchemaProperties: []string{"id", "name"},
		},
		{
			name: "Successful FINDBY schema extraction (array items)",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: List of items
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  properties:
                    id:
                      type: string
                    value:
                      type: integer
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "findby", Path: "/items", Method: "GET"},
			},
			targetAction:             "findby",
			expectedErr:              false,
			expectedSchemaProperties: []string{"id", "value"},
		},
		{
			name: "Action not found",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  securitySchemes: {}
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "create", Path: "/items", Method: "POST"},
			},
			targetAction:             "get",
			expectedErr:              false, // Should return nil schema, nil error
			expectedSchemaProperties: nil,
		},
		{
			name: "No 200/201 response (only 404)",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '404':
          description: Not found
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
			},
			targetAction:             "get",
			expectedErr:              false, // Should return nil schema, nil error
			expectedSchemaProperties: nil,
		},
		{
			name: "Missing schema in response content",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json: {}
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
			},
			targetAction:             "get",
			expectedErr:              false, // Should return nil schema, nil error
			expectedSchemaProperties: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := libopenapi.NewDocument([]byte(tt.oasContent))
			if err != nil {
				t.Fatalf("failed to create document: %v", err)
			}
			model, modelErrors := doc.BuildV3Model()
			if len(modelErrors) > 0 {
				t.Fatalf("failed to build model: %v", errors.Join(modelErrors...))
			}

			schema, err := extractSchemaForAction(model, tt.verbs, tt.targetAction)

			if tt.expectedErr {
				if err == nil {
					t.Errorf("expected an error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("did not expect an error but got: %v", err)
				}
				if tt.expectedSchemaProperties == nil {
					if schema != nil {
						t.Errorf("expected nil schema but got non-nil")
					}
				} else {
					if schema == nil {
						t.Fatalf("expected non-nil schema but got nil")
					}
					if schema.Properties == nil && len(tt.expectedSchemaProperties) > 0 {
						t.Errorf("expected properties but schema.Properties is nil")
					}
					for _, prop := range tt.expectedSchemaProperties {
						if _, ok := schema.Properties.Get(prop); !ok {
							t.Errorf("expected property '%s' not found in schema", prop)
						}
					}
					if schema.Properties.Len() != len(tt.expectedSchemaProperties) {
						t.Errorf("expected %d properties, got %d", len(tt.expectedSchemaProperties), schema.Properties.Len())
					}
				}
			}
		})
	}
}

func TestValidateSchemas(t *testing.T) {
	tests := []struct {
		name             string
		oasContent       string
		verbs            []definitionv1alpha1.VerbsDescription
		expectedWarnings int
	}{
		{
			name: "No get or findby action",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
components:
  securitySchemes: {}
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "create", Path: "/items", Method: "POST"},
			},
			expectedWarnings: 1,
		},
		{
			name: "Get action with compatible create",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  name:
                    type: string
    post:
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  name:
                    type: string
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
				{Action: "create", Path: "/items", Method: "POST"},
			},
			expectedWarnings: 0,
		},
		{
			name: "Get action with incompatible create (type mismatch)",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  name:
                    type: string
    post:
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  name:
                    type: integer
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
				{Action: "create", Path: "/items", Method: "POST"},
			},
			expectedWarnings: 1,
		},
		{
			name: "Get action with compatible findby",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  name:
                    type: string
  /items/find:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  properties:
                    id:
                      type: string
                    name:
                      type: string
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
				{Action: "findby", Path: "/items/find", Method: "GET"},
			},
			expectedWarnings: 0,
		},
		{
			name: "Get action with incompatible findby (type mismatch)",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  name:
                    type: string
  /items/find:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  properties:
                    id:
                      type: string
                    name:
                      type: integer
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
				{Action: "findby", Path: "/items/find", Method: "GET"},
			},
			expectedWarnings: 1,
		},
		{
			name: "Incompatible nested object",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: object
                    properties:
                      value:
                        type: string
    post:
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: object
                    properties:
                      value:
                        type: integer
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
				{Action: "create", Path: "/items", Method: "POST"},
			},
			expectedWarnings: 1,
		},
		{
			name: "Incompatible array items",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  tags:
                    type: array
                    items:
                      type: string
    post:
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                type: object
                properties:
                  tags:
                    type: array
                    items:
                      type: integer
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
				{Action: "create", Path: "/items", Method: "POST"},
			},
			expectedWarnings: 1,
		},
		{
			name: "Compatible get and update",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  name:
                    type: string
    put:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  name:
                    type: string
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
				{Action: "update", Path: "/items", Method: "PUT"},
			},
			expectedWarnings: 0,
		},
		{
			name: "Incompatible get and update",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  name:
                    type: string
    put:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  name:
                    type: boolean
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
				{Action: "update", Path: "/items", Method: "PUT"},
			},
			expectedWarnings: 1,
		},
		{
			name: "Compatible with nullability mismatch",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  name:
                    type:
                      - string
                      - "null"
    post:
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                type: object
                properties:
                  name:
                    type: string
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
				{Action: "create", Path: "/items", Method: "POST"},
			},
			expectedWarnings: 0,
		},
		{
			name: "Schema1 is nil in compareSchemas",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
    post:
      responses:
        '404':
          description: Not Found
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
				{Action: "create", Path: "/items", Method: "POST"}, // This will result in schema1 being nil
			},
			expectedWarnings: 1,
		},
		{
			name: "Schema2 is nil in compareSchemas",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '404':
          description: Not Found
    post:
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"}, // This will result in schema2 being nil
				{Action: "create", Path: "/items", Method: "POST"},
			},
			expectedWarnings: 1,
		},
		{
			name: "Schema1 has properties, Schema2 does not",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: object
                    properties:
                      field1:
                        type: string
    post:
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: string
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
				{Action: "create", Path: "/items", Method: "POST"},
			},
			expectedWarnings: 1,
		},
		{
			name: "Schema2 has properties, Schema1 does not",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: string
    post:
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: object
                    properties:
                      field1:
                        type: string
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
				{Action: "create", Path: "/items", Method: "POST"},
			},
			expectedWarnings: 1,
		},
		{
			name: "Array item schema is nil for first schema",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  tags:
                    type: array
                    items: # Missing schema for items
    post:
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                type: object
                properties:
                  tags:
                    type: array
                    items:
                      type: string
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
				{Action: "create", Path: "/items", Method: "POST"},
			},
			expectedWarnings: 1,
		},
		{
			name: "Array item schema is nil for second schema",
			oasContent: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  tags:
                    type: array
                    items:
                      type: string
    post:
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                type: object
                properties:
                  tags:
                    type: array
                    items: # Missing schema for items
`,
			verbs: []definitionv1alpha1.VerbsDescription{
				{Action: "get", Path: "/items", Method: "GET"},
				{Action: "create", Path: "/items", Method: "POST"},
			},
			expectedWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := libopenapi.NewDocument([]byte(tt.oasContent))
			if err != nil {
				t.Fatalf("failed to create document: %v", err)
			}
			model, modelErrors := doc.BuildV3Model()
			if len(modelErrors) > 0 {
				t.Fatalf("failed to build model: %v", errors.Join(modelErrors...))
			}

			warnings := validateSchemas(model, tt.verbs)
			if len(warnings) != tt.expectedWarnings {
				t.Errorf("expected %d warnings, got %d: %v", tt.expectedWarnings, len(warnings), warnings)
			}

			// print all warnings to help debugging
			t.Logf("All warnings for: %s, with length %d", tt.name, len(warnings))
			for _, warning := range warnings {
				t.Logf("Warning: %s", warning)
			}
		})
	}
}

func TestAreTypesCompatible(t *testing.T) {
	tests := []struct {
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
			name:     "Compatible with null (string vs string,null)",
			types1:   []string{"string"},
			types2:   []string{"string", "null"},
			expected: true,
		},
		{
			name:     "Compatible with null (string,null vs string)",
			types1:   []string{"string", "null"},
			types2:   []string{"string"},
			expected: true,
		},
		{
			name:     "Incompatible primary types",
			types1:   []string{"string"},
			types2:   []string{"integer"},
			expected: false,
		},
		{
			name:     "One has primary, other only null (compatible if primary allows null)",
			types1:   []string{"string", "null"},
			types2:   []string{"null"},
			expected: true,
		},
		{
			name:     "One has primary, other only null (incompatible if primary doesn't allow null)",
			types1:   []string{"string"},
			types2:   []string{"null"},
			expected: false,
		},
		{
			name:     "Both only null",
			types1:   []string{"null"},
			types2:   []string{"null"},
			expected: true,
		},
		{
			name:     "One empty, one primary",
			types1:   []string{},
			types2:   []string{"string"},
			expected: false, // An empty type implies "any", but for strict comparison, it's not directly compatible unless the other explicitly allows it.
		},
		{
			name:     "Both empty",
			types1:   []string{},
			types2:   []string{},
			expected: true, // Both "any"
		},
		{
			name:     "String vs Integer,null",
			types1:   []string{"string"},
			types2:   []string{"integer", "null"},
			expected: false,
		},
		{
			name:     "Integer,null vs String",
			types1:   []string{"integer", "null"},
			types2:   []string{"string"},
			expected: false,
		},
		{
			name:     "Object vs Object,null",
			types1:   []string{"object"},
			types2:   []string{"object", "null"},
			expected: true,
		},
		{
			name:     "Array vs Array,null",
			types1:   []string{"array"},
			types2:   []string{"array", "null"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := areTypesCompatible(tt.types1, tt.types2)
			if result != tt.expected {
				t.Errorf("areTypesCompatible(%v, %v) = %t; want %t", tt.types1, tt.types2, result, tt.expected)
			}
		})
	}
}

func TestGenerateByteSchemas_ErrorScenarios(t *testing.T) {
	oasContent := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pet:
    post:
      summary: Add a new pet
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
components:
  securitySchemes: {}
`
	doc, _ := libopenapi.NewDocument([]byte(oasContent))
	model, _ := doc.BuildV3Model()

	t.Run("Path not found in OAS", func(t *testing.T) {
		resource := definitionv1alpha1.Resource{
			Kind: "test-pet",
			VerbsDescription: []definitionv1alpha1.VerbsDescription{
				{
					Action: "create",
					Path:   "/non-existent-path", // This path does not exist
					Method: "POST",
				},
			},
		}

		_, _, fatalError := GenerateByteSchemas(model, resource, nil)
		if fatalError == nil {
			t.Fatal("expected a fatal error but got none")
		}
		if !strings.Contains(fatalError.Error(), "path /non-existent-path not found") {
			t.Errorf("expected error message to contain 'path /non-existent-path not found', but got: %v", fatalError)
		}
	})

	t.Run("Method not found for path", func(t *testing.T) {
		resource := definitionv1alpha1.Resource{
			Kind: "test-pet",
			VerbsDescription: []definitionv1alpha1.VerbsDescription{
				{
					Action: "create",
					Path:   "/pet",
					Method: "PUT", // This method does not exist for this path
				},
			},
		}

		_, _, fatalError := GenerateByteSchemas(model, resource, nil)
		if fatalError == nil {
			t.Fatal("expected a fatal error but got none")
		}
		if !strings.Contains(fatalError.Error(), "operation not found for PUT on path /pet") {
			t.Errorf("expected error message to contain 'operation not found for PUT on path /pet', but got: %v", fatalError)
		}
	})

	t.Run("Path in RestDefinition not found in OAS", func(t *testing.T) {
		resource := definitionv1alpha1.Resource{
			Kind: "test-pet",
			VerbsDescription: []definitionv1alpha1.VerbsDescription{
				{
					Action: "create",
					Path:   "/pet",
					Method: "POST",
				},
				{
					Action: "delete",
					Path:   "/non-existent-path", // This path does not exist in the OAS
					Method: "DELETE",
				},
			},
		}

		_, _, fatalError := GenerateByteSchemas(model, resource, nil)
		// we expect a fatal error here because the DELETE operation is not defined in the OAS
		if fatalError == nil {
			t.Fatal("expected a fatal error but got none")
		}
		if !strings.Contains(fatalError.Error(), "path /non-existent-path set in RestDefinition not found in OpenAPI spec") {
			t.Errorf("expected error message to contain 'path /non-existent-path set in RestDefinition not found in OpenAPI spec', but got: %v", fatalError)
		}

	})
}

func TestGenerateByteSchemas_Parameters(t *testing.T) {
	oasContent := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pet/{petId}:
    post:
      summary: Update a pet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: string
        - name: api_key
          in: header
          schema:
            type: string
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
components:
  securitySchemes: {}
`
	doc, _ := libopenapi.NewDocument([]byte(oasContent))
	model, _ := doc.BuildV3Model()

	resource := definitionv1alpha1.Resource{
		Kind: "test-pet",
		VerbsDescription: []definitionv1alpha1.VerbsDescription{
			{
				Action: "create",
				Path:   "/pet/{petId}",
				Method: "POST",
			},
		},
	}

	gen, _, fatalError := GenerateByteSchemas(model, resource, nil)
	if fatalError != nil {
		t.Fatalf("fatal error: %v", fatalError)
	}

	specSchemaBytes, err := gen.OASSpecJsonSchemaGetter().Get()
	if err != nil {
		t.Fatalf("failed to get spec schema bytes: %v", err)
	}

	var specSchema map[string]interface{}
	err = json.Unmarshal(specSchemaBytes, &specSchema)
	if err != nil {
		t.Fatalf("failed to unmarshal spec schema: %v", err)
	}

	properties, ok := specSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("spec schema does not contain properties map")
	}

	// Check for request body property
	if _, ok := properties["name"]; !ok {
		t.Error("expected field 'name' from request body not found")
	}

	// Check for path parameter
	petIdProp, ok := properties["petId"].(map[string]interface{})
	if !ok {
		t.Error("expected path parameter 'petId' not found")
	} else {
		if petIdProp["description"] != "PARAMETER: path, VERB: Post - " {
			t.Errorf("incorrect description for petId: got '%s'", petIdProp["description"])
		}
	}

	// Check for header parameter
	apiKeyProp, ok := properties["api_key"].(map[string]interface{})
	if !ok {
		t.Error("expected header parameter 'api_key' not found")
	} else {
		if apiKeyProp["description"] != "PARAMETER: header, VERB: Post - " {
			t.Errorf("incorrect description for api_key: got '%s'", apiKeyProp["description"])
		}
	}
}

func TestGenerateByteSchemas_CreateWithoutRequestBody(t *testing.T) {
	oasContent := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pet:
    post:
      summary: Add a new pet
  /pet/{petId}:
    delete:
      summary: Deletes a pet
      parameters:
        - name: api_key
          in: header
          schema:
            type: string
        - name: petId
          in: path
          required: true
          schema:
            type: string
components:
  securitySchemes:
    basicAuth:
      type: http
      scheme: basic
`
	doc, _ := libopenapi.NewDocument([]byte(oasContent))
	model, _ := doc.BuildV3Model()

	resource := definitionv1alpha1.Resource{
		Kind: "test-pet",
		VerbsDescription: []definitionv1alpha1.VerbsDescription{
			{
				Action: "create",
				Path:   "/pet",
				Method: "POST",
			},
			{
				Action: "delete",
				Path:   "/pet/{petId}",
				Method: "DELETE",
			},
		},
	}

	gen, _, fatalError := GenerateByteSchemas(model, resource, nil)
	if fatalError != nil {
		t.Fatalf("fatal error: %v", fatalError)
	}

	specSchemaBytes, err := gen.OASSpecJsonSchemaGetter().Get()
	if err != nil {
		t.Fatalf("failed to get spec schema bytes: %v", err)
	}

	var specSchema map[string]interface{}
	err = json.Unmarshal(specSchemaBytes, &specSchema)
	if err != nil {
		t.Fatalf("failed to unmarshal spec schema: %v", err)
	}

	properties, ok := specSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("spec schema does not contain properties map")
	}

	// Check for authenticationRefs
	authRefs, ok := properties["authenticationRefs"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'authenticationRefs' property")
	}

	authProps, ok := authRefs["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("authenticationRefs does not have properties")
	}

	if _, ok := authProps["basicAuthRef"]; !ok {
		t.Error("expected 'basicAuthRef' in authenticationRefs")
	}

	// Check for parameters from the delete verb
	if _, ok := properties["petId"]; !ok {
		t.Error("expected path parameter 'petId' not found")
	}
	if _, ok := properties["api_key"]; !ok {
		t.Error("expected header parameter 'api_key' not found")
	}
}

func TestGenerateByteSchemas_TopLevelArray(t *testing.T) {
	oasContent := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    post:
      summary: Add multiple pets
      requestBody:
        content:
          application/json:
            schema:
              type: array
              items:
                type: object
                properties:
                  name:
                    type: string
components:
  securitySchemes: {}
`
	doc, _ := libopenapi.NewDocument([]byte(oasContent))
	model, _ := doc.BuildV3Model()

	resource := definitionv1alpha1.Resource{
		Kind: "test-pet",
		VerbsDescription: []definitionv1alpha1.VerbsDescription{
			{
				Action: "create",
				Path:   "/pets",
				Method: "POST",
			},
		},
	}

	gen, _, fatalError := GenerateByteSchemas(model, resource, nil)
	if fatalError != nil {
		t.Fatalf("fatal error: %v", fatalError)
	}

	specSchemaBytes, err := gen.OASSpecJsonSchemaGetter().Get()
	if err != nil {
		t.Fatalf("failed to get spec schema bytes: %v", err)
	}

	var specSchema map[string]interface{}
	err = json.Unmarshal(specSchemaBytes, &specSchema)
	if err != nil {
		t.Fatalf("failed to unmarshal spec schema: %v", err)
	}

	if specSchema["type"] != "object" {
		t.Errorf("expected root type to be 'object', got '%s'", specSchema["type"])
	}

	properties, ok := specSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("spec schema does not contain properties map")
	}

	itemsProp, ok := properties["items"].(map[string]interface{})
	if !ok {
		t.Fatal("expected to find 'items' property")
	}

	if itemsProp["type"] != "array" {
		t.Errorf("expected items property to be of type 'array', got '%s'", itemsProp["type"])
	}
}

func TestGenerateStatusSchemaFromResponse(t *testing.T) {
	oasContentWithGet := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items/{id}:
    get:
      summary: Get an item by ID
      responses:
        '200':
          description: A single item
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: integer
                  name:
                    type: string
                  active:
                    type: boolean
                  config:
                    type: object
                    properties:
                      key:
                        type: string
                  tags:
                    type: array
                    items:
                      type: string
components:
  securitySchemes: {}
`
	oasContentWithFindby := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      summary: Find items
      responses:
        '200':
          description: A list of items
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  properties:
                    id:
                      type: integer
                    name:
                      type: string
components:
  securitySchemes: {}
`

	tests := []struct {
		name                   string
		oasContent             string
		resource               definitionv1alpha1.Resource
		identifiers            []string
		expectedProperties     map[string]string
		expectedWarningCount   int
		expectedWarningMessage string
	}{
		{
			name:       "Correctly infer types from GET response",
			oasContent: oasContentWithGet,
			resource: definitionv1alpha1.Resource{
				Kind: "TestResource",
				VerbsDescription: []definitionv1alpha1.VerbsDescription{
					{Action: "get", Path: "/items/{id}", Method: "GET"},
				},
				AdditionalStatusFields: []string{"name", "active", "config", "tags", "missing_field"},
			},
			identifiers: []string{"id"},
			expectedProperties: map[string]string{
				"id":            "integer",
				"name":          "string",
				"active":        "boolean",
				"config":        "object",
				"tags":          "array",
				"missing_field": "string", // Should default to string
			},
			expectedWarningCount:   1,
			expectedWarningMessage: "status field 'missing_field' defined in RestDefinition not found in GET or FINDBY response schema, defaulting to string",
		},
		{
			name:       "Correctly infer types from FINDBY response",
			oasContent: oasContentWithFindby,
			resource: definitionv1alpha1.Resource{
				Kind: "TestResource",
				VerbsDescription: []definitionv1alpha1.VerbsDescription{
					{Action: "findby", Path: "/items", Method: "GET"},
				},
				AdditionalStatusFields: []string{"name"},
			},
			identifiers: []string{"id"},
			expectedProperties: map[string]string{
				"id":   "integer",
				"name": "string",
			},
			expectedWarningCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := libopenapi.NewDocument([]byte(tt.oasContent))
			if err != nil {
				t.Fatalf("failed to create document: %v", err)
			}
			model, modelErrors := doc.BuildV3Model()
			if len(modelErrors) > 0 {
				t.Fatalf("failed to build model: %v", errors.Join(modelErrors...))
			}

			gen, warnings, fatalError := GenerateByteSchemas(model, tt.resource, tt.identifiers)
			if fatalError != nil {
				t.Fatalf("fatal error: %v", fatalError)
			}

			if len(warnings) != tt.expectedWarningCount {
				t.Errorf("expected %d warnings, got %d: %v", tt.expectedWarningCount, len(warnings), warnings)
			}

			if tt.expectedWarningCount > 0 {
				found := false
				for _, w := range warnings {
					if strings.Contains(w.Error(), tt.expectedWarningMessage) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected warning message not found: '%s'", tt.expectedWarningMessage)
				}
			}

			statusSchemaBytes, err := gen.OASStatusJsonSchemaGetter().Get()
			if err != nil {
				t.Fatalf("failed to get status schema bytes: %v", err)
			}

			var statusSchema map[string]interface{}
			err = json.Unmarshal(statusSchemaBytes, &statusSchema)
			if err != nil {
				t.Fatalf("failed to unmarshal status schema: %v", err)
			}

			properties, ok := statusSchema["properties"].(map[string]interface{})
			if !ok {
				t.Fatalf("status schema does not contain properties map")
			}

			if len(properties) != len(tt.expectedProperties) {
				t.Errorf("expected %d properties, got %d", len(tt.expectedProperties), len(properties))
			}

			for field, expectedType := range tt.expectedProperties {
				prop, ok := properties[field].(map[string]interface{})
				if !ok {
					t.Errorf("expected field '%s' not found in status properties", field)
					continue
				}
				// The type can be a single string or a slice of strings (e.g., ["string", "null"])
				// For simplicity in this test, we'll just check the first type.
				var actualType string
				if typeSlice, ok := prop["type"].([]interface{}); ok && len(typeSlice) > 0 {
					actualType = typeSlice[0].(string)
				} else if typeStr, ok := prop["type"].(string); ok {
					actualType = typeStr
				}

				if actualType != expectedType {
					t.Errorf("for field '%s', expected type '%s', got '%s'", field, expectedType, actualType)
				}
			}
		})
	}
}
