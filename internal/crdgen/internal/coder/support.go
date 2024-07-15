package coder

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/krateoplatformops/oasgen-provider/internal/crdgen/internal/transpiler"
	"github.com/krateoplatformops/oasgen-provider/internal/crdgen/internal/transpiler/jsonschema"
)

func normalizeVersion(ver string) string {
	return strings.ReplaceAll(ver, "-", "_")
}

func makeDirs(workdir string, dirs ...string) (string, error) {
	all := []string{workdir}
	all = append(all, dirs...)

	path := filepath.Join(all...)

	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			return path, err
		}
	}
	return path, nil
}

func jsonschemaToStruct(r io.Reader) (map[string]transpiler.Struct, error) {
	schema, err := jsonschema.ParseReader(r)
	if err != nil {
		return nil, err
	}

	return transpiler.Transpile(schema)
}
