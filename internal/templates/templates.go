package templates

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

type Renderoptions struct {
	Group     string
	Version   string
	Resource  string
	Namespace string
	Name      string
}

func Values(opts Renderoptions) map[string]any {
	if len(opts.Name) == 0 {
		opts.Name = fmt.Sprintf("%s-controller", opts.Resource)
	}

	if len(opts.Namespace) == 0 {
		opts.Namespace = "default"
	}

	values := map[string]any{
		"apiGroup":   opts.Group,
		"apiVersion": opts.Version,
		"resource":   opts.Resource,
		"name":       opts.Name,
		"namespace":  opts.Namespace,
	}

	return values
}

type Template string

func (t Template) Render(values map[string]any) ([]byte, error) {
	tpl, err := template.New("template").Funcs(sprig.FuncMap()).Parse(string(t))
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	if err := tpl.Execute(&buf, values); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
