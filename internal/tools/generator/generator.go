package generator

// import (
// 	"bytes"
// 	"context"
// 	"errors"
// 	"fmt"
// 	"io"
// 	"io/fs"
// 	"os"
// 	"os/exec"
// 	"path/filepath"
// 	"strings"

// 	"github.com/gobuffalo/flect"
// 	"github.com/krateoplatformops/core-provider/apis/definitions/v1alpha1"
// 	"github.com/krateoplatformops/core-provider/internal/helm/getter"
// 	"github.com/krateoplatformops/oasgen-provider/internal/tools/generator/code"
// 	"github.com/krateoplatformops/oasgen-provider/internal/tools/generator/text"
// 	"github.com/krateoplatformops/oasgen-provider/internal/tools/generator/tgzfs"
// 	"k8s.io/apimachinery/pkg/runtime/schema"
// 	"sigs.k8s.io/yaml"
// )

// const (
// 	tmpModPathFmt = "github.com/krateoplatformops/%s"
// 	defaultGroup  = "composition.krateo.io"
// )

// type CRDGenerator interface {
// 	GVK() (schema.GroupVersionKind, error)
// 	Generate(ctx context.Context) ([]byte, error)
// }

// func ForSpec(ctx context.Context, nfo *v1alpha1.ChartInfo) (CRDGenerator, error) {
// 	if nfo == nil {
// 		return nil, fmt.Errorf("chart infos cannot be nil")
// 	}

// 	dat, _, err := getter.Get(getter.GetOptions{
// 		URI:     nfo.Url,
// 		Version: nfo.Version,
// 		Repo:    nfo.Repo,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	return ForData(ctx, dat)
// }

// func ForData(ctx context.Context, bin []byte) (CRDGenerator, error) {
// 	pkg, err := tgzfs.New(bytes.NewBuffer(bin))
// 	if err != nil {
// 		return nil, err
// 	}

// 	all, err := fs.ReadDir(pkg, ".")
// 	if err != nil {
// 		return nil, err
// 	}

// 	if len(all) != 1 {
// 		return nil, fmt.Errorf("tgz archive should contain only one root dir")
// 	}

// 	return &defaultCRDGenerator{
// 		tgzFS:   pkg,
// 		rootDir: all[0].Name(),
// 	}, nil
// }

// var _ CRDGenerator = (*defaultCRDGenerator)(nil)

// type defaultCRDGenerator struct {
// 	tgzFS   fs.FS
// 	rootDir string
// }

// func (g *defaultCRDGenerator) GVK() (schema.GroupVersionKind, error) {
// 	fin, err := g.tgzFS.Open(g.rootDir + "/Chart.yaml")
// 	if err != nil {
// 		return schema.GroupVersionKind{}, err
// 	}
// 	defer fin.Close()

// 	din, err := io.ReadAll(fin)
// 	if err != nil {
// 		return schema.GroupVersionKind{}, err
// 	}

// 	res := map[string]interface{}{}
// 	if err := yaml.Unmarshal(din, &res); err != nil {
// 		return schema.GroupVersionKind{}, err
// 	}

// 	name := res["name"].(string)

// 	return schema.GroupVersionKind{
// 		Group:   defaultGroup,
// 		Version: fmt.Sprintf("v%s", strings.ReplaceAll(res["version"].(string), ".", "-")),
// 		Kind:    flect.Pascalize(text.ToGolangName(name)),
// 	}, nil
// }

// func (g *defaultCRDGenerator) Generate(ctx context.Context) ([]byte, error) {
// 	clean := len(os.Getenv("GEN_CLEAN_WORKDIR")) == 0

// 	cfg, res, err := g.crdInfoFromChart()
// 	if err != nil {
// 		return nil, err
// 	}
// 	if clean {
// 		defer os.RemoveAll(cfg.Workdir)
// 	}
// 	if err := code.Do(&res, cfg); err != nil {
// 		return nil, err
// 	}

