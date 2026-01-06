package app

import (
	"context"
	"sync"

	"github.com/oracle/oci-cloud-controller-manager/pkg/oci/client"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"go.uber.org/zap"
	cloudControllerManagerConfig "k8s.io/cloud-provider/app/config"
	"k8s.io/cloud-provider/options"

	csicontroller "github.com/oracle/oci-cloud-controller-manager/cmd/oci-csi-controller-driver/csi-controller"
	"github.com/oracle/oci-cloud-controller-manager/cmd/oci-csi-controller-driver/csioptions"
)

const csiController = "csi-controller"

type CSIController struct {
	ctx    context.Context
	logger *zap.SugaredLogger
	mgr    ctrl.Manager
}

// NewCSIController creates a new CloudControllerManager instance.
func NewCSIController(ctx context.Context, logger *zap.SugaredLogger, mgr ctrl.Manager) (Controller, error) {
	return &CSIController{
		ctx:    ctx,
		logger: logger,
		mgr:    mgr,
	}, nil
}

func (c *CSIController) Run(mgr ctrl.Manager, config *cloudControllerManagerConfig.CompletedConfig, options *options.CloudControllerManagerOptions) error {
	return c.RunCsiController(config, options)
}

func (c *CSIController) AddHealthCheck(mgr ctrl.Manager, extraArgs ...interface{}) error {
	c.logger.Infof(csiController, mgr)

	if err := mgr.AddHealthzCheck(csiController, healthz.Ping); err != nil {
		c.logger.Errorf("Failed to add healthz check: %v", err)
		client.SetHealthStatusForController(csiController, "failed")
		return err
	}
	client.SetHealthStatusForController(csiController, "ok")

	if err := mgr.AddReadyzCheck(csiController, healthz.Ping); err != nil {
		c.logger.Errorf("Failed to add readyz check: %v", err)
		client.SetReadinessForController(csiController, "failed")
		return err
	}
	client.SetReadinessForController(csiController, "ok")

	return nil
}

func (c *CSIController) RunCsiController(config *cloudControllerManagerConfig.CompletedConfig, options *options.CloudControllerManagerOptions) error {
	csioption.Master = options.Master
	csioption.Kubeconfig = options.Generic.ClientConnection.Kubeconfig
	csioption.FssCsiAddress = csioptions.GetFssAddress(csioption.CsiAddress, defaultFssAddress)
	csioption.FssEndpoint = csioptions.GetFssAddress(csioption.Endpoint, defaultFssEndpoint)
	csioption.FssVolumeNamePrefix = csioptions.GetFssVolumeNamePrefix(csioption.VolumeNamePrefix)
	csioption.LustreCsiAddress = csioptions.GetLustreAddress(csioption.CsiAddress, defaultLustreAddress)
	csioption.LustreEndpoint = csioptions.GetLustreAddress(csioption.Endpoint, defaultLustreEndpoint)
	csioption.LustreVolumeNamePrefix = csioptions.GetLustreVolumeNamePrefix(csioption.VolumeNamePrefix)

	// Check and update feature gate for CrossNamespaceDataSource
	csioption.FeatureGates = csioptions.UpdateFeatureGates(csioption.FeatureGates)
	csioption.RuntimeSchemeMutex = new(sync.Mutex)
	err := csicontroller.Run(csioption, c.ctx.Done())
	if err != nil {
		c.logger.With(zap.Error(err)).Error("Error running csi-controller")
	}
	return nil
}
