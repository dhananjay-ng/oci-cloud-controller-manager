// Copyright 2019 Oracle and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	cloudControllerManager "k8s.io/cloud-provider/app"
	cloudControllerManagerConfig "k8s.io/cloud-provider/app/config"
	"k8s.io/cloud-provider/names"
	"k8s.io/cloud-provider/options"
	cliflag "k8s.io/component-base/cli/flag"
	utilflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/term"
	"k8s.io/component-base/version/verflag"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/oracle/oci-cloud-controller-manager/cmd/oci-csi-controller-driver/csioptions"
	"github.com/oracle/oci-cloud-controller-manager/pkg/cloudprovider/providers/oci"
	"github.com/oracle/oci-cloud-controller-manager/pkg/logging"
	"github.com/oracle/oci-cloud-controller-manager/pkg/oci/client"
)

var (
	logLevel                                                           int8
	resourcePrincipalFile, logfilePath                                 string
	enableCSI, enableVolumeProvisioning, useResourcePrincipal, logJSON bool
	enableNPNController                                                bool
	enableOCIServiceController                                         bool
	resourcePrincipalInitialTimeout                                    time.Duration
)

var csioption = csioptions.CSIOptions{}

var (
	scheme                    = runtime.NewScheme()
	controllerManagerSetupLog = ctrl.Log.WithName("node-crd-controller-setup")
	configFilePath            = "/etc/oci/config.yaml"
)

const maxRetries = 50
const initialRetryDelay = 5 * time.Second

const (
	defaultFssAddress  = "/var/run/shared-tmpfs/csi-fss.sock"
	defaultFssEndpoint = "unix:///var/run/shared-tmpfs/csi-fss.sock"
	defaultLustreAddress  = "/var/run/shared-tmpfs/csi-lustre.sock"
	defaultLustreEndpoint = "unix:///var/run/shared-tmpfs/csi-lustre.sock"
)

// constant values for bvr and reboot APIs rate limiter used by node operation rule controller
const (
	norDefaultRateLimit     = 10
	norMaxBurstTokens   int = 10
)

const (
	initialInterval = 10 * time.Second
	maxInterval     = 5 * time.Minute
	jitterFactor    = 0.2
)

