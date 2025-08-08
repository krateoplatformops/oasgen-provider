package oas2jsonschema

import (
	"bytes"
	"testing"
)

func TestGetters(t *testing.T) {
	// Arrange
	mockResult := &GenerationResult{
		SpecSchema:   []byte(`{"spec": true}`),
		StatusSchema: []byte(`{"status": true}`),
		AuthCRDSchemas: map[string][]byte{
			"BasicAuth": []byte(`{"auth": "basic"}`),
		},
	}

	t.Run("OASSpecJsonSchemaGetter", func(t *testing.T) {
		// Act
		getter := mockResult.OASSpecJsonSchemaGetter()
		data, err := getter.Get()

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}
		if !bytes.Equal(data, mockResult.SpecSchema) {
			t.Errorf("Expected spec schema, but got: %s", string(data))
		}
	})

	t.Run("OASStatusJsonSchemaGetter", func(t *testing.T) {
		// Act
		getter := mockResult.OASStatusJsonSchemaGetter()
		data, err := getter.Get()

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}
		if !bytes.Equal(data, mockResult.StatusSchema) {
			t.Errorf("Expected status schema, but got: %s", string(data))
		}
	})

	t.Run("OASAuthCRDSchemaGetter", func(t *testing.T) {
		// Act
		getter := mockResult.OASAuthCRDSchemaGetter("BasicAuth")
		data, err := getter.Get()

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}
		if !bytes.Equal(data, mockResult.AuthCRDSchemas["BasicAuth"]) {
			t.Errorf("Expected auth schema, but got: %s", string(data))
		}
	})

	t.Run("StaticJsonSchemaGetter", func(t *testing.T) {
		// Act
		getter := StaticJsonSchemaGetter()
		data, err := getter.Get()

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}
		if data != nil {
			t.Errorf("Expected nil data, but got: %s", string(data))
		}
	})
}
