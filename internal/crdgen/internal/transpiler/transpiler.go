package transpiler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/krateoplatformops/oasgen-provider/internal/crdgen/internal/ptr"
	"github.com/krateoplatformops/oasgen-provider/internal/crdgen/internal/strutil"
	"github.com/krateoplatformops/oasgen-provider/internal/crdgen/internal/transpiler/jsonschema"
)

// Field defines the data required to generate a field in Go.
type Field struct {
	// The golang name, e.g. "Address1"
	Name string
	// The JSON name, e.g. "address1"
	JSONName string
	// The golang type of the field, e.g. a built-in type like "string" or the name of a struct generated
	// from the JSON schema.
	Type string

	Description string

	// Required is set to true when the field is required.
	Required bool
	// Optional
	Optional *bool

	Default any

	Minimum, Maximum, MultipleOf *float64

	Pattern *string

	Enum []string
}

// Struct defines the data required to generate a struct in Go.
type Struct struct {
	// The ID within the JSON schema, e.g. #/definitions/address
	ID string
	// The golang name, e.g. "Address"
	Name string
	// Description of the struct
	Description string
	Fields      map[string]Field

	GenerateCode   bool
	AdditionalType string
}

// Transpile creates an instance of a generator which will produce structs.
func Transpile(schemas ...*jsonschema.Schema) (map[string]Struct, error) {
	res := &transpiler{
		schemas:  schemas,
		resolver: jsonschema.NewRefResolver(schemas),
		Structs:  make(map[string]Struct),
		Aliases:  make(map[string]Field),
		refs:     make(map[string]string),
	}
	err := res.createStructs()

	return res.Structs, err
}

// transpiler will produce structs from the JSON schema.
type transpiler struct {
	schemas  []*jsonschema.Schema
	resolver *jsonschema.RefResolver
	Structs  map[string]Struct
	Aliases  map[string]Field
	// cache for reference types; k=url v=type
	refs      map[string]string
	anonCount int
}

func (g *transpiler) createField(name, rootType string, schema *jsonschema.Schema) Field {
	f := Field{
		Name:        name,
		JSONName:    "",
		Type:        rootType,
		Required:    false,
		Optional:    ptr.To(ptr.Deref(schema.Optional, false)),
		Description: schema.Description,
	}

	if schema.Default != nil {
		f.Default = schema.Default
	}

	if schema.Minimum != nil {
		f.Minimum = ptr.To(*schema.Minimum)
	}

	if schema.Maximum != nil {
		f.Maximum = ptr.To(*schema.Maximum)
	}

	if schema.MultipleOf != nil {
		f.MultipleOf = ptr.To(*schema.MultipleOf)
	}

	if schema.Enum != nil {
		f.Enum = strslice(schema.Enum)
	}

	if schema.Pattern != nil {
		f.Pattern = ptr.To(*schema.Pattern)
	}

	return f
}

// createStructs creates types from the JSON schemas, keyed by the golang name.
func (g *transpiler) createStructs() (err error) {
	if err := g.resolver.Init(); err != nil {
		return err
	}

	// extract the types
	for _, schema := range g.schemas {
		name := g.getSchemaName("", schema)
		rootType, err := g.processSchema(name, schema)
		if err != nil {
			return err
		}
		// ugh: if it was anything but a struct the type will not be the name...
		if rootType != "*"+name {
			f := g.createField(name, rootType, schema)
			g.Aliases[name] = f
		}
	}
	return
}

// process a block of definitions
func (g *transpiler) processDefinitions(schema *jsonschema.Schema) error {
	for key, subSchema := range schema.Definitions {
		if _, err := g.processSchema(strutil.ToGolangName(key), subSchema); err != nil {
			return err
		}
	}
	return nil
}

// process a reference string
func (g *transpiler) processReference(schema *jsonschema.Schema) (string, error) {
	schemaPath := g.resolver.GetPath(schema)
	if schema.Reference == "" {
		return "", errors.New("processReference empty reference: " + schemaPath)
	}
	refSchema, err := g.resolver.GetSchemaByReference(schema)
	if err != nil {
		return "", errors.New("processReference: reference \"" + schema.Reference + "\" not found at \"" + schemaPath + "\"")
	}
	if refSchema.GeneratedType == "" {
		// reference is not resolved yet. Do that now.
		refSchemaName := g.getSchemaName("", refSchema)
		typeName, err := g.processSchema(refSchemaName, refSchema)
		if err != nil {
			return "", err
		}
		return typeName, nil
	}
	return refSchema.GeneratedType, nil
}

// returns the type refered to by schema after resolving all dependencies
func (g *transpiler) processSchema(schemaName string, schema *jsonschema.Schema) (typ string, err error) {
	if len(schema.Definitions) > 0 {
		g.processDefinitions(schema)
	}
	schema.FixMissingTypeValue()
	// if we have multiple schema types, the golang type will be string ////any
	typ = "string" ////"any"
	types, isMultiType := schema.MultiType()
	if len(types) > 0 {
		for _, schemaType := range types {
			name := schemaName
			if isMultiType {
				name = name + "_" + schemaType
			}
			switch schemaType {
			case "object":
				rv, err := g.processObject(name, schema)
				if err != nil {
					return "", err
				}
				if !isMultiType {
					return rv, nil
				}
			case "array":
				rv, err := g.processArray(name, schema)
				if err != nil {
					return "", err
				}
				if !isMultiType {
					return rv, nil
				}
			default:
				rv, err := getPrimitiveTypeName(schemaType, "", false)
				if err != nil {
					return "", err
				}
				if !isMultiType {
					return rv, nil
				}
			}
		}
	} else {
		if schema.Reference != "" {
			return g.processReference(schema)
		}
	}
	return
}

