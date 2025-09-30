package oas2jsonschema

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/krateoplatformops/oasgen-provider/internal/tools/safety"
	"github.com/pb33f/libopenapi"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

// --- Parser Implementation ---

type libOASParser struct{}

// NewLibOASParser creates a new parser that uses the libopenapi library.
func NewLibOASParser() Parser {
	return &libOASParser{}
}

// Parse takes raw OpenAPI specification content and returns a document
// that conforms to the OASDocument interface.
func (p *libOASParser) Parse(content []byte) (OASDocument, error) {

	d, err := libopenapi.NewDocument(content)
	if err != nil {
		return nil, ParserError{
			Code:    CodeDocumentCreationError,
			Message: "failed to create new libopenapi document",
			Err:     err,
		}
	}

	doc, modelErrors := d.BuildV3Model()
	if len(modelErrors) > 0 {
		return nil, ParserError{
			Code:    CodeModelBuildError,
			Message: "failed to build V3 model",
			Err:     errors.Join(modelErrors...),
		}
	}
	if doc == nil {
		return nil, ParserError{
			Code:    CodeModelBuildError,
			Message: "resulting document was nil after building model",
		}
	}

	// Resolve model references
	// Ensures that all $ref pointers are properly resolved before further processing.
	// This is just a validation step, there is no replacement of references in the model
	resolvingErrors := doc.Index.GetResolver().Resolve()
	if len(resolvingErrors) > 0 {
		var errs []error
		for i := range resolvingErrors {
			errs = append(errs, resolvingErrors[i].ErrorRef)
		}
		return nil, ParserError{
			Code:    CodeModelResolutionError,
			Message: "failed to resolve model references",
			Err:     errors.Join(errs...),
		}
	}

	return NewLibOASDocumentAdapter(doc), nil
}

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
			Required:    p.Required != nil && *p.Required,
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

// convertLibopenapiSchema is the entry point for schema conversion. It sets up a
// map to track visited schema proxies to prevent infinite recursion in the case
// of circular references.
func convertLibopenapiSchema(proxy *base.SchemaProxy) *Schema {

	guard := safety.NewRecursionGuard(50, 5000, 30*time.Second)
	ctx, cancel := guard.WithContext()
	defer cancel()

	return convertLibopenapiSchemaWithVisited(ctx, proxy, guard, make(map[string]*Schema), 0)
}

