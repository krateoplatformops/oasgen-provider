package crd

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestConversionConf(t *testing.T) {
	tests := []struct {
		name     string
		crd      apiextensionsv1.CustomResourceDefinition
		conf     *apiextensionsv1.CustomResourceConversion
		expected *apiextensionsv1.CustomResourceDefinition
	}{
		{
			name: "Set conversion configuration",
			crd: apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{},
			},
			conf: &apiextensionsv1.CustomResourceConversion{
				Strategy: apiextensionsv1.NoneConverter,
			},
			expected: &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Conversion: &apiextensionsv1.CustomResourceConversion{
						Strategy: apiextensionsv1.NoneConverter,
					},
				},
			},
		},
		{
			name: "Update conversion configuration",
			crd: apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Conversion: &apiextensionsv1.CustomResourceConversion{
						Strategy: apiextensionsv1.WebhookConverter,
					},
				},
			},
			conf: &apiextensionsv1.CustomResourceConversion{
				Strategy: apiextensionsv1.NoneConverter,
			},
			expected: &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Conversion: &apiextensionsv1.CustomResourceConversion{
						Strategy: apiextensionsv1.NoneConverter,
					},
				},
			},
		},
		{
			name: "Nil conversion configuration",
			crd: apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{},
			},
			conf: nil,
			expected: &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Conversion: nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConversionConf(tt.crd, tt.conf)
			if diff := cmp.Diff(result, tt.expected); len(diff) > 0 {
				t.Fatalf("Unexpected result (-got +want):\n%s", diff)
			}
		})
	}
}

func TestAppendVersion(t *testing.T) {
	tests := []struct {
		name     string
		crd      apiextensionsv1.CustomResourceDefinition
		toAdd    apiextensionsv1.CustomResourceDefinition
		expected apiextensionsv1.CustomResourceDefinition
	}{
		{
			name: "Append new versions",
			crd: apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1"},
					},
				},
			},
			toAdd: apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha2"},
					},
				},
			},
			expected: apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1", Served: true, Storage: false},
						{Name: "v1alpha2", Served: true, Storage: false},
						{
							Name:    "vacuum",
							Served:  false,
							Storage: true,
							Schema: &apiextensionsv1.CustomResourceValidation{
								OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
									Type:        "object",
									Description: "This is a vacuum version to storage different versions",
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"apiVersion": {
											Type:        "string",
											Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
										},
										"kind": {
											Type: "string",
										},
										"metadata": {
											Type: "object",
										},
										"spec": {
											Type:                   "object",
											XPreserveUnknownFields: &[]bool{true}[0],
										},
										"status": {
											Type:                   "object",
											XPreserveUnknownFields: &[]bool{true}[0],
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Append existing version",
			crd: apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1", Served: true, Storage: true},
					},
				},
			},
			toAdd: apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1"},
					},
				},
			},
			expected: apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1", Served: true, Storage: true},
					},
				},
			},
		},
		{
			name: "Append version with existing vacuum",
			crd: apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1", Served: true, Storage: true},
					},
				},
			},
			toAdd: apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha2"},
					},
				},
			},
			expected: apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1", Served: true, Storage: false},
						{Name: "v1alpha2", Served: true, Storage: false},
						{
							Name:    "vacuum",
							Served:  false,
							Storage: true,
							Schema: &apiextensionsv1.CustomResourceValidation{
								OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
									Type:        "object",
									Description: "This is a vacuum version to storage different versions",
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"apiVersion": {
											Type:        "string",
											Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
										},
										"kind": {
											Type: "string",
										},
										"metadata": {
											Type: "object",
										},
										"spec": {
											Type:                   "object",
											XPreserveUnknownFields: &[]bool{true}[0],
										},
										"status": {
											Type:                   "object",
											XPreserveUnknownFields: &[]bool{true}[0],
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AppendVersion(tt.crd, tt.toAdd)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			ok := assert.ElementsMatch(t, result.Spec.Versions, tt.expected.Spec.Versions)
			if !ok {
				t.Log("Result:")
				t.Log(result.Spec.Versions)

				t.Log("Expected:")
				t.Log(tt.expected.Spec.Versions)

				t.Fatalf("Slice elements do not match")
			}

			// if diff := cmp.Diff(result, &tt.expected); len(diff) > 0 {
			// 	t.Fatalf("Unexpected result (-got +want):\n%s", diff)
			// }
		})
	}
}

func TestSetServedStorage(t *testing.T) {
	tests := []struct {
		name     string
		crd      *apiextensionsv1.CustomResourceDefinition
		version  string
		served   bool
		storage  bool
		expected *apiextensionsv1.CustomResourceDefinition
	}{
		{
			name: "Set served and storage to true",
			crd: &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1"},
					},
				},
			},
			version: "v1alpha1",
			served:  true,
			storage: true,
			expected: &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1", Served: true, Storage: true},
					},
				},
			},
		},
		{
			name: "Set served and storage to false",
			crd: &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1"},
					},
				},
			},
			version: "v1alpha1",
			served:  false,
			storage: false,
			expected: &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1", Served: false, Storage: false},
					},
				},
			},
		},
		{
			name: "Version not found",
			crd: &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1"},
					},
				},
			},
			version: "v1alpha2",
			served:  true,
			storage: true,
			expected: &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetServedStorage(tt.crd, tt.version, tt.served, tt.storage)
			if diff := cmp.Diff(tt.crd, tt.expected); len(diff) > 0 {
				t.Fatalf("Unexpected result (-got +want):\n%s", diff)
			}
		})
	}
}
