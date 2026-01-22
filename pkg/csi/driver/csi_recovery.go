package driver

import (
	"fmt"
	"runtime/debug"

	"github.com/oracle/oci-cloud-controller-manager/pkg/metrics"
	"github.com/oracle/oci-cloud-controller-manager/pkg/util"
	"go.uber.org/zap"
)

// MakeCSIPanicRecovery returns a defer-able closure that recovers from panics in CSI RPCs,
// logs the panic with a stack trace, and emits a panic metric. Use like:
//
//	defer MakeCSIPanicRecovery(logger, metricPusher, "CreateVolume", map[string]string{ metrics.ResourceOCIDDimension: req.GetName() })()
func MakeCSIPanicRecovery(logger *zap.SugaredLogger, metricPusher *metrics.MetricPusher, op string, extraDims map[string]string) func() {
	return func() {
		if rec := recover(); rec != nil {
			err := fmt.Errorf("panic recovered %v stack is %s", rec, string(debug.Stack()))
			// Log with generic CSI component context
			logger.With(zap.Error(err)).With("operation", op).Error("Recovered from panic in CSI RPC")

			// Build metric dimensions and emit PANIC metric (CSI scope)
			dimensionsMap := map[string]string{}
			for k, v := range extraDims {
				dimensionsMap[k] = v
			}
			metricDimension := util.GetComponentForMetricDimension(util.PANIC, util.CSIStorageType)
			dimensionsMap[metrics.ComponentDimension] = metricDimension
			metrics.SendMetricData(metricPusher, metricDimension, 1, dimensionsMap)
		}
	}
}
