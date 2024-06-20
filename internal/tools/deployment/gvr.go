package deployment

import (
	"strings"

	"github.com/gobuffalo/flect"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// func GroupVersionResource(fs *chartfs.ChartFS) (schema.GroupVersionResource, error) {
// 	fin, err := fs.Open(fs.RootDir() + "/Chart.yaml")
// 	if err != nil {
// 		return schema.GroupVersionResource{}, err
// 	}
// 	defer fin.Close()

// 	din, err := io.ReadAll(fin)
// 	if err != nil {
// 		return schema.GroupVersionResource{}, err
// 	}

// 	res := map[string]any{}
// 	if err := yaml.Unmarshal(din, &res); err != nil {
// 		return schema.GroupVersionResource{}, err
// 	}

// 	name := res["name"].(string)
// 	kind := flect.Pascalize(text.ToGolangName(name))
// 	resource := strings.ToLower(flect.Pluralize(kind))
// 	version := fmt.Sprintf("v%s", strings.ReplaceAll(res["version"].(string), ".", "-"))

// 	return schema.GroupVersionResource{
// 		Group:    "composition.krateo.io",
// 		Version:  version,
// 		Resource: resource,
// 	}, nil
// }

func ToGroupVersionResource(gvk schema.GroupVersionKind) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(flect.Pluralize(gvk.Kind)),
	}
}
