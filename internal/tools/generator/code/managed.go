package code

import (
	"os"
	"path/filepath"

	"github.com/dave/jennifer/jen"
	"github.com/matteogastaldello/swaggergen-provider/internal/tools/generator/text"
)

func GenerateManaged(workdir string, res *Resource) error {
	srcdir, err := createSourceDir(workdir, res)
	if err != nil {
		return err
	}

	g := jen.NewFile(normalizeVersion(res.Version))
	g.ImportAlias(pkgCommon, pkgCommonAlias)

	g.Add(generateConditionFuncs(res))
	g.Line()

	g.Add(generateDeletionPolicyFuncs(res))
	g.Line()

	src, err := os.Create(filepath.Join(srcdir, "managed.go"))
	if err != nil {
		return err
	}
	defer src.Close()

	return g.Render(src)
}

func generateConditionFuncs(res *Resource) jen.Code {
	kind := text.ToGolangName(res.Kind)

	getter := jen.Func().Params(jen.Id("mg").Op("*").Id(kind)).
		Id("GetCondition").Params(
		jen.Id("ct").Qual(pkgCommon, "ConditionType"),
	).Qual(pkgCommon, "Condition").Block(
		jen.Return(jen.Id("mg").Dot("Status").Dot("GetCondition").Call(jen.Id("ct"))),
	)

	setter := jen.Func().Params(jen.Id("mg").Op("*").Id(kind)).
		Id("SetConditions").Params(
		jen.Id("c").Op("...").Qual(pkgCommon, "Condition"),
	).Block(
		jen.Id("mg").Dot("Status").Dot("SetConditions").Call(jen.Id("c").Op("...")),
	)

	return getter.Line().Line().Add(setter)
}

func generateDeletionPolicyFuncs(res *Resource) jen.Code {
	kind := text.ToGolangName(res.Kind)

	getter := jen.Func().Params(jen.Id("mg").Op("*").Id(kind)).
		Id("GetDeletionPolicy").Params().
		Qual(pkgCommon, "DeletionPolicy").Block(
		jen.Return(jen.Id("mg").Dot("Spec").Dot("DeletionPolicy")),
	)

	setter := jen.Func().Params(jen.Id("mg").Op("*").Id(kind)).
		Id("SetDeletionPolicy").Params(
		jen.Id("p").Qual(pkgCommon, "DeletionPolicy"),
	).Block(
		jen.Id("mg").Dot("Spec").Dot("DeletionPolicy").Op("=").Id("p"),
	)

	return getter.Line().Line().Add(setter)
}
