//go:build integration
// +build integration

package generator

// import (
// 	"context"
// 	"fmt"
// 	"io"
// 	"os"
// 	"testing"

// 	"github.com/krateoplatformops/oasgen-provider/internal/helm/getter"
// )

// const (
// 	testChartFile = "../../../../testdata/charts/postgresql-12.8.3.tgz"
// )

// func TestGeneratorTGZ(t *testing.T) {
// 	buf, _, err := getter.Get(getter.GetOptions{
// 		URI: "https://github.com/lucasepe/busybox-chart/releases/download/v0.2.0/dummy-chart-0.2.0.tgz",
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	gen, err := ForData(context.Background(), buf)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	dat, err := gen.Generate(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	fmt.Println(string(dat))
// }

// func TestGeneratorTGZIssues(t *testing.T) {
// 	buf, _, err := getter.Get(getter.GetOptions{
// 		URI: "https://github.com/krateoplatformops/kargo-chart/releases/download/0.1.4/kargo-0.1.0.tgz",
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	gen, err := ForData(context.Background(), buf)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	//os.Setenv("GEN_CLEAN_WORKDIR", "NO")
// 	dat, err := gen.Generate(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	fmt.Println(string(dat))
// }

// func TestGeneratorOCI(t *testing.T) {
// 	buf, _, err := getter.Get(getter.GetOptions{
// 		URI:     "oci://registry-1.docker.io/bitnamicharts/postgresql",
// 		Version: "12.8.3",
// 		Repo:    "",
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	gen, err := ForData(context.Background(), buf)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	dat, err := gen.Generate(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	fmt.Println(string(dat))
// }

// func TestGeneratorREPO(t *testing.T) {
// 	buf, url, err := getter.Get(getter.GetOptions{
// 		URI:     "https://charts.bitnami.com/bitnami",
// 		Version: "12.8.3",
// 		Repo:    "postgresql",
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	fmt.Println(url)

// 	gen, err := ForData(context.Background(), buf)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	dat, err := gen.Generate(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	fmt.Println(string(dat))
// }

// func TestGeneratorFromFile(t *testing.T) {
// 	fin, err := os.Open(testChartFile)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer fin.Close()

// 	all, err := io.ReadAll(fin)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	gen, err := ForData(context.Background(), all)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	dat, err := gen.Generate(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	fmt.Println(string(dat))
// }
