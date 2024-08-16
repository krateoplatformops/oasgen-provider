package jsonschema

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRefResolver_Init(t *testing.T) {
	schema := &Schema{ID: "http://example.com/schema"}
	resolver := NewRefResolver([]*Schema{schema})
	err := resolver.Init()

	assert.NoError(t, err)
	assert.NotNil(t, resolver.pathToSchema)
	assert.Contains(t, resolver.pathToSchema, schema.ID)
}

func TestRefResolver_GetPath(t *testing.T) {
	root := &Schema{}
	child := &Schema{Parent: root, PathElement: "child"}
	resolver := NewRefResolver([]*Schema{root, child})

	path := resolver.GetPath(child)
	assert.Equal(t, "/child", path)
}
func TestRefResolver_GetSchemaByReference(t *testing.T) {
	root := &Schema{ID: "http://example.com/schema", Reference: "#/child"}
	child := &Schema{Parent: root, PathElement: "child", ID: "http://example.com/child"}
	resolver := NewRefResolver([]*Schema{root, child})
	_ = resolver.Init()

	b, _ := json.Marshal([]*Schema{root, child})
	fmt.Println(string(b))

	refSchema := &Schema{Parent: root, Reference: "child"}

	schema, err := resolver.GetSchemaByReference(refSchema)
	assert.NoError(t, err)
	assert.Equal(t, child, schema)
}
func TestRefResolver_GetSchemaByReference_Error(t *testing.T) {
	root := &Schema{ID: "http://example.com/schema"}
	resolver := NewRefResolver([]*Schema{root})
	_ = resolver.Init()

	refSchema := &Schema{Parent: root, Reference: "#/nonexistent"}

	_, err := resolver.GetSchemaByReference(refSchema)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reference not found")
}