// name: name of this array, usually the js key
// schema: items element
func (g *transpiler) processArray(name string, schema *jsonschema.Schema) (typeStr string, err error) {
	if schema.Items != nil {
		// subType: fallback name in case this array contains inline object without a title
		subName := g.getSchemaName(name+"Items", schema.Items)
		subTyp, err := g.processSchema(subName, schema.Items)
		if err != nil {
			return "", err
		}
		finalType, err := getPrimitiveTypeName("array", subTyp, true)
		if err != nil {
			return "", err
		}
		// only alias root arrays
		if schema.Parent == nil {
			f := g.createField(name, finalType, schema)
			f.Required = contains(schema.Required, name)
			g.Aliases[name] = f
		}
		return finalType, nil
	}
	return "[]any", nil
}

// name: name of the struct (calculated by caller)
// schema: detail incl properties & child objects
// returns: generated type
func (g *transpiler) processObject(name string, schema *jsonschema.Schema) (typ string, err error) {
	strct := Struct{
		Name:        name,
		Description: schema.Description,
		Fields:      make(map[string]Field, len(schema.Properties)),
	}
	// cache the object name in case any sub-schemas recursively reference it
	schema.GeneratedType = "*" + name
	// regular properties
	for propKey, prop := range schema.Properties {
		fieldName := strutil.ToGolangName(propKey)
		// calculate sub-schema name here, may not actually be used depending on type of schema!
		subSchemaName := g.getSchemaName(fieldName, prop)
		fieldType, err := g.processSchema(subSchemaName, prop)
		if err != nil {
			return "", err
		}

		f := g.createField(fieldName, fieldType, prop)
		f.JSONName = propKey
		f.Required = contains(schema.Required, propKey)
		if f.Required {
			strct.GenerateCode = true
		}
		strct.Fields[fieldName] = f
	}
	// additionalProperties with typed sub-schema
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.AdditionalPropertiesBool == nil {
		ap := (*jsonschema.Schema)(schema.AdditionalProperties)
		apName := g.getSchemaName("", ap)
		subTyp, err := g.processSchema(apName, ap)
		if err != nil {
			return "", err
		}
		mapTyp := "map[string]" + subTyp
		// If this object is inline property for another object, and only contains additional properties, we can
		// collapse the structure down to a map.
		//
		// If this object is a definition and only contains additional properties, we can't do that or we end up with
		// no struct
		isDefinitionObject := strings.HasPrefix(schema.PathElement, "definitions")
		if len(schema.Properties) == 0 && !isDefinitionObject {
			// since there are no regular properties, we don't need to emit a struct for this object - return the
			// additionalProperties map type.
			return mapTyp, nil
		}
		// this struct will have both regular and additional properties
		f := Field{
			Name:        "AdditionalProperties",
			JSONName:    "-",
			Type:        mapTyp,
			Required:    false,
			Optional:    ptr.To(true),
			Description: "",
		}
		strct.Fields[f.Name] = f
		// setting this will cause marshal code to be emitted in Output()
		strct.GenerateCode = true
		strct.AdditionalType = subTyp
	}
	// additionalProperties as either true (everything) or false (nothing)
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.AdditionalPropertiesBool != nil {
		if *schema.AdditionalProperties.AdditionalPropertiesBool {
			// everything is valid additional
			subTyp := "map[string]any"
			f := Field{
				Name:        "AdditionalProperties",
				JSONName:    "-",
				Type:        subTyp,
				Required:    false,
				Optional:    ptr.To(true),
				Description: "",
			}
			strct.Fields[f.Name] = f
			// setting this will cause marshal code to be emitted in Output()
			strct.GenerateCode = true
			strct.AdditionalType = "any"
		} else {
			// nothing
			strct.GenerateCode = true
			strct.AdditionalType = "false"
		}
	}
	g.Structs[strct.Name] = strct
	// objects are always a pointer
	return getPrimitiveTypeName("object", name, true)
}

// return a name for this (sub-)schema.
func (g *transpiler) getSchemaName(keyName string, schema *jsonschema.Schema) string {
	if len(schema.Title) > 0 {
		return strutil.ToGolangName(schema.Title)
	}
	if keyName != "" {
		return strutil.ToGolangName(keyName)
	}
	if schema.Parent == nil {
		return "Root"
	}
	if schema.JSONKey != "" {
		return strutil.ToGolangName(schema.JSONKey)
	}
	if schema.Parent != nil && schema.Parent.JSONKey != "" {
		return strutil.ToGolangName(schema.Parent.JSONKey + "Item")
	}
	g.anonCount++
	return fmt.Sprintf("Anonymous%d", g.anonCount)
}

func getPrimitiveTypeName(schemaType string, subType string, pointer bool) (name string, err error) {
	switch schemaType {
	case "array":
		if subType == "" {
			return "error_creating_array", errors.New("can't create an array of an empty subtype")
		}
		return "[]" + subType, nil
	case "boolean":
		return "bool", nil
	case "integer":
		return "int", nil
	case "number":
		return "float64", nil
	case "null":
		return "nil", nil
	case "object":
		if subType == "" {
			return "error_creating_object", errors.New("can't create an object of an empty subtype")
		}
		if pointer {
			return "*" + subType, nil
		}
		return subType, nil
	case "string":
		return "string", nil
	}

	return "undefined", fmt.Errorf("failed to get a primitive type for schemaType %s and subtype %s",
		schemaType, subType)
}
