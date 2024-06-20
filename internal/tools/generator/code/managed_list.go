package code

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dave/jennifer/jen"
	"github.com/matteogastaldello/swaggergen-provider/internal/tools/generator/text"
)

const (
	pkgResource      = "github.com/krateoplatformops/provider-runtime/pkg/resource"
	pkgResourceAlias = "resource"
)

func GenerateManagedList(workdir string, res *Resource) error {
	srcdir, err := createSourceDir(workdir, res)
	if err != nil {
		return err
	}

	kind := text.ToGolangName(res.Kind)

	g := jen.NewFile(normalizeVersion(res.Version))
	g.ImportAlias(pkgResource, pkgResourceAlias)

	g.Add(
		jen.Func().Params(jen.Id("ml").Op("*").Id(fmt.Sprintf("%sList", kind))).
			Id("GetItems").Params().Index().Qual(pkgResource, "Managed").
			Block(
				jen.Id("items").Op(":=").Make(
					jen.Index().Qual(pkgResource, "Managed"),
					jen.Len(jen.Id("ml").Dot("Items")),
				),

				jen.For(
					jen.Id("i").Op(":=").Range().Id("ml").Dot("Items"),
				).Block(
					jen.Id("items").Index(jen.Id("i")).Op("=").Op("&").Id("ml").Dot("Items").Index(jen.Id("i")),
				),

				jen.Return(jen.Id("items")),
			),
	)
	g.Line()

	src, err := os.Create(filepath.Join(srcdir, "managed_list.go"))
	if err != nil {
		return err
	}
	defer src.Close()

	return g.Render(src)
}
