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
