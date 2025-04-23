package objects

import (
	"fmt"
	"os"

	"github.com/krateoplatformops/oasgen-provider/internal/tools/objects/templates"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
)

func CreateK8sObject(obj runtime.Object, gvr schema.GroupVersionResource, nn types.NamespacedName, path string, additionalvalues ...any) error {
	templateF, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read object template file: %w", err)
	}

	values := templates.Values(templates.Renderoptions{
		Group:     gvr.Group,
		Version:   gvr.Version,
		Resource:  gvr.Resource,
		Namespace: nn.Namespace,
		Name:      nn.Name,
	})

	if len(additionalvalues)%2 != 0 {
		return fmt.Errorf("additionalvalues must be in pairs: %w", err)
	}
	for i := 0; i < len(additionalvalues); i += 2 {
		key, ok := additionalvalues[i].(string)
		if !ok {
			return fmt.Errorf("additionalvalues key must be a string: %w", err)
		}
		values[key] = additionalvalues[i+1]
	}

	template := templates.Template(string(templateF))
	dat, err := template.Render(values)
	if err != nil {
		return err
	}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory,
		clientsetscheme.Scheme,
		clientsetscheme.Scheme)

	_, _, err = s.Decode(dat, nil, obj)
	if err != nil {
		return fmt.Errorf("failed to decode object: %w", err)
	}
	return nil
}
