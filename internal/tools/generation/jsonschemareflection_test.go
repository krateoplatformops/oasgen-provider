package generation_test

import (
	"reflect"
	"testing"

	"github.com/krateoplatformops/oasgen-provider/internal/tools/generation"
	"github.com/stretchr/testify/require"
)

func TestReflectBytes(t *testing.T) {
	type TestStruct struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	expectedSchema := `{
		"type": "object",
		"properties": {
			"id": {
				"type": "int"
			},
			"name": {
				"type": "string"
			}
		},
		"required": ["id", "name"]
	}`

	schemaBytes, err := generation.ReflectBytes(reflect.TypeOf(TestStruct{}))
	if err != nil {
		t.Errorf("error reflecting bytes: %v", err)
	}

	require.JSONEq(t, expectedSchema, string(schemaBytes))
}
