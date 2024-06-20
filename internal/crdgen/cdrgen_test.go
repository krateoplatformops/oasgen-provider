//go:build integration
// +build integration

package crdgen_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/krateoplatformops/crdgen"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGenerate(t *testing.T) {
	//os.Setenv("CRDGEN_CLEAN_WORKDIR", "NO")

	res := crdgen.Generate(context.TODO(), crdgen.Options{
		WorkDir: "form1",
		GVK: schema.GroupVersionKind{
			Group:   "example.org",
			Version: "v1alpha1",
			Kind:    "Form",
		},
		SpecJsonSchemaGetter: &urlJsonSchemaGetter{sampleSchemaURL},
	})
	if res.Err != nil {
		t.Fatal(res.Err)
	}

	fmt.Println("digest: ", res.Digest)
	fmt.Println()

	fmt.Println(string(res.Manifest))
}

const (
	sampleSchemaURL = "https://raw.githubusercontent.com/krateoplatformops/krateo-v2-template-fireworksapp/main/chart/values.schema.json"
)

var _ crdgen.JsonSchemaGetter = (*urlJsonSchemaGetter)(nil)

type urlJsonSchemaGetter struct {
	uri string
}

func (f *urlJsonSchemaGetter) Get() ([]byte, error) {
	res, err := http.Get(f.uri)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	return io.ReadAll(http.MaxBytesReader(nil, res.Body, 512*1024))
}
