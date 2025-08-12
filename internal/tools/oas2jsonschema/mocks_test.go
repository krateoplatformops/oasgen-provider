package oas2jsonschema

// Note: File named in this way to avoid warnings about unused imports.

// --- Mock Implementations ---

// mockOperation implements the Operation interface for testing.
type mockOperation struct {
	Parameters  []ParameterInfo
	RequestBody RequestBodyInfo
	Responses   map[int]ResponseInfo
}

func (m *mockOperation) GetParameters() []ParameterInfo     { return m.Parameters }
func (m *mockOperation) GetRequestBody() RequestBodyInfo    { return m.RequestBody }
func (m *mockOperation) GetResponses() map[int]ResponseInfo { return m.Responses }

// mockPathItem implements the PathItem interface for testing.
type mockPathItem struct {
	Ops map[string]Operation
}

func (m *mockPathItem) GetOperations() map[string]Operation { return m.Ops }

// mockOASDocument implements the OASDocument interface for testing.
type mockOASDocument struct {
	Paths           map[string]*mockPathItem
	securitySchemes []SecuritySchemeInfo
}

func (m *mockOASDocument) FindPath(path string) (PathItem, bool) {
	p, ok := m.Paths[path]
	return p, ok
}

func (m *mockOASDocument) SecuritySchemes() []SecuritySchemeInfo {
	return m.securitySchemes
}
