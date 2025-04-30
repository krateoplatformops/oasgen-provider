package generation

import (
	"fmt"
	"reflect"

	rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
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
	Username    string                 `json:"username"`
	PasswordRef rtv1.SecretKeySelector `json:"passwordRef"`
}

type BearerAuth struct {
	TokenRef rtv1.SecretKeySelector `json:"tokenRef"`
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
		return ReflectBytes(reflect.TypeOf(BasicAuth{}))
	} else if doc.Type == "http" && doc.Scheme == "bearer" {
		return ReflectBytes(reflect.TypeOf(BearerAuth{}))
	}

	return nil, fmt.Errorf("error: %v", ErrInvalidSecuritySchema)
}