// 	cmd := exec.Command("go", "mod", "init", cfg.Module)
// 	cmd.Dir = cfg.Workdir
// 	if err := cmd.Run(); err != nil {
// 		return nil, fmt.Errorf("%s: performing 'go mod init' (workdir: %s, module: %s, gvk: %s/%s,%s)",
// 			err.Error(), cfg.Workdir, cfg.Module, res.Group, res.Version, res.Kind)
// 	}

// 	cmd = exec.Command("go", "mod", "tidy")
// 	cmd.Dir = cfg.Workdir
// 	out, err := cmd.CombinedOutput()
// 	if err != nil {
// 		if len(out) > 0 {
// 			return nil, fmt.Errorf("%s: performing 'go mod tidy' (workdir: %s, module: %s, gvk: %s/%s,%s)",
// 				string(out), cfg.Workdir, cfg.Module, res.Group, res.Version, res.Kind)
// 		}
// 		return nil, fmt.Errorf("%s: performing 'go mod tidy' (workdir: %s, module: %s, gvk: %s/%s,%s)",
// 			err.Error(), cfg.Workdir, cfg.Module, res.Group, res.Version, res.Kind)
// 	}

// 	cmd = exec.Command("go",
// 		"run",
// 		"--tags",
// 		"generate",
// 		"sigs.k8s.io/controller-tools/cmd/controller-gen",
// 		"object:headerFile=./hack/boilerplate.go.txt",
// 		"paths=./...", "crd:crdVersions=v1",
// 		"output:artifacts:config=./crds",
// 	)
// 	cmd.Dir = cfg.Workdir
// 	out, err = cmd.CombinedOutput()
// 	if err != nil {
// 		if len(out) > 0 {
// 			return nil, fmt.Errorf("%s: performing 'go run --tags generate...' (workdir: %s, module: %s, gvk: %s/%s,%s)",
// 				string(out), cfg.Workdir, cfg.Module, res.Group, res.Version, res.Kind)
// 		}
// 		return nil, fmt.Errorf("%s: performing 'go run --tags generate...' (workdir: %s, module: %s, gvk: %s/%s,%s)",
// 			err.Error(), cfg.Workdir, cfg.Module, res.Group, res.Version, res.Kind)
// 	}

// 	fsys := os.DirFS(cfg.Workdir)
// 	all, err := fs.ReadDir(fsys, "crds")
// 	if err != nil {
// 		return nil, err
// 	}

// 	fp, err := fsys.Open(filepath.Join("crds", all[0].Name()))
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer fp.Close()

// 	dat, err := io.ReadAll(fp)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return dat, nil
// }

// func (g *defaultCRDGenerator) crdInfoFromChart() (opts code.Options, res code.Resource, err error) {
// 	opts.Module = fmt.Sprintf(tmpModPathFmt, g.rootDir)
// 	opts.Workdir = filepath.Join(os.TempDir(), opts.Module)
// 	err = os.MkdirAll(opts.Workdir, os.ModePerm)
// 	if err != nil {
// 		if !errors.Is(err, os.ErrExist) {
// 			return opts, res, err
// 		}
// 	}

// 	dat, err := g.valuesSchemaFromChart()
// 	if err != nil {
// 		return opts, res, err
// 	}

// 	gvk, err := g.GVK()
// 	if err != nil {
// 		return opts, res, err
// 	}

// 	return opts, code.Resource{
// 		Group:      gvk.Group,
// 		Version:    gvk.Version,
// 		Kind:       gvk.Kind,
// 		Schema:     dat,
// 		Categories: []string{"krateo", "composition"},
// 	}, nil
// }

// func (g *defaultCRDGenerator) valuesSchemaFromChart() ([]byte, error) {
// 	fin, err := g.tgzFS.Open(g.rootDir + "/values.schema.json")
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer fin.Close()

// 	return io.ReadAll(fin)
// }
