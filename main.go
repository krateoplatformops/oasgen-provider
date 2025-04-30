package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/krateoplatformops/oasgen-provider/internal/controllers"
	"github.com/krateoplatformops/snowplow/plumbing/env"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/krateoplatformops/oasgen-provider/apis"
	"github.com/krateoplatformops/provider-runtime/pkg/controller"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"github.com/krateoplatformops/provider-runtime/pkg/ratelimiter"

	"github.com/stoewer/go-strcase"
)

const (
	providerName = "oasgen"
)

func main() {
	envVarPrefix := fmt.Sprintf("%s_PROVIDER", strcase.UpperSnakeCase(providerName))

	debug := flag.Bool("debug", env.Bool(fmt.Sprintf("%s_DEBUG", envVarPrefix), false), "Run with debug logging.")
	namespace := flag.String("namespace", env.String(fmt.Sprintf("%s_NAMESPACE", envVarPrefix), ""), "Watch resources only in this namespace.")
	syncPeriod := flag.Duration("sync", env.Duration(fmt.Sprintf("%s_SYNC", envVarPrefix), time.Hour*1), "Controller manager sync period such as 300ms, 1.5h, or 2h45m")
	pollInterval := flag.Duration("poll", env.Duration(fmt.Sprintf("%s_POLL_INTERVAL", envVarPrefix), time.Minute*3), "Poll interval controls how often an individual resource should be checked for drift.")
	maxReconcileRate := flag.Int("max-reconcile-rate", env.Int(fmt.Sprintf("%s_MAX_RECONCILE_RATE", envVarPrefix), 3), "The global maximum rate per second at which resources may checked for drift from the desired state.")
	leaderElection := flag.Bool("leader-election", env.Bool(fmt.Sprintf("%s_LEADER_ELECTION", envVarPrefix), false), "Use leader election for the controller manager.")
	maxErrorRetryInterval := flag.Duration("max-error-retry-interval", env.Duration(fmt.Sprintf("%s_MAX_ERROR_RETRY_INTERVAL", envVarPrefix), 0*time.Minute), "The maximum interval between retries when an error occurs. This should be less than the half of the poll interval.")
	minErrorRetryInterval := flag.Duration("min-error-retry-interval", env.Duration(fmt.Sprintf("%s_MIN_ERROR_RETRY_INTERVAL", envVarPrefix), 1*time.Second), "The minimum interval between retries when an error occurs. This should be less than max-error-retry-interval.")

	flag.Parse()

	log.Default().SetOutput(io.Discard)
	ctrl.SetLogger(zap.New(zap.WriteTo(io.Discard)))

	zl := zap.New(zap.UseDevMode(*debug))
	logr := logging.NewLogrLogger(zl.WithName(fmt.Sprintf("%s-provider", strcase.KebabCase(providerName))))
	if *debug {
		ctrl.SetLogger(zl)
	}

	if maxErrorRetryInterval.Seconds() == 0 {
		retryInterval := (*pollInterval / 2)
		maxErrorRetryInterval = &retryInterval
	} else if maxErrorRetryInterval.Seconds() >= pollInterval.Seconds() {
		retryInterval := (*pollInterval / 2)
		maxErrorRetryInterval = &retryInterval

		logr.Info("[WARNING] max-error-retry-interval is greater than or equal to poll interval, setting to half of poll interval", "max-error-retry-interval", maxErrorRetryInterval.String())
	}

	if minErrorRetryInterval.Seconds() >= maxErrorRetryInterval.Seconds() {
		retryInterval := 1 * time.Second
		minErrorRetryInterval = &retryInterval

		logr.Info("[WARNING] min-error-retry-interval is greater than or equal to max-error-retry-interval, setting to 1 second", "min-error-retry-interval", minErrorRetryInterval.String())
	}

	logr.Debug("Starting", "sync-period", syncPeriod.String(), "poll-interval", pollInterval.String(), "max-error-retry-interval", maxErrorRetryInterval.String())

	cfg, err := ctrl.GetConfig()
	if err != nil {
		log.Fatalf("Cannot get API server rest config: %v", err)
	}

	co := cache.Options{
		SyncPeriod: syncPeriod,
	}
	if len(*namespace) > 0 {
		co.DefaultNamespaces = map[string]cache.Config{
			*namespace: {},
		}
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		LeaderElection:   *leaderElection,
		LeaderElectionID: fmt.Sprintf("leader-election-%s-provider", strcase.KebabCase(providerName)),
		Cache:            co,
	})
	if err != nil {
		log.Fatalf("Cannot create controller manager: %v", err)
	}

	o := controller.Options{
		Logger:                  logr,
		MaxConcurrentReconciles: *maxReconcileRate,
		PollInterval:            *pollInterval,
		GlobalRateLimiter:       ratelimiter.NewGlobalExponential(*minErrorRetryInterval, *maxErrorRetryInterval),
	}

	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalf("Cannot add APIs to scheme: %v", err)
	}
	if err := controllers.Setup(mgr, o); err != nil {
		log.Fatalf("Cannot setup controllers: %v", err)
	}
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Fatalf("Cannot start controller manager: %v", err)
	}
}
