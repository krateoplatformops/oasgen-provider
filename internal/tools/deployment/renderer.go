package deployment

func marker() {}

// import (
// 	"github.com/krateoplatformops/core-provider/internal/tools/chartfs"
// )

// func RenderChartTemplates(pkg *chartfs.ChartFS) (string, error) {
// 	cwd, err := fs.Sub(pkg, pkg.RootDir())
// 	if err != nil {
// 		return "", err
// 	}

// 	chart, err := helm.LoadChart(context.TODO(), cwd)
// 	if err != nil {
// 		return "", err
// 	}

// 	// Execute helm template.
// 	return helm.Template(context.TODO(), helm.TemplateConfig{
// 		Chart:       chart,
// 		ReleaseName: "test",
// 		Namespace:   "no-kube-system",
// 	})
// }
