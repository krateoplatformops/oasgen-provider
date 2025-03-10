package crd

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func ConversionConf(crd apiextensionsv1.CustomResourceDefinition, conf *apiextensionsv1.CustomResourceConversion) *apiextensionsv1.CustomResourceDefinition {
	crd.Spec.Conversion = conf
	return &crd
}

func SetServedStorage(crd *apiextensionsv1.CustomResourceDefinition, version string, served, storage bool) {
	for i := range crd.Spec.Versions {
		if crd.Spec.Versions[i].Name == version {
			crd.Spec.Versions[i].Served = served
			crd.Spec.Versions[i].Storage = storage
		}
	}
}

// AppendVersion appends the version of the toadd CRD to the crd CRD and sets the Storage and Served fields in the last version of the crd CRD.
func AppendVersion(crd apiextensionsv1.CustomResourceDefinition, toadd apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
	for _, el2 := range toadd.Spec.Versions {
		exist := false
		vacuum := false
		for _, el1 := range crd.Spec.Versions {
			if el1.Name == el2.Name {
				exist = true
				break
			}
		}
		for _, el1 := range crd.Spec.Versions {
			if el1.Name == "vacuum" {
				vacuum = true
				break
			}
		}

		if !exist {
			crd.Spec.Versions = append(crd.Spec.Versions, el2)
			if !vacuum {
				crd.Spec.Versions = append(crd.Spec.Versions, apiextensionsv1.CustomResourceDefinitionVersion{
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
				})
			}
			for i := range crd.Spec.Versions {
				// if different from vacuum served: false and storage: true
				if crd.Spec.Versions[i].Name != "vacuum" {
					crd.Spec.Versions[i].Served = true
					crd.Spec.Versions[i].Storage = false
				}
			}
		}
	}

	return &crd, nil
}

type VersionConf struct {
	Name   string
	Served bool
}
