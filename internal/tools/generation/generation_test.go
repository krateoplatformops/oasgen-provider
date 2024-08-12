package generation

import (
	"fmt"
	"reflect"
	"testing"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

func TestGenerateAuthSchemaFromSecuritySchema(t *testing.T) {
	testCases := []struct {
		name     string
		doc      *v3.SecurityScheme
		expected []byte
		err      error
	}{
		{
			name: "BasicAuth",
			doc: &v3.SecurityScheme{
				Type:   "http",
				Scheme: "basic",
			},
			expected: []byte(`{"properties":{"passwordRef":{"properties":{"key":{"type":"string"},"name":{"type":"string"},"namespace":{"type":"string"}},"required":["key","name","namespace"],"type":"object"},"username":{"type":"string"}},"required":["username","passwordRef"],"type":"object"}`),
			err:      nil,
		},
		{
			name: "BearerAuth",
			doc: &v3.SecurityScheme{
				Type:   "http",
				Scheme: "bearer",
			},
			expected: []byte(`{"properties":{"tokenRef":{"properties":{"key":{"type":"string"},"name":{"type":"string"},"namespace":{"type":"string"}},"required":["key","name","namespace"],"type":"object"}},"required":["tokenRef"],"type":"object"}`),
			err:      nil,
		},
		{
			name: "InvalidAuthSchema",
			doc: &v3.SecurityScheme{
				Type:   "http",
				Scheme: "invalid",
			},
			expected: nil,
			err:      fmt.Errorf(ErrInvalidSecuritySchema),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			byteSchema, err := GenerateAuthSchemaFromSecuritySchema(tc.doc)
			if !reflect.DeepEqual(byteSchema, tc.expected) {
				t.Errorf("Expected byte schema: %v, got: %v", string(tc.expected), string(byteSchema))
			}

			if err != nil && tc.err != nil {
				if err.Error() != tc.err.Error() {
					t.Errorf("Expected error: %v, got: %v", tc.err, err)
				}
			}
			if err == nil && tc.err != nil {
				t.Errorf("Expected error: %v, got nil", tc.err)
			}
			if err != nil && tc.err == nil {
				t.Errorf("Expected error: nil, got: %v", err)
			}

		})
	}
}