// NewCloudProviderOCICommand creates a *cobra.Command object with default parameters
func NewCloudProviderOCICommand(logger *zap.SugaredLogger) *cobra.Command {

	// FIXME Create CLoudProviderOCIOptions struct that shall contain options for all the components
	s, err := options.NewCloudControllerManagerOptions()
	if err != nil {
		logger.With(zap.Error(err)).Fatalf("unable to initialize command options")
	}

	command := &cobra.Command{
		Use: "cloud-provider-oci",
		Long: `The cloud provider oci daemon is a agglomeration of oci cloud controller
manager and oci volume provisioner. It embeds the cloud specific control loops shipped with Kubernetes.`,
		Run: func(cmd *cobra.Command, args []string) {
			log := logging.Logger()
			defer log.Sync()
			zap.ReplaceGlobals(log)
			logger = log.Sugar()
			verflag.PrintAndExitIfRequested()
			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				logger.Infof("FLAG: --%s=%q", flag.Name, flag.Value)
			})

			c, err := s.Config(cloudControllerManager.ControllerNames(cloudControllerManager.DefaultInitFuncConstructors),
				cloudControllerManager.ControllersDisabledByDefault.List(), names.CCMControllerAliases(), cloudControllerManager.AllWebhooks, cloudControllerManager.DisabledByDefaultWebhooks)
			if err != nil {
				logger.With(zap.Error(err)).Fatalf("Unable to create cloud controller manager config")
			}

			run(logger, c.Complete(), s)

		},
	}

	namedFlagSets := s.Flags(cloudControllerManager.ControllerNames(cloudControllerManager.DefaultInitFuncConstructors),
		cloudControllerManager.ControllersDisabledByDefault.List(), names.CCMControllerAliases(), cloudControllerManager.AllWebhooks, cloudControllerManager.DisabledByDefaultWebhooks)

	// logging parameters flagset
	loggingFlagSet := namedFlagSets.FlagSet("logging variables")
	loggingFlagSet.Int8Var(&logLevel, "log-level", int8(zapcore.InfoLevel), "Adjusts the level of the logs that will be omitted.")
	loggingFlagSet.BoolVar(&logJSON, "log-json", false, "Log in json format.")
	loggingFlagSet.StringVar(&logfilePath, "logfile-path", "", "If specified, write log messages to a file at this path.")

	// prometheus metrics endpoint flagset
	metricsFlagSet := namedFlagSets.FlagSet("metrics endpoint")
	metricsFlagSet.StringVar(&metricsEndpoint, "metrics-endpoint", "0.0.0.0:8080", "The endpoint where to expose metrics")

	// volume provisioner flag set
	vpFlagSet := namedFlagSets.FlagSet("volume provisioner")
	vpFlagSet.BoolVar(&enableVolumeProvisioning, "enable-volume-provisioning", true, "When enabled volumes will be provisioned/deleted by cloud controller manager")
	vpFlagSet.BoolVar(&volumeRoundingEnabled, "rounding-enabled", true, "When enabled volumes will be rounded up if less than 'minVolumeSizeMB'")
	vpFlagSet.StringVar(&minVolumeSize, "min-volume-size", "50Gi", "The minimum size for a block volume. By default OCI only supports block volumes > 50GB")

	// oci authentication mode flag set
	ociAuthFlagSet := namedFlagSets.FlagSet("oci authentication modes")
	ociAuthFlagSet.BoolVar(&useResourcePrincipal, "use-resource-principal", false, "If true use resource principal as authentication mode else use service principal as authentication mode")
	ociAuthFlagSet.StringVar(&resourcePrincipalFile, "resource-principal-file", "", "The filesystem path at which the serialized Resource Principal is stored")
	ociAuthFlagSet.DurationVar(&resourcePrincipalInitialTimeout, "resource-principal-initial-timeout", 1*time.Minute, "How long to wait for an initial Resource Principal before terminating with an error if one is not supplied")

	// csi flag set.
	csiFlagSet := namedFlagSets.FlagSet("CSI Controller")
	csiFlagSet.BoolVar(&enableCSI, "csi-enabled", false, "Whether to enable CSI feature in OKE")
	csiFlagSet.StringVar(&csioption.CsiAddress, "csi-address", "/run/csi/socket", "Address of the CSI Block Volume driver socket.")
	csiFlagSet.StringVar(&csioption.Endpoint, "csi-endpoint", "unix://tmp/csi.sock", "CSI Block Volume endpoint")
	csiFlagSet.StringVar(&csioption.VolumeNamePrefix, "csi-volume-name-prefix", "pvc", "Prefix to apply to the name of a created volume.")
	csiFlagSet.IntVar(&csioption.VolumeNameUUIDLength, "csi-volume-name-uuid-length", -1, "Truncates generated UUID of a created volume to this length. Defaults behavior is to NOT truncate.")
	csiFlagSet.BoolVar(&csioption.ShowVersion, "csi-version", false, "Show version.")
	csiFlagSet.DurationVar(&csioption.RetryIntervalStart, "csi-retry-interval-start", time.Second, "Initial retry interval of failed provisioning or deletion. It doubles with each failure, up to retry-interval-max.")
	csiFlagSet.DurationVar(&csioption.RetryIntervalMax, "csi-retry-interval-max", 5*time.Minute, "Maximum retry interval of failed provisioning or deletion.")
	csiFlagSet.UintVar(&csioption.WorkerThreads, "csi-worker-threads", 100, "Number of provisioner worker threads, in other words nr. of simultaneous CSI calls.")
	csiFlagSet.DurationVar(&csioption.OperationTimeout, "csi-op-timeout", 120*time.Second, "Timeout for waiting for creation or deletion of a volume")
	csiFlagSet.BoolVar(&csioption.EnableLeaderElection, "csi-enable-leader-election", false, "Enables leader election. If leader election is enabled, additional RBAC rules are required. Please refer to the Kubernetes CSI documentation for instructions on setting up these RBAC rules.")
	csiFlagSet.StringVar(&csioption.LeaderElectionType, "csi-leader-election-type", "endpoints", "the type of leader election, options are 'endpoints' (default) or 'leases' (strongly recommended). The 'endpoints' option is deprecated in favor of 'leases'.")
	csiFlagSet.StringVar(&csioption.LeaderElectionNamespace, "csi-leader-election-namespace", "", "Namespace where the leader election resource lives. Defaults to the pod namespace if not set.")
	csiFlagSet.BoolVar(&csioption.StrictTopology, "csi-strict-topology", false, "Passes only selected node topology to CreateVolume Request, unlike default behavior of passing aggregated cluster topologies that match with topology keys of the selected node.")
	csiFlagSet.BoolVar(&csioption.ImmediateTopology, "csi-immediate-topology", true, "Immediate binding: pass aggregated cluster topologies for all nodes where the CSI driver is available (enabled, the default) or no topology requirements (if disabled)")
	csiFlagSet.DurationVar(&csioption.Resync, "csi-resync", 10*time.Minute, "Resync interval of the controller.")
	csiFlagSet.DurationVar(&csioption.Timeout, "csi-timeout", 15*time.Second, "Timeout for waiting for attaching or detaching the volume.")
	csiFlagSet.BoolVar(&csioption.EnableResizer, "csi-bv-expansion-enabled", false, "Enables go routine csi-resizer.")
	csiFlagSet.UintVar(&csioption.FinalizerThreads, "cloning-protection-threads", 1, "Number of simultaneously running threads, handling cloning finalizer removal")
	csiFlagSet.Var(utilflag.NewMapStringBool(&csioption.FeatureGates), "csi-feature-gates", "A set of key=value pairs that describe feature gates for alpha/experimental features. ")
	csiFlagSet.StringVar(&csioption.GroupSnapshotNamePrefix, "groupsnapshot-name-prefix", "groupsnapshot", "Prefix to apply to the name of a created group snapshot.")
	csiFlagSet.IntVar(&csioption.GroupSnapshotNameUUIDLength, "groupsnapshot-name-uuid-length", -1, "Length in characters for the generated uuid of a created group snapshot. Defaults behavior is to NOT truncate.")

	verflag.AddFlags(namedFlagSets.FlagSet("global"))
	globalflag.AddGlobalFlags(namedFlagSets.FlagSet("global"), command.Name())

	npnFlagSet := namedFlagSets.FlagSet("NPN Controller")
	npnFlagSet.BoolVar(&enableNPNController, "enable-npn-controller", false, "Whether to enable Native Pod Network controller")

	ociSvcCtrlFlagSet := namedFlagSets.FlagSet("OCI Service Controller")
	ociSvcCtrlFlagSet.BoolVar(&enableOCIServiceController, "enable-oci-service-controller", false, "Whether to enable OCI service controller instead of Kubernetes Cloud Provider service controller")

	if flag.CommandLine.Lookup("cloud-provider-gce-lb-src-cidrs") != nil {
		// hoist this flag from the global flagset to preserve the commandline until
		// the gce cloudprovider is removed.
		globalflag.Register(namedFlagSets.FlagSet("generic"), "cloud-provider-gce-lb-src-cidrs")
	}
	for _, f := range namedFlagSets.FlagSets {
		command.Flags().AddFlagSet(f)
	}
	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(command.OutOrStdout())
	command.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), namedFlagSets, cols)
		return nil
	})
	command.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
	})

	viper.BindPFlags(command.Flags())

	return command
}