// convertLibopenapiSchemaWithVisited is the recursive implementation of the schema conversion.
// It uses the 'visited' map to detect and handle circular references.
func convertLibopenapiSchemaWithVisited(
	ctx context.Context,
	proxy *base.SchemaProxy,
	guard *safety.RecursionGuard,
	visited map[string]*Schema,
	depth int,
) *Schema {

	if proxy == nil {
		return nil
	}

	if err := guard.Check(ctx, depth); err != nil {
		log.Printf("schema recursion aborted: %v", err)
		return &Schema{Type: []string{"object"}}
	}

	// Use the reference pointer string as the unique key for cycle detection,
	// but only if the proxy is actually a reference.
	if proxy.IsReference() {
		ref := proxy.GetReference()
		if ref != "" {
			if existingSchema, ok := visited[ref]; ok {
				log.Printf("Circular reference detected at depth %d for ref '%s'. Returning placeholder schema to break the loop.", depth, ref)
				return existingSchema
			}
		}
	}

	// Create a new schema placeholder and add it to the visited map before building or processing the schema.
	// This is used to break the recursion.
	// domainSchema is the schema defined in this domain (oas2jsonschema)
	domainSchema := &Schema{}
	if proxy.IsReference() {
		ref := proxy.GetReference()
		if ref != "" {
			visited[ref] = domainSchema
		}
	}

	// Gracefully handle panics from the underlying library, which can occur with invalid schemas (e.g., dangling references).
	defer func() {
		if r := recover(); r != nil {
			// Log the panic for debugging
			log.Printf("Schema conversion panic: %v", r)
		}
	}()

	s, err := proxy.BuildSchema() // This step resolves the reference if it's a $ref
	if err != nil {
		log.Printf("Schema build error: %v", err)
		return domainSchema // Return the placeholder to avoid breaking parent
	}

	if s == nil {
		return domainSchema
	}

	// Default handling
	var defaultVal interface{}
	if s.Default != nil {
		if err := s.Default.Decode(&defaultVal); err != nil {
			log.Printf("Failed to decode default value: %v", err)
		}
	}

	// Populate the placeholder schema we created earlier by modifying its fields directly.
	// Do not reassign domainSchema as that would break the circular reference handling
	domainSchema.Type = s.Type
	domainSchema.Description = s.Description
	domainSchema.Required = s.Required
	domainSchema.Default = defaultVal

	// Enum handling
	var enumValues []interface{}
	if len(s.Enum) > 0 {
		for _, enumProxy := range s.Enum {
			var enumVal interface{}
			if err := enumProxy.Decode(&enumVal); err != nil {
				log.Printf("Failed to decode enum value: %v (raw: %#v)", err, enumProxy)
				continue
			}
			enumValues = append(enumValues, enumVal)
		}
		domainSchema.Enum = enumValues
	}

	// AdditionalProperties handling
	if s.AdditionalProperties != nil {
		switch {
		case s.AdditionalProperties.IsB():
			// Boolean form: allows or disallows any additional properties
			domainSchema.AdditionalProperties = s.AdditionalProperties.B
		case s.AdditionalProperties.IsA():
			// Schema form: recurse to handle nested schemas properly
			// Currently not handled in `oasgen-provider`, ignored
			//log.Print("Warning: Schema form of AdditionalProperties is not currently handled")
		default:
			//log.Print("Warning: Unknown AdditionalProperties type")
		}
	}

	// MaxProperties handling
	// as per JSON Schema spec (any non-negative integer).
	if s.MaxProperties != nil {
		domainSchema.MaxProperties = int(*s.MaxProperties)
	}

	// Format handling
	// If a format is specified, append it to the description for additional context.
	// There is no format validation
	if s.Format != "" {
		domainSchema.Description = appendFormatToDescription(domainSchema.Description, s.Format)
	}

	// Properties handling
	if s.Properties != nil {
		domainSchema.Properties = make([]Property, 0, s.Properties.Len())
		for pair := s.Properties.First(); pair != nil; pair = pair.Next() {
			domainSchema.Properties = append(domainSchema.Properties, Property{
				Name:   pair.Key(),
				Schema: convertLibopenapiSchemaWithVisited(ctx, pair.Value(), guard, visited, depth+1),
			})
		}
	}

	// Items handling (OAS 3.0 / 3.1)
	if s.Items != nil {
		switch {
		case s.Items.IsA():
			// Single schema (OAS 3.0 style or OAS 3.1 single schema)
			domainSchema.Items = convertLibopenapiSchemaWithVisited(ctx, s.Items.A, guard, visited, depth+1)

		case s.Items.IsB():
			// OAS 3.1 tuple validation: array of schemas
			// Currently not handled in `oasgen-provider`
			log.Print("Warning: array of schemas in 'items' is not currently handled")
		default:
			log.Print("Warning: Unknown 'items' type")
		}
	}

	// AllOf handling
	// At this time we only handle the recursive conversion of allOf schemas.
	// We do not merge properties or other attributes from allOf into the parent schema.
	// That will be handled later in the processing pipeline (in helpers.go).
	if len(s.AllOf) > 0 {
		domainSchema.AllOf = make([]*Schema, 0, len(s.AllOf))
		for _, allOfProxy := range s.AllOf {
			domainSchema.AllOf = append(domainSchema.AllOf, convertLibopenapiSchemaWithVisited(ctx, allOfProxy, guard, visited, depth+1))
		}
	}

	return domainSchema
}

// appendFormatToDescription appends the format information to the description string.
// If the format is empty, it returns the original description unchanged.
// This is a simple utility to enhance schema descriptions with format details.
func appendFormatToDescription(description, format string) string {
	if format == "" {
		return description
	}
	return fmt.Sprintf("%s (format: %s)", description, format)
}

// Note: function currently not used
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
