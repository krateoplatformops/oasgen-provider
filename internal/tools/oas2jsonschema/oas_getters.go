package oas2jsonschema

import "github.com/krateoplatformops/crdgen"

// OASSpecJsonSchemaGetter returns a JsonSchemaGetter for the spec schema.
func (r *GenerationResult) OASSpecJsonSchemaGetter() crdgen.JsonSchemaGetter {
	return &oasSpecJsonSchemaGetter{
		result: r,
	}
}

var _ crdgen.JsonSchemaGetter = (*oasSpecJsonSchemaGetter)(nil)

type oasSpecJsonSchemaGetter struct {
	result *GenerationResult
}

func (a *oasSpecJsonSchemaGetter) Get() ([]byte, error) {
	return a.result.SpecSchema, nil
}

// OASStatusJsonSchemaGetter returns a JsonSchemaGetter for the status schema.
func (r *GenerationResult) OASStatusJsonSchemaGetter() crdgen.JsonSchemaGetter {
	return &oasStatusJsonSchemaGetter{
		result: r,
	}
}

var _ crdgen.JsonSchemaGetter = (*oasStatusJsonSchemaGetter)(nil)

type oasStatusJsonSchemaGetter struct {
	result *GenerationResult
}

func (a *oasStatusJsonSchemaGetter) Get() ([]byte, error) {
	return a.result.StatusSchema, nil
}

// OASAuthCRDSchemaGetter returns a JsonSchemaGetter for a specific auth schema.
func (r *GenerationResult) OASAuthCRDSchemaGetter(secSchemaName string) crdgen.JsonSchemaGetter {
	return &oasAuthCRDSchemaGetter{
		result:        r,
		secSchemaName: secSchemaName,
	}
}

var _ crdgen.JsonSchemaGetter = (*oasAuthCRDSchemaGetter)(nil)

type oasAuthCRDSchemaGetter struct {
	result        *GenerationResult
	secSchemaName string
}

func (a *oasAuthCRDSchemaGetter) Get() ([]byte, error) {
	return a.result.AuthCRDSchemas[a.secSchemaName], nil
}

// StaticJsonSchemaGetter returns a getter that returns nil.
func StaticJsonSchemaGetter() crdgen.JsonSchemaGetter {
	return &staticJsonSchemaGetter{}
}

var _ crdgen.JsonSchemaGetter = (*staticJsonSchemaGetter)(nil)

type staticJsonSchemaGetter struct{}

func (f *staticJsonSchemaGetter) Get() ([]byte, error) {
	return nil, nil
}
