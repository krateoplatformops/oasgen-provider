package templates

import (
	_ "embed"
	"fmt"
	"testing"
)

func TestDeploymentManifest(t *testing.T) {
	values := Values(Renderoptions{
		Group:     "composition.krateo.io",
		Version:   "v12-8-3",
		Resource:  "postgresqls",
		Name:      "postgres-tgz",
		Namespace: "default",
	})
	bin, err := RenderDeployment(values)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(bin))
}
