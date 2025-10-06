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

	t.Run("should shallow copy Default field with reference type", func(t *testing.T) {
		originalMap := map[string]string{"key": "original"}
		original := &Schema{Default: originalMap}
		copied := original.deepCopy()

		assert.Equal(t, original.Default, copied.Default)

		// Modify the map in the copied schema
		copiedMap := copied.Default.(map[string]string)
		copiedMap["key"] = "modified"

		// Assert that the original map is also modified
		assert.Equal(t, "modified", originalMap["key"], "Default field should be a shallow copy")
	})

	t.Run("should shallow copy Enum field with reference type", func(t *testing.T) {
		originalMap := map[string]string{"key": "original"}
		original := &Schema{Enum: []interface{}{originalMap}}
		copied := original.deepCopy()

		assert.Equal(t, original.Enum, copied.Enum)

		// Modify the map in the copied schema's enum
		copiedMap := copied.Enum[0].(map[string]string)
		copiedMap["key"] = "modified"

		// Assert that the original map is also modified
		assert.Equal(t, "modified", originalMap["key"], "Enum elements should be shallow copied")
	})

	t.Run("should distinguish between nil and empty slices", func(t *testing.T) {
		originalNil := &Schema{
			Properties: nil,
			Required:   nil,
		}
		copiedNil := originalNil.deepCopy()
		assert.Nil(t, copiedNil.Properties)
		assert.Nil(t, copiedNil.Required)

		originalEmpty := &Schema{
			Properties: []Property{},
			Required:   []string{},
		}
		copiedEmpty := originalEmpty.deepCopy()
		assert.NotNil(t, copiedEmpty.Properties)
		assert.Len(t, copiedEmpty.Properties, 0)
		assert.NotNil(t, copiedEmpty.Required)
		assert.Len(t, copiedEmpty.Required, 0)
	})

	t.Run("should handle highly complex nested schemas with multiple circular references", func(t *testing.T) {
		// A -> B -> C -> A (cycle 1)
		// A -> D -> A (cycle 2)
		// C has a nested, non-cyclic property
		// D is part of an AllOf
		// B has a nil property schema
		schemaA := &Schema{Description: "A"}
		schemaB := &Schema{Description: "B"}
		schemaC := &Schema{Description: "C"}
		schemaD := &Schema{Description: "D"}
		nestedInC := &Schema{Description: "NestedInC"}
		allOfD := &Schema{Description: "AllOfD"}

		// Build the structure
		schemaA.Properties = []Property{
			{Name: "propB", Schema: schemaB},
			{Name: "propD", Schema: schemaD},
		}
		schemaB.Properties = []Property{
			{Name: "propC", Schema: schemaC},
			{Name: "nilProp", Schema: nil},
		}
		schemaC.Properties = []Property{
			{Name: "propA", Schema: schemaA},
			{Name: "nested", Schema: nestedInC},
		}
		schemaD.AllOf = []*Schema{allOfD}
		allOfD.Properties = []Property{
			{Name: "backToA", Schema: schemaA},
		}

		// Perform the deep copy
		var copiedA *Schema
		require.NotPanics(t, func() {
			copiedA = schemaA.deepCopy()
		})

		// --- Verification ---
		// 1. Basic structure and distinctness
		assert.NotNil(t, copiedA)
		assert.NotSame(t, schemaA, copiedA)
		assert.Equal(t, "A", copiedA.Description)

		// 2. Traverse and verify copied structure
		require.Len(t, copiedA.Properties, 2)
		copiedB := copiedA.Properties[0].Schema
		copiedD := copiedA.Properties[1].Schema

		assert.NotSame(t, schemaB, copiedB)
		assert.Equal(t, "B", copiedB.Description)
		require.Len(t, copiedB.Properties, 2)
		assert.Nil(t, copiedB.Properties[1].Schema, "Nil property schema should be preserved")

		copiedC := copiedB.Properties[0].Schema
		assert.NotSame(t, schemaC, copiedC)
		assert.Equal(t, "C", copiedC.Description)
		require.Len(t, copiedC.Properties, 2)

		copiedNestedInC := copiedC.Properties[1].Schema
		assert.NotSame(t, nestedInC, copiedNestedInC)
		assert.Equal(t, "NestedInC", copiedNestedInC.Description)

		assert.NotSame(t, schemaD, copiedD)
		assert.Equal(t, "D", copiedD.Description)
		require.Len(t, copiedD.AllOf, 1)
		copiedAllOfD := copiedD.AllOf[0]
		assert.NotSame(t, allOfD, copiedAllOfD)
		assert.Equal(t, "AllOfD", copiedAllOfD.Description)
		require.Len(t, copiedAllOfD.Properties, 1)

		// 3. Verify both circular references are intact in the new structure
		assert.Same(t, copiedA, copiedC.Properties[0].Schema, "Cycle 1 (A->B->C->A) should be preserved")
		assert.Same(t, copiedA, copiedAllOfD.Properties[0].Schema, "Cycle 2 (A->D->A) should be preserved")

		// 4. Verify that modifying the deep copy does not affect the original
		copiedA.Description = "Modified A"
		copiedB.Properties[0].Schema.Description = "Modified C"
		assert.Equal(t, "A", schemaA.Description)
		assert.Equal(t, "C", schemaC.Description)
	})

	t.Run("should handle complex AllOf with nesting, cycles, and nils", func(t *testing.T) {
		// A -> allOf(B, C, nil)
		// B -> prop -> A (cycle)
		// C -> allOf(D)
		schemaA := &Schema{Description: "A"}
		schemaB := &Schema{Description: "B"}
		schemaC := &Schema{Description: "C"}
		schemaD := &Schema{Description: "D"}

		// Build the structure
		schemaA.AllOf = []*Schema{schemaB, schemaC, nil}
		schemaB.Properties = []Property{{Name: "backToA", Schema: schemaA}}
		schemaC.AllOf = []*Schema{schemaD}

		// Perform the deep copy
		var copiedA *Schema
		require.NotPanics(t, func() {
			copiedA = schemaA.deepCopy()
		})

		// --- Verification ---
		// 1. Basic structure and distinctness
		assert.NotNil(t, copiedA)
		assert.NotSame(t, schemaA, copiedA)
		assert.Equal(t, "A", copiedA.Description)

		// 2. Verify AllOf structure
		require.Len(t, copiedA.AllOf, 3)
		assert.Nil(t, copiedA.AllOf[2], "Nil entry in AllOf should be preserved")

		copiedB := copiedA.AllOf[0]
		copiedC := copiedA.AllOf[1]

		assert.NotSame(t, schemaB, copiedB)
		assert.Equal(t, "B", copiedB.Description)
		require.Len(t, copiedB.Properties, 1)

		assert.NotSame(t, schemaC, copiedC)
		assert.Equal(t, "C", copiedC.Description)
		require.Len(t, copiedC.AllOf, 1)

		copiedD := copiedC.AllOf[0]
		assert.NotSame(t, schemaD, copiedD)
		assert.Equal(t, "D", copiedD.Description)

		// 3. Verify circular reference is intact
		assert.Same(t, copiedA, copiedB.Properties[0].Schema, "Cycle through AllOf should be preserved")

		// 4. Verify modification isolation
		copiedB.Description = "Modified B"
		assert.Equal(t, "B", schemaB.Description)
	})
}
