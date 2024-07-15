package coder

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/krateoplatformops/oasgen-provider/internal/crdgen/internal/ptr"
	"github.com/krateoplatformops/oasgen-provider/internal/crdgen/internal/strutil"
	"github.com/krateoplatformops/oasgen-provider/internal/crdgen/internal/transpiler"
)

const (
	pkgCommon           = "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	pkgCommonAlias      = "rtv1"
	pkgMeta             = "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgMetaAlias        = "metav1"
	pkgSpecCommentFmt   = "%s defines the desired state of %s"
	pkgStatusCommentFmt = "%s defines the observed state of %s"
)

func CreateTypesDotGo(workdir string, res *Resource) error {
	path, err := makeDirs(workdir, "apis",
		strings.ToLower(res.Kind), normalizeVersion(res.Version))
	if err != nil {
		return err
	}

	spec, err := jsonschemaToStruct(bytes.NewReader(res.SpecSchema))
	if err != nil {
		return err
	}

	kind := strutil.ToGolangName(res.Kind)

	g := jen.NewFile(normalizeVersion(res.Version))
	g.ImportAlias(pkgCommon, pkgCommonAlias)
	g.ImportAlias(pkgMeta, pkgMetaAlias)

	for k, v := range spec {
		g.Add(renderSpec(kind, k, v, res.Managed))
	}

	g.Add(jen.Line())

	hasStatus := len(res.StatusSchema) > 0
	if hasStatus {
		status, err := jsonschemaToStruct(bytes.NewReader(res.StatusSchema))
		if err != nil {
			return err
		}

		g.Add(createFailedObjectRef())
		g.Add(jen.Line())

		for k, v := range status {
			g.Add(renderStatus(kind, k, v, res.Managed))
		}
	}

	g.Add(jen.Comment("+kubebuilder:object:root=true"))

	if hasStatus {
		g.Add(jen.Comment("+kubebuilder:subresource:status"))
	}

	if len(res.Categories) > 0 {
		g.Add(jen.Comment(
			fmt.Sprintf("+kubebuilder:resource:scope=Namespaced,categories={%s}",
				strings.Join(res.Categories, ","))))
	} else {
		g.Add(jen.Comment("+kubebuilder:resource:scope=Namespaced"))
	}

	if hasStatus {
		g.Add(jen.Comment(`+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"`))
		g.Add(jen.Comment(`+kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"`).Line())
	}

	g.Add(jen.Line())

	fields := []jen.Code{
		jen.Qual(pkgMeta, "TypeMeta").Tag(map[string]string{"json": ",inline"}),
		jen.Qual(pkgMeta, "ObjectMeta").Tag(map[string]string{"json": ",inline"}),
		jen.Line(),
		jen.Id("Spec").Id(fmt.Sprintf("%sSpec", kind)).Tag(map[string]string{"json": "spec,omitempty"}),
	}

	if hasStatus {
		fields = append(fields,
			jen.Id("Status").Id(fmt.Sprintf("%sStatus", kind)).
				Tag(map[string]string{"json": "status,omitempty"}),
		)
	}

	g.Add(jen.Type().Id(kind).Struct(fields...).Line())

	g.Add(jen.Comment("+kubebuilder:object:root=true"))
	g.Add(jen.Line())

	g.Add(jen.Type().Id(fmt.Sprintf("%sList", kind)).Struct(
		jen.Qual(pkgMeta, "TypeMeta").Tag(map[string]string{"json": ",inline"}),
		jen.Qual(pkgMeta, "ListMeta").Tag(map[string]string{"json": "metadata,omitempty"}),
		jen.Line(),
		jen.Id("Items").Id(fmt.Sprintf("[]%s", kind)).Tag(map[string]string{"json": "items"}),
	).Line())

	src, err := os.Create(filepath.Join(path, "types.go"))
	if err != nil {
		return err
	}
	defer src.Close()

	return g.Render(src)
}

func renderSpec(kind, key string, el transpiler.Struct, managed bool) jen.Code {
	fields := []jen.Code{}

	if key == "Root" {
		key = strutil.ToGolangName(fmt.Sprintf("%sSpec", kind))
		if managed {
			fields = append(fields,
				jen.Qual(pkgCommon, "ManagedSpec").
					Tag(map[string]string{
						"json": ",inline",
					}).Line())
		}
	}

	for _, f := range el.Fields {
		fields = append(fields, renderField(f))
	}

	return jen.Type().Id(key).Struct(fields...).Line()
}

