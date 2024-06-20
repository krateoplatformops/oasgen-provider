package templates

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

var (
	//go:embed assets/deployment.yaml
	deploymentTpl string
)

type Renderoptions struct {
	Group      string
	Version    string
	Resource   string
	Namespace  string
	Name       string
	Tag        string
	ClientType string
}

func Values(opts Renderoptions) map[string]string {
	if len(opts.Name) == 0 {
		opts.Name = fmt.Sprintf("%s-controller", opts.Resource)
	}

	if len(opts.Namespace) == 0 {
		opts.Namespace = "default"
	}

	return map[string]string{
		"apiGroup":   opts.Group,
		"apiVersion": opts.Version,
		"resource":   opts.Resource,
		"name":       opts.Name,
		"namespace":  opts.Namespace,
		"tag":        opts.Tag,
		"clientType": opts.ClientType,
	}
}

func RenderDeployment(values map[string]string) ([]byte, error) {
	tpl, err := template.New("deployment").Funcs(TxtFuncMap()).Parse(deploymentTpl)
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	if err := tpl.Execute(&buf, values); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
