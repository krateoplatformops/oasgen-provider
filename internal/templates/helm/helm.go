package helm

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// Chart represents a loaded Helm chart.
type Chart struct {
	c *chart.Chart
}

// TemplateConfig is the configuration for Helm Template rendering.
type TemplateConfig struct {
	// ReleaseName is the name of the release.
	ReleaseName string
	// Chart is the loaded chart. Use `LoadChart`.
	Chart *Chart
	// Values are the custom values to be used on the chart template..
	Values map[string]interface{}
	// IncludeCRDs when enabled will template/render the CRDs.
	IncludeCRDs bool
	// Namespace is the namespace used to render the chart.
	Namespace string
	// ShowFiles is a list of files that can be used to only template the provided files,
	// by default it will render all.
	// This can be handy on specific use cases like unit tests for charts.
	ShowFiles []string
}

func (c *TemplateConfig) defaults() error {
	if c.Chart == nil {
		return fmt.Errorf("chart is required")
	}

	if c.ReleaseName == "" {
		return fmt.Errorf("release name is required")
	}

	if c.Values == nil {
		c.Values = map[string]interface{}{}
	}

	return nil
}

// Template will runhelm template in the provided chart and values without the need of the Helm binary
// and without executing an external command.
func Template(ctx context.Context, config TemplateConfig) (string, error) {
	err := config.defaults()
	if err != nil {
		return "", fmt.Errorf("invalid configuration: %w", err)
	}

	// Create chart renderer.
	client := action.NewInstall(&action.Configuration{})
	client.ClientOnly = true
	client.DryRun = true
	client.ReleaseName = config.ReleaseName
	client.IncludeCRDs = config.IncludeCRDs
	client.Namespace = config.Namespace

	// Render chart.
	rel, err := client.Run(config.Chart.c, config.Values)
	if err != nil {
		return "", fmt.Errorf("could not render helm chart correctly: %w", err)
	}

	manifests := rel.Manifest
	if len(config.ShowFiles) > 0 {
		manifests, err = filterFiles(manifests, config.ShowFiles)
		if err != nil {
			return "", fmt.Errorf("could not filter manifest files: %w", err)
		}
	}

	return manifests, nil
}

// LoadChart loads a chart from a fs.FS system.
// There chart files must be at the root of the provided fs.FS.
// e.g: ./Chart.yaml, ./values.yaml ./templates/deployment.yaml...
//
// You can use `fs.Sub` as a helper tool to get the root chart.
func LoadChart(ctx context.Context, f fs.FS) (*Chart, error) {
	files := []*loader.BufferedFile{}

	err := fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(f, path)
		if err != nil {
			return fmt.Errorf("could not read manifest %s: %w", path, err)
		}

		files = append(files, &loader.BufferedFile{
			Name: path,
			Data: data,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not walk chart directory: %w", err)
	}

	chart, err := loader.LoadFiles(files)
	if err != nil {
		return nil, fmt.Errorf("could not load chart from files: %w", err)
	}

	return &Chart{c: chart}, nil
}

var (
	splitMarkRe             = regexp.MustCompile("(?m)^---")
	chartRenderedFileNameRe = regexp.MustCompile(`(?m)^# Source:(.*)$`)
)

func filterFiles(rendered string, files []string) (string, error) {
	renderedSplit := splitMarkRe.Split(rendered, -1)
	// Create an index to check if we need to filter (and a counter to see if we filtered something related with the file).
	fileIndexAndMatched := map[string]int{}
	for _, f := range files {
		fileIndexAndMatched[f] = 0
	}

	filteredRendered := []string{}
	for _, t := range renderedSplit {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}

		// Get file name.
		match := chartRenderedFileNameRe.FindStringSubmatch(t)
		if len(match) == 0 {
			return "", fmt.Errorf("could not match file")
		}

		// Sanitize and remove chart name.
		renderedFile := strings.TrimSpace(match[1])
		_, renderedFile, _ = strings.Cut(renderedFile, "/")

		// If the file is the one we want to filter, add to result.
		if _, ok := fileIndexAndMatched[renderedFile]; ok {
			filteredRendered = append(filteredRendered, t)
			fileIndexAndMatched[renderedFile]++
		}
	}

	// Check all files matched at least once.
	for k, v := range fileIndexAndMatched {
		if v == 0 {
			return "", fmt.Errorf("file %q didn't have any file match", k)
		}
	}

	var b bytes.Buffer
	for _, m := range filteredRendered {
		_, _ = fmt.Fprintf(&b, "\n---\n%s", strings.TrimSpace(m))
	}
	result := strings.TrimSpace(b.String())

	return result, nil
}
