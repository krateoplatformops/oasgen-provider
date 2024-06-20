package code

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dave/jennifer/jen"
	"github.com/matteogastaldello/swaggergen-provider/internal/tools/generator/text"
)

const (
	pkgRuntimeSchema                = "k8s.io/apimachinery/pkg/runtime/schema"
	pkgRuntimeSchemaAlias           = "runtimeschema"
	pkgControllerRuntimeScheme      = "sigs.k8s.io/controller-runtime/pkg/scheme"
	pkgControllerRuntimeSchemeAlias = "scheme"
)

func CreateGroupVersionInfoDotGo(workdir string, res *Resource) error {
	srcdir, err := createSourceDir(workdir, res)
	if err != nil {
		return err
	}

	g := jen.NewFile(normalizeVersion(res.Version))
	g.PackageComment("+kubebuilder:object:generate=true")
	g.PackageComment(fmt.Sprintf("+groupName=%s", res.Group))
	g.PackageComment(fmt.Sprintf("+versionName=%s", res.Version))

	g.ImportAlias(pkgRuntimeSchema, pkgRuntimeSchemaAlias)
	g.ImportAlias(pkgControllerRuntimeScheme, pkgControllerRuntimeSchemeAlias)

	g.Add(generateConsts(res))
	g.Add(jen.Line())

	g.Add(generateVars(res))
	g.Add(jen.Line())

	g.Add(generateInitFunc(res))
	g.Add(jen.Line())

	src, err := os.Create(filepath.Join(srcdir, "groupversion_info.go"))
	if err != nil {
		return err
	}
	defer src.Close()

	return g.Render(src)
}

func generateConsts(res *Resource) jen.Code {
	return jen.Const().Defs(
		jen.Id("Group").Op("=").Lit(res.Group),
		jen.Id("Version").Op("=").Lit(res.Version),
	)
}

func generateVars(res *Resource) jen.Code {
	kind := text.ToGolangName(res.Kind)

	code := jen.Var().Defs(
		jen.Id("SchemeGroupVersion").Op("=").Qual(pkgRuntimeSchema, "GroupVersion").Values(
			jen.Dict{
				jen.Id("Group"):   jen.Id("Group"),
				jen.Id("Version"): jen.Id("Version"),
			},
		),

		jen.Id("SchemeBuilder").Op("=").Op("&").Qual(pkgControllerRuntimeScheme, "Builder").Values(
			jen.Dict{
				jen.Id("GroupVersion"): jen.Id("SchemeGroupVersion"),
			},
		),
	)

	code.Line().Line()

	code.Var().Defs(
		jen.Id(fmt.Sprintf("%sKind", kind)).
			Op("=").
			Qual("reflect", "TypeOf").
			Parens(jen.Id(kind).Values(jen.Dict{})).
			Dot("Name").Call(),

		jen.Id(fmt.Sprintf("%sGroupKind", kind)).
			Op("=").
			Qual(pkgRuntimeSchema, "GroupKind").Values(
			jen.Dict{
				jen.Id("Group"): jen.Id("Group"),
				jen.Id("Kind"):  jen.Id(fmt.Sprintf("%sKind", kind)),
			},
		).Dot("String").Call(),

		jen.Id(fmt.Sprintf("%sKindAPIVersion", kind)).
			Op("=").
			Id(fmt.Sprintf("%sKind", kind)).
			Op("+").Lit(".").Op("+").
			Id("SchemeGroupVersion").Dot("String").Call(),

		jen.Id(fmt.Sprintf("%sGroupVersionKind", kind)).
			Op("=").
			Id("SchemeGroupVersion").Dot("WithKind").
			Call(jen.Id(fmt.Sprintf("%sKind", kind))),
	)

	return code
}

func generateInitFunc(res *Resource) jen.Code {
	kind := text.ToGolangName(res.Kind)

	return jen.Func().Id("init").Params().Block(
		jen.Id("SchemeBuilder").Dot("Register").Call(
			jen.Op("&").Id(kind).Values(jen.Dict{}),
			jen.Op("&").Id(fmt.Sprintf("%sList", kind)).Values(jen.Dict{}),
		),
	)
}
