package oas2jsonschema

//import (
//	"encoding/json"
//	"testing"
//
//	"github.com/google/go-cmp/cmp"
//)
//
//// TODO: to be dismissed
//
//func TestAnnotateSchemas(t *testing.T) {
//	tests := []struct {
//		name          string
//		specInput     []byte
//		statusInput   []byte
//		annotationKey string
//		wantSpec      map[string]interface{}
//		wantStatus    map[string]interface{}
//	}{
//		{
//			name: "annotates repeated fields across spec and status",
//			specInput: []byte(`{
//				"metadata": {
//					"type": "object",
//					"properties": {
//						"links": {
//							"type": "object"
//						}
//					}
//				}
//			}`),
//			statusInput: []byte(`{
//				"status": {
//					"type": "object",
//					"properties": {
//						"links": { "type": "object" }
//					}
//				}
//			}`),
//			annotationKey: "x-crdgen-identifier-name",
//			wantSpec: map[string]interface{}{
//				"metadata": map[string]interface{}{
//					"type": "object",
//					"properties": map[string]interface{}{
//						"links": map[string]interface{}{
//							"type":                     "object",
//							"x-crdgen-identifier-name": "Links1",
//						},
//					},
//				},
//			},
//			wantStatus: map[string]interface{}{
//				"status": map[string]interface{}{
//					"type": "object",
//					"properties": map[string]interface{}{
//						"links": map[string]interface{}{
//							"type":                     "object",
//							"x-crdgen-identifier-name": "Links2",
//						},
//					},
//				},
//			},
//		},
//		{
//			name:      "underscore and not underscore names",
//			specInput: []byte(`{}`),
//			statusInput: []byte(`{
//				"properties": {
//					"closedBy": {
//						"properties": {
//							"_links": {
//								"description": "The class to represent a collection of REST reference links.",
//								"properties": {
//									"links": {
//										"description": "The readonly view of the links. Because Reference links are readonly, we only want to expose them as read only.",
//										"type": "object"
//									}
//								},
//								"type": "object"
//							},
//							"descriptor": {
//								"description": "The descriptor is the primary way to reference the graph subject while the system is running. This field will uniquely identify the same graph subject across both Accounts and Organizations.",
//								"type": "string"
//							},
//							"directoryAlias": {
//          						"description": "Deprecated - Can be retrieved by querying the Graph user referenced in the \"self\" entry of the IdentityRef \"_links\" dictionary",
//          						"type": "string"
//        					}
//						},
//						"type": "object"
//					}
//				},
//				"type": "object"
//			}`),
//			annotationKey: "x-crdgen-identifier-name",
//			wantSpec:      map[string]interface{}{},
//			wantStatus: map[string]interface{}{
//				"properties": map[string]interface{}{
//					"closedBy": map[string]interface{}{
//						"properties": map[string]interface{}{
//							"_links": map[string]interface{}{
//								"description": "The class to represent a collection of REST reference links.",
//								"properties": map[string]interface{}{
//									"links": map[string]interface{}{
//										"description":              "The readonly view of the links. Because Reference links are readonly, we only want to expose them as read only.",
//										"type":                     "object",
//										"x-crdgen-identifier-name": "Links2",
//									},
//								},
//								"type":                     "object",
//								"x-crdgen-identifier-name": "Links1",
//							},
//							"descriptor": map[string]interface{}{
//								"description": "The descriptor is the primary way to reference the graph subject while the system is running. This field will uniquely identify the same graph subject across both Accounts and Organizations.",
//								"type":        "string",
//							},
//							"directoryAlias": map[string]interface{}{
//								"description": "Deprecated - Can be retrieved by querying the Graph user referenced in the \"self\" entry of the IdentityRef \"_links\" dictionary",
//								"type":        "string",
//							},
//						},
//						"type": "object",
//					},
//				},
//				"type": "object",
//			},
//		},
//		{
//			name: "does not annotate unique fields",
//			specInput: []byte(`{
//				"user": { "type": "object", "properties": { "name": { "type": "string" } } }
//			}`),
//			statusInput: []byte(`{
//				"status": { "type": "object", "properties": { "active": { "type": "boolean" } } }
//			}`),
//			annotationKey: "x-crdgen-identifier-name",
//			wantSpec: map[string]interface{}{
//				"user": map[string]interface{}{
//					"type": "object",
//					"properties": map[string]interface{}{
//						"name": map[string]interface{}{"type": "string"},
//					},
//				},
//			},
//			wantStatus: map[string]interface{}{
//				"status": map[string]interface{}{
//					"type": "object",
//					"properties": map[string]interface{}{
//						"active": map[string]interface{}{"type": "boolean"},
//					},
//				},
//			},
//		},
//		{
//			name:      "handles nil spec safely",
//			specInput: nil,
//			statusInput: []byte(`{
//				"status": {
//					"type": "object",
//					"properties": {
//						"links": {
//							"type": "object"
//						},
//						"details": {
//							"type": "object",
//							"properties": {
//								"links": {
//									"type": "object"
//								}
//							}
//						}
//					}
//				}
//			}`),
//			annotationKey: "x-crdgen-identifier-name",
//			wantSpec:      nil,
//			wantStatus: map[string]interface{}{
//				"status": map[string]interface{}{
//					"type": "object",
//					"properties": map[string]interface{}{
//						"links": map[string]interface{}{
//							"type":                     "object",
//							"x-crdgen-identifier-name": "Links1",
//						},
//						"details": map[string]interface{}{
//							"type": "object",
//							"properties": map[string]interface{}{
//								"links": map[string]interface{}{
//									"type":                     "object",
//									"x-crdgen-identifier-name": "Links2",
//								},
//							},
//						},
//					},
//				},
//			},
//		},
//		{
//			name: "handles nil status safely",
//			specInput: []byte(`{
//				"properties": {
//					"config": {
//						"type": "object",
//						"properties": {
//							"enabled": {
//								"type": "boolean"
//							}
//						}
//					},
//					"enabled": {
//						"type": "boolean"
//					}
//				},
//				"type": "object"
//			}`),
//			statusInput:   nil,
//			annotationKey: "x-crdgen-identifier-name",
//			wantSpec: map[string]interface{}{
//				"properties": map[string]interface{}{
//					"config": map[string]interface{}{
//						"type": "object",
//						"properties": map[string]interface{}{
//							"enabled": map[string]interface{}{
//								"type": "boolean",
//							},
//						},
//					},
//					"enabled": map[string]interface{}{
//						"type": "boolean",
//					},
//				},
//				"type": "object",
//			},
//			wantStatus: nil,
//		},
//		{
//			name: "handles longer snake_case",
//			specInput: []byte(`{
//				"properties": {
//					"firstName": { "type": "string" },
//					"last_name": { "type": "string" },
//					"Address": { "type": "object" },
//					"address": { "type": "object" },
//					"phoneNumber": { "type": "string" },
//					"phone_number_long_form": {
//						"type": "object",
//						"properties": {
//							"phone_number_long_form": {
//								"type": "object"
//							}
//						}
//					}
//				},
//				"type": "object"
//			}`),
//			statusInput:   nil,
//			annotationKey: "x-crdgen-identifier-name",
//			wantSpec: map[string]interface{}{
//				"properties": map[string]interface{}{
//					"firstName": map[string]interface{}{
//						"type": "string",
//					},
//					"last_name": map[string]interface{}{
//						"type": "string",
//					},
//					"Address": map[string]interface{}{
//						"type":                     "object",
//						"x-crdgen-identifier-name": "Address1",
//					},
//					"address": map[string]interface{}{
//						"type":                     "object",
//						"x-crdgen-identifier-name": "Address2",
//					},
//					"phoneNumber": map[string]interface{}{
//						"type": "string",
//					},
//					"phone_number_long_form": map[string]interface{}{
//						"type":                     "object",
//						"x-crdgen-identifier-name": "PhoneNumberLongForm1",
//						"properties": map[string]interface{}{
//							"phone_number_long_form": map[string]interface{}{
//								"type":                     "object",
//								"x-crdgen-identifier-name": "PhoneNumberLongForm2",
//							},
//						},
//					},
//				},
//				"type": "object",
//			},
//			wantStatus: nil,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			annotatedSpec, annotatedStatus, err := annotateSchemas(tt.specInput, tt.statusInput, tt.annotationKey)
//			if err != nil {
//				t.Fatalf("annotateSchemas() returned unexpected error: %v", err)
//			}
//
//			var specGot, statusGot map[string]interface{}
//			if len(annotatedSpec) == 0 {
//				specGot = nil
//			} else {
//				if err := json.Unmarshal(annotatedSpec, &specGot); err != nil {
//					t.Fatalf("failed to unmarshal spec output: %v", err)
//				}
//			}
//			if len(annotatedStatus) == 0 {
//				statusGot = nil
//			} else {
//				if err := json.Unmarshal(annotatedStatus, &statusGot); err != nil {
//					t.Fatalf("failed to unmarshal status output: %v", err)
//				}
//			}
//
//			if diff := cmp.Diff(tt.wantSpec, specGot); diff != "" {
//				t.Errorf("spec mismatch (-want +got):\n%s", diff)
//			}
//			if diff := cmp.Diff(tt.wantStatus, statusGot); diff != "" {
//				t.Errorf("status mismatch (-want +got):\n%s", diff)
//			}
//		})
//	}
//}
//
//func TestCountNames(t *testing.T) {
//	tests := []struct {
//		name           string
//		input          string
//		expectedCounts map[string]int
//	}{
//		{
//			name: "counts field names correctly",
//			input: `
//			{
//				"properties": {
//					"firstName": "Alice",
//					"lastName": "Smith",
//					"address": {
//						"type": "object",
//						"properties": {
//							"street": "123 Main St",
//							"city": "Metropolis"
//						}
//					},
//					"contacts": {
//						"type": "array",
//						"items": {
//							"description": "Contact information",
//							"properties": {
//								"email": {"type": "string" },
//								"phone": {"type": "string" },
//								"address": {
//									"type": "object",
//									"properties": {
//										"street": "456 Side St",
//										"city": "Gotham"
//									}
//								}
//							}
//						}
//					},
//					"type": "admin",
//					"value": "42",
//					"snake_case_with_number123": "test",
//					"FieldAlreadyPascal": "value"
//					}
//			}`,
//			expectedCounts: map[string]int{
//				"FirstName":              1,
//				"LastName":               1,
//				"Address":                2,
//				"Street":                 2,
//				"City":                   2,
//				"Contacts":               1,
//				"Email":                  1,
//				"Phone":                  1,
//				"Type":                   1,
//				"Value":                  1,
//				"SnakeCaseWithNumber123": 1,
//				"FieldAlreadyPascal":     1,
//			},
//		},
//		{
//			name:           "handles empty input",
//			input:          `{}`,
//			expectedCounts: map[string]int{},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			var data interface{}
//			if err := json.Unmarshal([]byte(tt.input), &data); err != nil {
//				t.Fatalf("invalid JSON input: %v", err)
//			}
//
//			counts := make(map[string]int)
//			countNames(data, counts, "")
//
//			if diff := cmp.Diff(tt.expectedCounts, counts); diff != "" {
//				t.Errorf("unexpected name counts (-want +got):\n%s", diff)
//			}
//		})
//	}
//}