func run(logger *zap.SugaredLogger, config *cloudControllerManagerConfig.CompletedConfig, options *options.CloudControllerManagerOptions) {
	var wg sync.WaitGroup
	ctx, cancelFunc := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 2)
	defer close(sigs)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		cancelFunc()
		<-sigs
		os.Exit(1)
	}()

	err := InitSharedManager(scheme, options)
	if err != nil {
		logger.Fatalf("Failed to initialise manager %v", err)
	}
	mgr, err := GetSharedManager()
	if err != nil {
		logger.Fatalf("Failed to get manager %v", err)
	}

	// Metrics Controller
	wg.Add(1)
	go func() {
		defer wg.Done()
		runControllerWithRetry(ctx, mgr, metricsController, config, options, logger)
	}()

	// Volume Provisioner
	if enableVolumeProvisioning {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runControllerWithRetry(ctx, mgr, volumeProvisioner, config, options, logger)
		}()
	} else {
		logger.Info("Volume provisioning is disabled")
	}

	// CCM controller
	wg.Add(1)
	go func() {
		defer wg.Done()
		runControllerWithRetry(ctx, mgr, ccmController, config, options, logger)
	}()

	// CSI controller
	if enableCSI == true {
		logger.Info("CSI is enabled.")
		wg.Add(1)
		go func() {
			defer wg.Done()
			runControllerWithRetry(ctx, mgr, csiController, config, options, logger)
		}()
	} else {
		logger.Info("CSI is disabled.")
	}

	enableNPN := oci.GetIsFeatureEnabledFromEnv(logger, "ENABLE_NPN_CONTROLLER", false)
	enableNOR := oci.GetIsFeatureEnabledFromEnv(logger, "ENABLE_NOR_CONTROLLER", false)
	enableNodeControllers := shouldStartControllerManager(enableNOR, enableNPN, enableNPNController)

	// NPN controller
	if enableNodeControllers {
		//setup NPN Controller with manager if NPN Enabled
		if enableNPN || enableNPNController {
			npnCtrl, err := GetController(npnController, ctx, logger, mgr)
			if err != nil {
				logger.Fatalf("Failed to create NPN Controller: %v", err)
			}

			if err := npnCtrl.Run(mgr, config, options); err != nil {
				logger.Fatalf("NPN Controller failed to start: %v", err)
			}
		}
	} else {
		logger.Info("crd manager not instantiated as no customer resource enabled.")
	}

	// NOR controller
	if enableNodeControllers {
		if enableNOR {
			norCtrl, err := GetController(norController, ctx, logger, mgr)
			if err != nil {
				logger.Fatalf("Failed to create NPN Controller: %v", err)
			}

			if err := norCtrl.Run(mgr, config, options); err != nil {
				logger.Fatalf("NOR Controller failed to start: %v", err)
			}
		}
	} else {
		logger.Info("crd manager not instantiated as no customer resource enabled.")
	}

	controllers := []string{metricsController, ccmController, volumeProvisioner}
	if enableCSI {
		controllers = append(controllers, csiController)
	}
	if enableNPN {
		controllers = append(controllers, npnController)
	}
	if enableNOR {
		controllers = append(controllers, norController)
	}
	for _, controllerName := range controllers {
		controller, err := GetController(controllerName, ctx, logger, mgr)
		if err != nil {
			logger.Fatalf("Failed to create %s: %v", controllerName, err)
		}
		if err := controller.AddHealthCheck(mgr, options); err != nil {
			logger.Fatalf("Failed to add health checks for %s: %v", controllerName, err)
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		monitorHealth(ctx, logger)
	}()

	//Now that all the controllers are hooked up, start the manager
	logger = logger.With(zap.String("component", "shared-manager"))
	ctrl.SetLogger(zapr.NewLogger(logger.Desugar()))
	controllerManagerSetupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		controllerManagerSetupLog.Error(err, "problem running manager")
	}
	// wait for all the go routines to finish.
	wg.Wait()
}

