package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/krateoplatformops/provider-runtime/pkg/helpers"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"github.com/krateoplatformops/provider-runtime/pkg/ratelimiter"
	"github.com/matteogastaldello/swaggergen-provider/apis"

	"github.com/krateoplatformops/provider-runtime/pkg/controller"
	definition "github.com/matteogastaldello/swaggergen-provider/internal/controllers/definition"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/stoewer/go-strcase"
)

const (
	providerName = "OAS Gen"
)

func main() {
	envVarPrefix := fmt.Sprintf("%s_PROVIDER", strcase.UpperSnakeCase(providerName))

	var (
		app = kingpin.New(filepath.Base(os.Args[0]), fmt.Sprintf("Krateo %s Provider.", providerName)).
			DefaultEnvars()
		servicePortHack = app.Flag("service-port-hack", "Force Kubernetes Service Port env variable hack.").
				OverrideDefaultFromEnvar(fmt.Sprintf("%s_SERVICE_PORT_HACK", envVarPrefix)).
				Bool()
		debug = app.Flag("debug", "Run with debug logging.").Short('d').
			OverrideDefaultFromEnvar(fmt.Sprintf("%s_DEBUG", envVarPrefix)).
			Bool()
		syncPeriod = app.Flag("sync", "Controller manager sync period such as 300ms, 1.5h, or 2h45m").Short('s').
				Default("1h").
				Duration()
		pollInterval = app.Flag("poll", "Poll interval controls how often an individual resource should be checked for drift.").
				Default("2m").
				OverrideDefaultFromEnvar(fmt.Sprintf("%s_POLL_INTERVAL", envVarPrefix)).
				Duration()
		maxReconcileRate = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").
					Default("5").
					OverrideDefaultFromEnvar(fmt.Sprintf("%s_MAX_RECONCILE_RATE", envVarPrefix)).
					Int()
		leaderElection = app.Flag("leader-election", "Use leader election for the controller manager.").
				Short('l').
				Default("false").
				OverrideDefaultFromEnvar(fmt.Sprintf("%s_LEADER_ELECTION", envVarPrefix)).
				Bool()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if helpers.Bool(servicePortHack) {
		helpers.FixKubernetesServicePort()
	}

	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl.WithName(fmt.Sprintf("%s-provider", strcase.KebabCase(providerName))))
	if *debug {
		// The controller-runtime runs with a no-op logger by default. It is
		// *very* verbose even at info level, so we only provide it a real
		// logger when we're running in debug mode.
		ctrl.SetLogger(zl)
	}

	log.Debug("Starting", "sync-period", syncPeriod.String())

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		LeaderElection:   *leaderElection,
		LeaderElectionID: fmt.Sprintf("leader-election-%s-provider", strcase.KebabCase(providerName)),
		Cache: cache.Options{
			SyncPeriod: syncPeriod,
		},
		Metrics: metricsserver.Options{
			BindAddress: ":8080",
		},
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")

	o := controller.Options{
		Logger:                  log,
		MaxConcurrentReconciles: *maxReconcileRate,
		PollInterval:            *pollInterval,
		GlobalRateLimiter:       ratelimiter.NewGlobal(*maxReconcileRate),
	}

	kingpin.FatalIfError(apis.AddToScheme(mgr.GetScheme()), "Cannot add APIs to scheme")
	kingpin.FatalIfError(definition.Setup(mgr, o), "Cannot setup controllers")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
