package generation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"sigs.k8s.io/yaml"
)

// recursive function to build the map of properties
func buildPropMap(t reflect.Type) (*orderedmap.Map[string, *base.SchemaProxy], []string, error) {
	// Create a new ordered map to store the properties
	propMap := orderedmap.New[string, *base.SchemaProxy]()
	// Required fields
	required := []string{}
	inlineRequired := []string{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldType := field.Type
		fieldName := field.Tag.Get("json")
		split := strings.Split(fieldName, ",") // split the json tag by comma to get the field name
		if len(split) > 1 {
			fieldName = split[0]
		} else {
			required = append(required, fieldName)
		}

		if fieldType.Kind() == reflect.Struct {
			// Recursively reflect the struct
			fieldMap, req, err := buildPropMap(fieldType)
			if err != nil {
				return nil, nil, err
			}

			if fieldName == "" {
				inlineRequired = []string{}
				for k := fieldMap.First(); k != nil; k = k.Next() {
					propMap.Set(k.Key(), k.Value())
					inlineRequired = append(inlineRequired, k.Key())
				}
			} else {
				propMap.Set(fieldName, base.CreateSchemaProxy(&base.Schema{
					Type:       []string{"object"},
					Properties: fieldMap,
					Required:   req,
				}))
			}
		} else {
			// Reflect the field
			propMap.Set(fieldName, base.CreateSchemaProxy(&base.Schema{Type: []string{fieldType.Name()}}))
		}
	}
	return propMap, append(required, inlineRequired...), nil
}

func ReflectBytes(t reflect.Type) ([]byte, error) {
	if t == nil {
		return nil, nil
	}

	propMap, req, err := buildPropMap(t)

	if err != nil {
		return nil, err
	}

	schemaproxy := base.CreateSchemaProxy(&base.Schema{
		Type:       []string{"object"},
		Properties: propMap,
		Required:   req,
	})
	if schemaproxy == nil {
		return nil, fmt.Errorf("schemaproxy is nil")
	}

	bSchemaYAML, err := schemaproxy.Render()
	if err != nil {
		return nil, err
	}
	bSchemaJSON, err := yaml.YAMLToJSON(bSchemaYAML)
	if err != nil {
		return nil, err
	}
	return bSchemaJSON, nil
}
