package oas2jsonschema

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultGeneratorConfig(t *testing.T) {
	t.Run("should return a config with default values", func(t *testing.T) {
		// Execute
		config := DefaultGeneratorConfig()

		// Assert
		assert.NotNil(t, config)
		assert.Equal(t, []string{"application/json"}, config.AcceptedMIMETypes)
		assert.Equal(t, []int{http.StatusOK, http.StatusCreated}, config.SuccessCodes)
	})
}

func TestSchema_DeepCopy(t *testing.T) {
	t.Run("should return nil for a nil schema", func(t *testing.T) {
		var original *Schema = nil
		copied := original.deepCopy()
		assert.Nil(t, copied)
	})

	t.Run("should create a distinct copy of a simple schema", func(t *testing.T) {
		original := &Schema{Description: "original"}
		copied := original.deepCopy()

		// Assert that the copy is not the same object as the original
		assert.NotSame(t, original, copied)
		// Assert that the content is the same
		assert.Equal(t, original.Description, copied.Description)
	})

	t.Run("should ensure that modifying the copy does not affect the original", func(t *testing.T) {
		original := &Schema{
			Description: "original",
			Properties: []Property{
				{Name: "prop1", Schema: &Schema{Description: "nested original"}},
			},
		}
		copied := original.deepCopy()

		// Modify the copy
		copied.Description = "modified"
		copied.Properties[0].Schema.Description = "nested modified"

		// Assert that the original remains unchanged
		assert.Equal(t, "original", original.Description)
		assert.Equal(t, "nested original", original.Properties[0].Schema.Description)
	})

	t.Run("should correctly copy a schema with nested properties and items", func(t *testing.T) {
		original := &Schema{
			Properties: []Property{
				{Name: "prop1", Schema: &Schema{Description: "prop1 schema"}},
			},
			Items: &Schema{Description: "items schema"},
		}
		copied := original.deepCopy()

		// Assert that nested structures are not the same objects
		assert.NotSame(t, original.Properties[0].Schema, copied.Properties[0].Schema)
		assert.NotSame(t, original.Items, copied.Items)

		// Assert that the content of nested structures is the same
		assert.Equal(t, "prop1 schema", copied.Properties[0].Schema.Description)
		assert.Equal(t, "items schema", copied.Items.Description)
	})

	t.Run("should handle direct circular references without stack overflow", func(t *testing.T) {
		original := &Schema{Description: "circular"}
		// Create a circular reference
		original.Items = original

		var copied *Schema
		// This will stack overflow if deepCopy is not recursion-safe
		require.NotPanics(t, func() {
			copied = original.deepCopy()
		})

		// Assert that the copied structure is also circular
		assert.NotNil(t, copied)
		assert.Equal(t, "circular", copied.Description)
		assert.Same(t, copied, copied.Items, "The copied schema should also have a circular reference")
	})

	t.Run("should handle indirect circular references without stack overflow", func(t *testing.T) {
		schemaA := &Schema{Description: "A"}
		schemaB := &Schema{Description: "B"}

		// Create indirect circular reference A -> B -> A
		schemaA.Properties = []Property{{Name: "propB", Schema: schemaB}}
		schemaB.Properties = []Property{{Name: "propA", Schema: schemaA}}

		var copiedA *Schema
		require.NotPanics(t, func() {
			copiedA = schemaA.deepCopy()
		})

		// Assert that the copied structure is also circular
		assert.NotNil(t, copiedA)
		copiedB := copiedA.Properties[0].Schema
		assert.NotNil(t, copiedB)
		assert.Equal(t, "B", copiedB.Description)

		// Check that the cycle is completed correctly in the copy
		assert.Same(t, copiedA, copiedB.Properties[0].Schema)
	})

	t.Run("should correctly copy a schema with AllOf", func(t *testing.T) {
		original := &Schema{
			AllOf: []*Schema{
				{Description: "allOf schema 1"},
			},
		}
		copied := original.deepCopy()

		require.Len(t, copied.AllOf, 1)
		// Assert that nested structures are not the same objects
		assert.NotSame(t, original.AllOf[0], copied.AllOf[0])
		// Assert that the content is the same
		assert.Equal(t, "allOf schema 1", copied.AllOf[0].Description)

		// Modify the copy and check the original is unchanged
		copied.AllOf[0].Description = "modified"
		assert.Equal(t, "allOf schema 1", original.AllOf[0].Description)
	})

	t.Run("should correctly copy a schema with Enum", func(t *testing.T) {
		original := &Schema{
			Enum: []interface{}{"a", 1, "c"},
		}
		copied := original.deepCopy()

		assert.Equal(t, original.Enum, copied.Enum)

		// Modify the copy and check the original is unchanged
		copied.Enum[0] = "z"
		assert.Equal(t, "a", original.Enum[0])
	})

	t.Run("should correctly copy a schema with a Default value", func(t *testing.T) {
		original := &Schema{
			Default: "default value",
		}
		copied := original.deepCopy()

		assert.Equal(t, original.Default, copied.Default)

		// Modify the copy and check the original is unchanged
		copied.Default = "new default"
		assert.Equal(t, "default value", original.Default)
	})
}