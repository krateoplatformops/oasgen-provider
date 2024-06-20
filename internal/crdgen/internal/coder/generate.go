package coder

import (
	"io"
	"os"
	"path/filepath"

	"github.com/dave/jennifer/jen"
)

const (
	pkgControllerGen = "sigs.k8s.io/controller-tools/cmd/controller-gen"
)

func Generate(wri io.Writer) error {
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

	return g.Render(wri)
}

func CreateGenerateDotGo(workdir string) error {
	path, err := makeDirs(workdir, "apis")
	if err != nil {
		return err
	}

	src, err := os.Create(filepath.Join(path, "generate.go"))
	if err != nil {
		return err
	}
	defer src.Close()

	return Generate(src)
}
