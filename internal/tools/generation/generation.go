package generation

import (
	"fmt"

	"github.com/invopop/jsonschema"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"sigs.k8s.io/yaml"
)

const (
	ErrInvalidSecuritySchema = "invalid security schema type or scheme"
)

func GenerateJsonSchemaFromSchemaProxy(schema *base.SchemaProxy) ([]byte, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}

	bSchemaYAML, err := schema.Render()
	if err != nil {
		return nil, err
	}
	bSchemaJSON, err := yaml.YAMLToJSON(bSchemaYAML)
	if err != nil {
		return nil, err
	}
	return bSchemaJSON, nil
}

type BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type BearerAuth struct {
	Token string `json:"token"`
}

func IsValidAuthSchema(doc *v3.SecurityScheme) bool {
	if doc.Type == "http" && (doc.Scheme == "basic" || doc.Scheme == "bearer") {
		return true
	}
	return false
}

func GenerateAuthSchemaName(doc *v3.SecurityScheme) (string, error) {
	if doc.Type == "http" && doc.Scheme == "basic" {
		return "BasicAuth", nil
	} else if doc.Type == "http" && doc.Scheme == "bearer" {
		return "BearerAuth", nil
	}
	return "", fmt.Errorf("type: %s - %v", doc.Type, ErrInvalidSecuritySchema)
}

func GenerateAuthSchemaFromSecuritySchema(doc *v3.SecurityScheme) (byteSchema []byte, err error) {
	if doc.Type == "http" && doc.Scheme == "basic" {
		authSchema := jsonschema.Reflect(&BasicAuth{})
		byteSchema, err = authSchema.Definitions["BasicAuth"].MarshalJSON()
		return byteSchema, err
	} else if doc.Type == "http" && doc.Scheme == "bearer" {
		authSchema := jsonschema.Reflect(&BearerAuth{})
		byteSchema, err = authSchema.Definitions["BearerAuth"].MarshalJSON()
		return byteSchema, err
	}

	return nil, fmt.Errorf(ErrInvalidSecuritySchema)
}
