package oas2jsonschema

import (
	"strconv"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

// --- Adapter Implementation ---

type libOASDocumentAdapter struct {
	doc *libopenapi.DocumentModel[v3.Document]
}

// We implement the OASDocument interface for the libopenapi DocumentModel
func NewLibOASDocumentAdapter(doc *libopenapi.DocumentModel[v3.Document]) OASDocument {
	return &libOASDocumentAdapter{doc: doc}
}

func (a *libOASDocumentAdapter) FindPath(path string) (PathItem, bool) {
	p, ok := a.doc.Model.Paths.PathItems.Get(path)
	if !ok {
		return nil, false
	}
	return &libOASPathItemAdapter{path: p}, true
}

func (a *libOASDocumentAdapter) SecuritySchemes() []SecuritySchemeInfo {
	if a.doc.Model.Components == nil || a.doc.Model.Components.SecuritySchemes == nil {
		return nil
	}
	var schemes []SecuritySchemeInfo
	for pair := a.doc.Model.Components.SecuritySchemes.First(); pair != nil; pair = pair.Next() {
		v := pair.Value()
		schemes = append(schemes, SecuritySchemeInfo{
			Name:      pair.Key(),
			Type:      SecuritySchemeType(v.Type),
			Scheme:    v.Scheme,
			In:        v.In,
			ParamName: v.Name,
		})
	}
	return schemes
}

type libOASPathItemAdapter struct {
	path *v3.PathItem
}

func (a *libOASPathItemAdapter) GetOperations() map[string]Operation {
	ops := make(map[string]Operation)
	rawOps := a.path.GetOperations()
	for pair := rawOps.First(); pair != nil; pair = pair.Next() {
		ops[pair.Key()] = &libOASOperationAdapter{op: pair.Value()}
	}
	return ops
}

type libOASOperationAdapter struct {
	op *v3.Operation
}

func (a *libOASOperationAdapter) GetParameters() []ParameterInfo {
	var params []ParameterInfo
	for _, p := range a.op.Parameters {
		params = append(params, ParameterInfo{
			Name:        p.Name,
			In:          p.In,
			Description: p.Description,
			Schema:      convertLibopenapiSchema(p.Schema),
		})
	}
	return params
}

func (a *libOASOperationAdapter) GetRequestBody() RequestBodyInfo {
	if a.op.RequestBody == nil || a.op.RequestBody.Content == nil {
		return RequestBodyInfo{}
	}
	content := make(map[string]*Schema)
	for pair := a.op.RequestBody.Content.First(); pair != nil; pair = pair.Next() {
		content[pair.Key()] = convertLibopenapiSchema(pair.Value().Schema)
	}
	return RequestBodyInfo{Content: content}
}

func (a *libOASOperationAdapter) GetResponses() map[int]ResponseInfo {
	if a.op.Responses == nil || a.op.Responses.Codes == nil {
		return nil
	}
	responses := make(map[int]ResponseInfo)
	for pair := a.op.Responses.Codes.First(); pair != nil; pair = pair.Next() {
		code, err := strconv.Atoi(pair.Key())
		if err != nil {
			continue
		}
		content := make(map[string]*Schema)
		if pair.Value().Content != nil {
			for contentPair := pair.Value().Content.First(); contentPair != nil; contentPair = contentPair.Next() {
				content[contentPair.Key()] = convertLibopenapiSchema(contentPair.Value().Schema)
			}
		}
		responses[code] = ResponseInfo{Content: content}
	}
	return responses
}

// --- Conversion Utilities ---

func convertLibopenapiSchema(proxy *base.SchemaProxy) (domainSchema *Schema) {
	// Gracefully handle panics from the underlying library, which can occur with
	// invalid schemas (e.g., dangling references).
	defer func() {
		if r := recover(); r != nil {
			// On panic, ensure we return a nil schema.
			domainSchema = nil
		}
	}()

	if proxy == nil {
		return nil
	}

	s, err := proxy.BuildSchema()
	if err != nil {
		return nil // Handle expected errors, like resolution failures.
	}

	// This check is necessary because even if err is nil, the schema can be nil
	// in some edge cases (like a valid but empty proxy).
	if s == nil {
		return nil
	}

	domainSchema = &Schema{
		Type:        s.Type,
		Description: s.Description,
		Required:    s.Required,
	}

	if s.Properties != nil {
		for pair := s.Properties.First(); pair != nil; pair = pair.Next() {
			domainSchema.Properties = append(domainSchema.Properties, Property{
				Name:   pair.Key(),
				Schema: convertLibopenapiSchema(pair.Value()),
			})
		}
	}

	if s.Items != nil && s.Items.IsA() {
		domainSchema.Items = convertLibopenapiSchema(s.Items.A)
	}

	for _, allOfProxy := range s.AllOf {
		domainSchema.AllOf = append(domainSchema.AllOf, convertLibopenapiSchema(allOfProxy))
	}

	return domainSchema
}

func convertToLibopenapiSchema(schema *Schema) *base.Schema {
	if schema == nil {
		return nil
	}

	libSchema := &base.Schema{
		Type:        schema.Type,
		Description: schema.Description,
		Required:    schema.Required,
	}

	if len(schema.Properties) > 0 {
		libSchema.Properties = orderedmap.New[string, *base.SchemaProxy]()
		for _, prop := range schema.Properties {
			libSchema.Properties.Set(prop.Name, base.CreateSchemaProxy(convertToLibopenapiSchema(prop.Schema)))
		}
	}

	if schema.Items != nil {
		libSchema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{
			A: base.CreateSchemaProxy(convertToLibopenapiSchema(schema.Items)),
		}
	}

	for _, allOfSchema := range schema.AllOf {
		libSchema.AllOf = append(libSchema.AllOf, base.CreateSchemaProxy(convertToLibopenapiSchema(allOfSchema)))
	}

	return libSchema
}
