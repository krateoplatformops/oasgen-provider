package jsonschema_test

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/krateoplatformops/oasgen-provider/internal/crdgen/internal/transpiler/jsonschema"
)

func TestThatTheRootSchemaCanBeParsed(t *testing.T) {
	s := `{
        "$schema": "http://json-schema.org/schema#",
        "title": "root"
    }`

	so, err := jsonschema.Parse([]byte(s))
	if err != nil {
		t.Fatal("It should be possible to unmarshal a simple schema, but received error:", err)
	}

	if so.Title != "root" {
		t.Errorf("The title was not deserialised from the JSON schema, expected %s, but got %s", "root", so.Title)
	}
}

func TestThatPropertiesCanBeParsed(t *testing.T) {
	s := `{
        "$schema": "http://json-schema.org/schema#",
        "title": "root",
        "properties": {
            "name": {
                "type": "string"
            },
            "address": {
                "$ref": "#/definitions/address"
            },
            "status": {
                "$ref": "#/definitions/status"
            }
        }
    }`

	so, err := jsonschema.Parse([]byte(s))
	if err != nil {
		t.Fatal("It was not possible to unmarshal the schema:", err)
	}

	nameType, nameMultiple := so.Properties["name"].Type()
	if nameType != "string" || nameMultiple {
		t.Errorf("expected property 'name' type to be 'string', but was '%v'", nameType)
	}

	addressType, _ := so.Properties["address"].Type()
	if addressType != "" {
		t.Errorf("expected property 'address' type to be '', but was '%v'", addressType)
	}

	if so.Properties["address"].Reference != "#/definitions/address" {
		t.Errorf("expected property 'address' reference to be '#/definitions/address', but was '%v'", so.Properties["address"].Reference)
	}

	if so.Properties["status"].Reference != "#/definitions/status" {
		t.Errorf("expected property 'status' reference to be '#/definitions/status', but was '%v'", so.Properties["status"].Reference)
	}
}

func TestThatPropertiesCanHaveMultipleTypes(t *testing.T) {
	s := `{
        "$schema": "http://json-schema.org/schema#",
        "title": "root",
        "properties": {
            "name": {
                "type": [ "integer", "string" ]
            }
        }
    }`
	so, err := jsonschema.Parse([]byte(s))
	if err != nil {
		t.Fatal("It was not possible to unmarshal the schema:", err)
	}

	nameType, nameMultiple := so.Properties["name"].Type()
	if nameType != "integer" {
		t.Errorf("expected first value of property 'name' type to be 'integer', but was '%v'", nameType)
	}

	if !nameMultiple {
		t.Errorf("expected multiple types, but only returned one")
	}
}

func TestThatParsingInvalidValuesReturnsAnError(t *testing.T) {
	s := `{ " }`
	_, err := jsonschema.Parse([]byte(s))
	if err == nil {
		t.Fatal("Expected a parsing error, but got nil")
	}
}

func TestThatDefaultsCanBeParsed(t *testing.T) {
	s := `{
        "$schema": "http://json-schema.org/schema#",
        "title": "root",
        "properties": {
            "name": {
                "type": [ "integer", "string" ],
                "default":"Enrique"
            }
        }
    }`
	so, err := jsonschema.Parse([]byte(s))
	if err != nil {
		t.Fatal("It was not possible to unmarshal the schema:", err)
	}

	defaultValue := so.Properties["name"].Default
	if defaultValue != "Enrique" {
		t.Errorf("expected default value of property 'name' type to be 'Enrique', but was '%v'", defaultValue)
	}
}

func TestThatObjectArrayCanBeParsed(t *testing.T) {
	s := `{
        "$schema": "http://json-schema.org/schema#",
        "title": "root",
        "properties": {
          "data": {
            "type": "array",
            "items": {
               
                  "type": "object",
                  "properties": {
                    "path": {
                       "type": "string"
                    },
                    "value": {
                        "type": "string"
                    }
                  },
                  "required": [
                    "path",
                    "value"
                  ]
                }
            }
        }
    }`
	_, err := jsonschema.Parse([]byte(s))
	if err != nil {
		t.Fatal("It was not possible to unmarshal the schema:", err)
	}

}

const (
	sample = `
{
    "$id": "https://example.com/person.schema.json",
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "title": "Person",
    "type": "object",
    "properties": {
        "firstName": {
            "type": "string",
            "description": "The person's first name."
        },
        "lastName": {
            "type": "string",
            "description": "The person's last name."
        },
        "age": {
            "description": "Age in years which must be equal to or greater than zero.",
            "type": "integer",
            "minimum": 10
        }
    }
}
`
)

func TestThatIntegerCanBeParsed(t *testing.T) {

	so, err := jsonschema.Parse([]byte(sample))
	if err != nil {
		t.Fatal("It was not possible to unmarshal the schema:", err)
	}

	spew.Dump(so.Properties["age"].Minimum)
}