func renderField(el transpiler.Field) jen.Code {
	defValCmt := func(typ string, val any) string {
		switch in := val.(type) {
		case string:
			if typ == "string" {
				return fmt.Sprintf("+kubebuilder:default:=%q", in)
			}
			return fmt.Sprintf("+kubebuilder:default:=%v", in)
		case []byte:
			return fmt.Sprintf("+kubebuilder:default:=%q", string(in))
		case fmt.Stringer:
			return fmt.Sprintf("+kubebuilder:default:=%q", in)
		default:
			return fmt.Sprintf("+kubebuilder:default:=%v", in)
		}
	}

	res := &jen.Statement{}
	if len(el.Description) > 0 {
		cmt := fmt.Sprintf("%s: %s", el.Name, el.Description)
		res.Add(jen.Comment(cmt).Line())
	}

	if val := el.Default; val != nil {
		res.Add(jen.Comment(defValCmt(el.Type, val)).Line())
	}

	if el.Minimum != nil {
		val := ptr.Deref(el.Minimum, 0)
		cmt := fmt.Sprintf("+kubebuilder:validation:Minimum:=%d", int(val))
		res.Add(jen.Comment(cmt).Line())
	}

	if el.Maximum != nil {
		val := ptr.Deref(el.Maximum, 0)
		cmt := fmt.Sprintf("+kubebuilder:validation:Maximum:=%d", int(val))
		res.Add(jen.Comment(cmt).Line())
	}

	if el.MultipleOf != nil {
		val := ptr.Deref(el.MultipleOf, 0)
		cmt := fmt.Sprintf("+kubebuilder:validation:MultipleOf:=%d", int(val))
		res.Add(jen.Comment(cmt).Line())
	}

	if el.Pattern != nil {
		cmt := fmt.Sprintf("+kubebuilder:validation:Pattern:=`%s`", ptr.Deref(el.Pattern, ""))
		res.Add(jen.Comment(cmt).Line())
	}

	if len(el.Enum) > 0 {
		cmt := fmt.Sprintf("+kubebuilder:validation:Enum:=%s", strings.Join(el.Enum, ";"))
		res.Add(jen.Comment(cmt).Line())
	}

	if !el.Required {
		res.Add(jen.Comment("+optional").Line())
		if !strings.HasPrefix(el.Type, "*") {
			res.Add(jen.Id(el.Name).Op("*").Id(el.Type))
		} else {
			res.Add(jen.Id(el.Name).Id(el.Type))
		}
		res.Add(jen.Tag(map[string]string{
			"json": fmt.Sprintf("%s,omitempty", el.JSONName),
		}).Line())
	} else {
		res.Add(jen.Id(el.Name).Id(el.Type))
		res.Add(jen.Tag(map[string]string{
			"json": el.JSONName,
		}).Line())
	}

	return res
}

func renderStatus(kind, key string, el transpiler.Struct, managed bool) jen.Code {
	fields := []jen.Code{}

	if key == "Root" {
		key = strutil.ToGolangName(fmt.Sprintf("%sStatus", kind))
		if managed {
			fields = append(fields,
				jen.Qual(pkgCommon, "ManagedStatus").
					Tag(map[string]string{
						"json": ",inline",
					}),
				jen.Id("FailedObjectRef").Op("*").Id("FailedObjectRef").
					Tag(map[string]string{
						"json": "failedObjectRef,omitempty",
					}),
			)
		}
	}

	for _, f := range el.Fields {
		fields = append(fields, renderField(f))
	}

	return jen.Type().Id(key).Struct(fields...).Line()
}

func createFailedObjectRef() jen.Code {
	meta := []transpiler.Field{
		{
			Name:        "APIVersion",
			JSONName:    "apiVersion",
			Description: "API version of the object.",
			Optional:    ptr.To(false),
			Type:        "string",
		},
		{
			Name:        "Kind",
			JSONName:    "kind",
			Description: "Kind of the object.",
			Optional:    ptr.To(false),
			Type:        "string",
		},
		{
			Name:        "Name",
			JSONName:    "name",
			Description: "Name of the object.",
			Optional:    ptr.To(false),
			Type:        "string",
		},
		{
			Name:        "Namespace",
			JSONName:    "namespace",
			Description: "Namespace of the object.",
			Optional:    ptr.To(false),
			Type:        "string",
		},
	}

	fields := []jen.Code{}
	for _, el := range meta {
		fields = append(fields, renderField(el))
	}

	return jen.Type().Id("FailedObjectRef").Struct(fields...).Line()
}