func runControllerWithRetry(ctx context.Context, mgr ctrl.Manager, controllerName string, config *cloudControllerManagerConfig.CompletedConfig, options *options.CloudControllerManagerOptions, logger *zap.SugaredLogger) {
	logger = logger.With(zap.String("component", controllerName))
	for {
		select {
		case <-ctx.Done():
			logger.Infof("Stopping retries for %s due to context cancellation", controllerName)
			return
		default:
			controller, err := GetController(controllerName, ctx, logger, mgr)
			if err != nil {
				logger.Errorf("Failed to get controller: %v", err)
				time.Sleep(initialInterval)
				continue
			}

			var retryInterval = initialInterval

			for {
				select {
				case <-ctx.Done():
					logger.Infof("Stopping retries for %s due to context cancellation", controllerName)
					return
				default:
					err := func() error {
						defer func() {
							if r := recover(); r != nil {
								logger.Errorf("Recovered from panic in %s: %v", controllerName, r)
							}
						}()
						return controller.Run(mgr, config, options)
					}()

					if err != nil {
						logger.Errorf("Controller %s failed: %v", controllerName, err)
						retryInterval = calculateExponentialBackoff(retryInterval, maxInterval, jitterFactor)
						logger.Infof("Retrying %s in %s...", controllerName, retryInterval)
						time.Sleep(retryInterval)
					} else {
						logger.Infof("Controller %s started successfully", controllerName)
						// If controller exits normally, restart it after a short delay
						time.Sleep(initialInterval)
					}
				}
			}
		}
	}
}

func monitorHealth(ctx context.Context, logger *zap.SugaredLogger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("Recovered from Panic in monitorHealth %v", r)
		}
	}()

	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Shutting down health monitoring")
			return
		case <-ticker.C:
			controllers := []string{metricsController, ccmController, csiController, volumeProvisioner, npnController, norController}
			for _, controller := range controllers {
				healthResp, err := getHealth(controller)
				if err != nil {
					logger.Error("Error occurred", err)
				}
				client.SetHealthStatusForController(controller, healthResp)

				livenessResp, err := getLiveness(controller)
				if err != nil {
					logger.Error("Error occurred", err)
				}
				client.SetHealthStatusForController(controller, livenessResp)
			}
		}
	}
}
