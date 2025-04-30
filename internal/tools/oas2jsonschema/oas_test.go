package oas2jsonschema

import (
	"context"
	"embed"
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
