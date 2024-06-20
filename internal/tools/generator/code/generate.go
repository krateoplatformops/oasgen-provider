package code

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/dave/jennifer/jen"
)

const (
	pkgControllerGen      = "sigs.k8s.io/controller-tools/cmd/controller-gen"
	pkgControllerGenAlias = "-"
)

func CreateGenerateDotGo(workdir string) error {
	err := os.MkdirAll(filepath.Join(workdir, "apis"), os.ModePerm)
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			return err
		}
	}

	g := jen.NewFile("apis")

	//g.HeaderComment("go:build generate")
	g.HeaderComment("+build generate")
	g.Line().Line()
	g.HeaderComment("Remove existing CRDs")
	g.HeaderComment("go:generate rm -rf ../crds")
	g.Line().Line()
	g.HeaderComment("Generate deepcopy methodsets and CRD manifests")
	g.HeaderComment("go:generate go run -tags generate sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../hack/boilerplate.go.txt paths=./... crd:crdVersions=v1 output:artifacts:config=../crds")
	g.Line()

	g.Anon(pkgControllerGen)

	src, err := os.Create(filepath.Join(workdir, "apis", "generate.go"))
	if err != nil {
		return err
	}
	defer src.Close()

	return g.Render(src)
}
