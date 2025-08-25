package oas2jsonschema

import (
	"reflect"
	"strings"
)

// reflectSchema generates a schema from a Go type using reflection.
func reflectSchema(t reflect.Type) (*Schema, error) {
	if t == nil {
		return nil, nil
	}

	props, req, err := buildSchemaProperties(t)
	if err != nil {
		return nil, err
	}

	return &Schema{
		Type:       []string{"object"},
		Properties: props,
		Required:   req,
	}, nil
}

// buildSchemaProperties recursively builds the properties of a schema from a Go type.
func buildSchemaProperties(t reflect.Type) ([]Property, []string, error) {
	var props []Property
	var required []string
	var inlineRequired []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldType := field.Type
		fieldName := field.Tag.Get("json")
		split := strings.Split(fieldName, ",")
		if len(split) > 1 {
			fieldName = split[0]
		} else if fieldName != "" {
			required = append(required, fieldName)
		}

		if fieldType.Kind() == reflect.Struct {
			fieldProps, req, err := buildSchemaProperties(fieldType)
			if err != nil {
				return nil, nil, err
			}

			if fieldName == "" {
				props = append(props, fieldProps...)
				inlineRequired = append(inlineRequired, req...)
			} else {
				props = append(props, Property{
					Name: fieldName,
					Schema: &Schema{
						Type:       []string{"object"},
						Properties: fieldProps,
						Required:   req,
					},
				})
			}
		} else {
			props = append(props, Property{
				Name:   fieldName,
				Schema: &Schema{Type: []string{fieldType.Name()}},
			})
		}
	}
	return props, append(required, inlineRequired...), nil
}
