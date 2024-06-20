package code

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dave/jennifer/jen"
)

const (
	pkgApiMachineryRuntime = "k8s.io/apimachinery/pkg/runtime"
)

func CreateApisDotGo(el *Resource, cfg Options) error {
	g := jen.NewFile("apis")
	g.ImportName(pkgApiMachineryRuntime, "runtime")

	alias := fmt.Sprintf("%s%s", strings.ToLower(el.Kind), normalizeVersion(el.Version))
	pkg := fmt.Sprintf("%s/apis/%s/%s", cfg.Module, strings.ToLower(el.Kind), normalizeVersion(el.Version))
	g.ImportAlias(pkg, alias)

	g.Line()

	stmts := make([]jen.Code, 2)
	stmts[0] = jen.Id("AddToSchemes")
	stmts[1] = generateAddToScheme(el, cfg)

	g.Func().Id("init").Params().Block(
		jen.Id("AddToSchemes").Op("=").Append(stmts...),
	)

	g.Line()
	g.Var().Id("AddToSchemes").Qual(pkgApiMachineryRuntime, "SchemeBuilder")
	g.Line()

	g.Func().Id("AddToScheme").Params(jen.Id("s").Op("*").Qual(pkgApiMachineryRuntime, "Scheme")).Error().Block(
		jen.Return(jen.Id("AddToSchemes").Dot("AddToScheme").Call(jen.Id("s"))),
	)

	src, err := os.Create(filepath.Join(cfg.Workdir, "apis", "apis.go"))
	if err != nil {
		return err
	}
	defer src.Close()

	return g.Render(src)
}

func generateAddToScheme(res *Resource, cfg Options) *jen.Statement {
	kind := strings.ToLower(res.Kind)
	pkg := fmt.Sprintf("%s/apis/%s/%s", cfg.Module, kind, normalizeVersion(res.Version))
	return jen.Qual(pkg, "SchemeBuilder").Dot("AddToScheme")
}
