package controllers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	providercfg "github.com/oracle/oci-cloud-controller-manager/pkg/cloudprovider/providers/oci/config"
	"github.com/oracle/oci-cloud-controller-manager/pkg/metrics"
	ociclient "github.com/oracle/oci-cloud-controller-manager/pkg/oci/client"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	narName                    string         = "narName"
	repairProblemDetectedLabel string         = "oci.oraclecloud.com/node-problem-detected"
	repairEnabledLabel         string         = "oci.oraclecloud.com/node-auto-repair-enabled"
	repairFrequencyLabel       string         = "oci.oraclecloud.com/node-auto-repair-freq"
	REPAIR_TAINT_KEY           string         = "oci.oraclecloud.com/node-auto-repair-scheduled"
	REPAIR_TAINT_EFFECT        v1.TaintEffect = v1.TaintEffectNoSchedule
)

var CONDITIONS map[string]string = map[string]string{
	"IMDSUnreachable": "True",
	"GPUCount":        "True",
	"GPUCdfpCable":    "True",
	"GPURowRemap":     "True",
	"GPUClock":        "True",
	"PCIeBus":         "True",
	"PCIeLanes":       "True",
	"RDMALinkCount":   "True",
	"RxDiscards":      "True",
	"GIDIndex":        "True",
	"RDMALink":        "True",
	"ETHLink":         "True",
	"RDMALinkAuth":    "True",
	"GPUSRAM":         "True",
	"GPUDriver":       "True",
	"ETH0Check":       "True",
	"CDFPCable":       "True",
	"HCACheck":        "True",
	"PCIeInterface":   "True",
	"GPUMemory":       "True",
	"GPUThermal":      "True",
	"SourceRouting":   "True",
	"OCAVersion":      "True",
	"RDMALinkFlap":    "True",
	"RDMANicSpeed":    "True",
	"RDMALinkSpeed":   "True",
	"HPCMetadata":     "True",
	"AdvancedRDMA":    "True",
	"XGMILink":        "True",
}

type NodeAutoRepairReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	leaderElection   <-chan struct{}
	MetricPusher     *metrics.MetricPusher
	OCIClient        ociclient.Interface
	KubeClient       clientset.Interface
	TimeTakenTracker sync.Map
	Recorder         record.EventRecorder
	Config           *providercfg.Config
	ControllerID     string
	leaseManager     *repairLeaseManager
	leaseStopCh      chan struct{}
}

// Cool-down window after a repair finishes. During this window, new repairs are throttled.
var (
	repairCoolDown = getEnvDuration("NODE_AUTOREPAIR_COOLDOWN", 60*time.Minute)
)

// SetupWithManager sets up the controller with the Manager.
func (r *NodeAutoRepairReconciler) SetupWithManager(mgr ctrl.Manager) error {
	log := zap.L().Sugar()
	log.Info("Setting up NAR controller with manager")
	r.Recorder = mgr.GetEventRecorderFor("node-auto-repair-controller")
	r.leaderElection = mgr.Elected()
	if id, err := os.Hostname(); err == nil {
		r.ControllerID = id
	} else {
		r.ControllerID = "unknown"
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Node{}, builder.WithPredicates(ConditionChangedPredicate{log: log})).
		WithOptions(controller.Options{MaxConcurrentReconciles: getMaxConcurrentRepairs(), CacheSyncTimeout: time.Hour}).
		Complete(r)
}

func getMaxConcurrentRepairs() int {
	const defaultConcurrency = 1
	if val := os.Getenv("NODE_AUTOREPAIR_MAX_CONCURRENCY"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultConcurrency
}

// Reconcile is the main controller loop, now acting as an orchestrator.
// Reconcile is the main controller loop.
func (r *NodeAutoRepairReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx).WithValues(narName, "NAR")

	node := &v1.Node{}
	if err := r.Client.Get(ctx, req.NamespacedName, node); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.ensureControllerID(logger); err != nil {
		logger.Error(err, "CCM: Unable to determine controller identity")
		return ctrl.Result{RequeueAfter: defaultRetryBase}, nil
	}

	unhealthyConditions := findUnhealthyConditions(node)
	if len(unhealthyConditions) == 0 {
		return r.cleanupRepairArtifacts(ctx, logger, node)
	}

	repairEnabled := true
	if !repairEnabled {
		return r.handleUnhealthyNode(ctx, logger, node.DeepCopy(), unhealthyConditions)
	}

	isLeader, err := r.isLeader(ctx, logger)
	if err != nil {
		logger.Error(err, "CCM: failed to determine leader status")
		return ctrl.Result{RequeueAfter: defaultRetryBase}, nil
	}
	if !isLeader {
		logger.Info("CCM: skipping repair because this instance is not the leader")
		return ctrl.Result{RequeueAfter: getRequeueDuration(logger, node)}, nil
	}

	return r.handleUnhealthyNode(ctx, logger, node.DeepCopy(), unhealthyConditions)
}

// currentTerminalRepairState inspects the node's NAR state annotation and reports
// whether it is in a terminal state (Succeeded/Failed). It returns the current
// state string and a boolean indicating terminality.
func currentTerminalRepairState(node *v1.Node) (string, bool) {
    if node == nil || node.Annotations == nil {
        return "", false
    }
    state := node.Annotations[narStateAnnotationKey]
    switch state {
    case string(stateSucceeded), string(stateFailed):
        return state, true
    default:
        return state, false
    }
}

// findUnhealthyConditions checks if a node has any conditions that warrant repair.
// It returns a slice of all matching unhealthy conditions, or an empty slice if the node is healthy.
func findUnhealthyConditions(node *v1.Node) []*v1.NodeCondition {
	var unhealthyConditions []*v1.NodeCondition
	for _, condition := range node.Status.Conditions {
		// Check if the condition type exists in our map and its status matches the expected value ("True").
		if expectedStatus, ok := CONDITIONS[string(condition.Type)]; ok && string(condition.Status) == expectedStatus {
			unhealthyConditions = append(unhealthyConditions, condition.DeepCopy())
		}
	}
	return unhealthyConditions
}

// handleUnhealthyNode performs all actions when a node is found to be unhealthy.
func (r *NodeAutoRepairReconciler) handleUnhealthyNode(ctx context.Context, logger logr.Logger, node *v1.Node, conditions []*v1.NodeCondition) (ctrl.Result, error) {
	// Throttle if the node has been repaired recently (cool-down window)
	node = node.DeepCopy()
	if node.Annotations != nil {
		if ts, ok := node.Annotations[narLastRepairEndAnnotation]; ok && ts != "" {
			if endTime, err := time.Parse(time.RFC3339, ts); err == nil {
				until := endTime.Add(repairCoolDown)
				now := time.Now()
				if now.Before(until) {
					remaining := time.Until(until)
					// Emit event and log, then skip repair attempts during cool-down
					if r.Recorder != nil {
						r.Recorder.Event(node, v1.EventTypeNormal, eventRepairThrottled, fmt.Sprintf("[Node Auto Repair]: Throttled due to recent repair; wait %s before next attempt", remaining.Truncate(time.Second)))
					}
					logger.Info("CCM: Throttling node auto repair due to cool-down window", "node", node.Name, "remaining", remaining)
					return ctrl.Result{RequeueAfter: remaining}, nil
				}
			}
		}
	}

	repairEnabled := true
	if repairEnabled {
		if err := r.ensureLeaseManager(logger); err != nil {
			logger.Error(err, "CCM: failed to initialize repair lease manager")
			return ctrl.Result{RequeueAfter: defaultRetryBase}, nil
		}
		acquired, activeNode, err := r.leaseManager.TryAcquire(ctx, node.Name)
		if err != nil {
			logger.Error(err, "CCM: failed to acquire repair lease")
			return ctrl.Result{RequeueAfter: defaultRetryBase}, nil
		}
		if !acquired {
			if activeNode == "" {
				logger.Info("CCM: Another controller holds the repair lease, waiting")
			} else {
				logger.Info("CCM: Another node is currently under repair", "activeNode", activeNode)
			}
			return ctrl.Result{RequeueAfter: defaultRetryBase}, nil
		}
		if err := r.startLeaseHeartbeat(ctx, node.Name, logger); err != nil {
			logger.Error(err, "CCM: failed to start lease heartbeat")
			return ctrl.Result{RequeueAfter: defaultRetryBase}, nil
		}
		logger.Info("CCM: acquired repair lease", "node", node.Name)
	}

	problemTypes := make([]string, 0, len(conditions))
	for _, cond := range conditions {
		problemTypes = append(problemTypes, string(cond.Type))
	}
	aggregatedLabelValue := strings.Join(problemTypes, ",")

	if err := r.ensureRepairMarkers(ctx, node, aggregatedLabelValue); err != nil {
		logger.Error(err, "CCM: Failed to patch node with repair label/taint")
		return ctrl.Result{}, err
	}

	// Action 4: Trigger the repair action (terminate).
	// if string(condition.Type) == "GPUCountMismatch" && taintFound {
	// 	logger.Info("GPUCountMismatch condition detected on already tainted node. Triggering terminate action.", "node", node.Name)
	// 	workrequestId, _ := r.OCIClient.Compute().TerminateInstance(ctx, node.Spec.ProviderID)
	// 	logger.Info("CCM: Terminate instance workrequest id: " + workrequestId)
	// 	return ctrl.Result{}, nil
	// }

	// Step 2: Record a single, combined event and log message.
	eventMessage := "[Node Auto Repair]: Node conditions are unhealthy: " + strings.Join(problemTypes, ", ")
	if !repairEnabled {
		eventMessage += " (dry run: will not repair)"
	}
	r.Recorder.Event(node, v1.EventTypeWarning, "NodeUnhealthy", eventMessage)
	logger.Info("CCM: Node conditions triggered repair action", "node", node.Name, "conditions", strings.Join(problemTypes, ", "))

	if !repairEnabled {
		requeueDuration := getRequeueDuration(logger, node)
		logger.Info("CCM: Repair disabled via label; requeueing", "node", node.Name, "after", requeueDuration)
		return ctrl.Result{RequeueAfter: requeueDuration}, nil
	}

	repairSM := newNodeRepairStateMachine(r, node, logger)
	return repairSM.Run(ctx)
}

func (r *NodeAutoRepairReconciler) ensureRepairMarkers(ctx context.Context, node *v1.Node, labelValue string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &v1.Node{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: node.Name}, latest); err != nil {
			return err
		}
		updated := latest.DeepCopy()
		if updated.Labels == nil {
			updated.Labels = make(map[string]string)
		}
		if updated.Labels[repairProblemDetectedLabel] != labelValue {
			updated.Labels[repairProblemDetectedLabel] = labelValue
		}
		ta := CreateRepairTaint()
		repairTaintExists := false
		for _, taint := range updated.Spec.Taints {
			if taint.Key == REPAIR_TAINT_KEY && taint.Effect == REPAIR_TAINT_EFFECT {
				repairTaintExists = true
				break
			}
		}
		if !repairTaintExists {
			updated.Spec.Taints = append(updated.Spec.Taints, ta)
		}
		if err := r.Client.Patch(ctx, updated, client.MergeFrom(latest)); err != nil {
			return err
		}
		*node = *updated
		return nil
	})
}

func (r *NodeAutoRepairReconciler) ensureControllerID(logger logr.Logger) error {
	if r.ControllerID != "" {
		return nil
	}
	id, err := os.Hostname()
	if err != nil {
		logger.Error(err, "CCM: failed to resolve controller hostname")
		return fmt.Errorf("controller ID not set: %w", err)
	}
	r.ControllerID = id
	return nil
}

func (r *NodeAutoRepairReconciler) isLeader(ctx context.Context, logger logr.Logger) (bool, error) {
	if r.leaderElection == nil {
		err := errors.New("leader election channel not initialized")
		logger.Error(err, "CCM: leader election channel missing")
		return false, err
	}
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-r.leaderElection:
		return true, nil
	default:
		return false, nil
	}
}

// cleanupRepairArtifacts checks a healthy node for leftover repair items and removes them.
func (r *NodeAutoRepairReconciler) cleanupRepairArtifacts(ctx context.Context, logger logr.Logger, node *v1.Node) (ctrl.Result, error) {
	patch := client.MergeFrom(node.DeepCopy())
	var needsPatch bool

    // Determine if the node bears NAR markers (taints/annotations) so that we can
    // safely auto-uncordon only when NAR had previously acted on this node.
    hasNARTaint := false
    if len(node.Spec.Taints) > 0 {
        for _, t := range node.Spec.Taints {
            if t.Key == REPAIR_TAINT_KEY && t.Effect == REPAIR_TAINT_EFFECT {
                hasNARTaint = true
                break
            }
        }
    }
    hasNARAnnotation := false
    if node.Annotations != nil {
        // Any working-state NAR annotations
        for _, k := range repairAnnotationKeys {
            if _, ok := node.Annotations[k]; ok {
                hasNARAnnotation = true
                break
            }
        }
        // Or terminal summary annotations
        if !hasNARAnnotation {
            if _, ok := node.Annotations[narLastRepairEndAnnotation]; ok {
                hasNARAnnotation = true
            }
        }
        if !hasNARAnnotation {
            if _, ok := node.Annotations[narLastRepairResultAnnotation]; ok {
                hasNARAnnotation = true
            }
        }
    }

    // If the node is healthy (we're in cleanup), and it still remains cordoned, auto-uncordon
    // but only if there are NAR markers indicating the cordon likely originated from NAR.
    if (hasNARTaint || hasNARAnnotation) && node.Spec.Unschedulable {
        logger.Info("Node is healthy; auto-uncordon due to previous NAR markers", "node", node.Name)
        node.Spec.Unschedulable = false
        needsPatch = true
    }

	// 1. Clean up repair labels.
	for key := range node.Labels {
		if strings.HasPrefix(key, repairProblemDetectedLabel) {
			logger.Info("Node is healthy, removing repair label", "node", node.Name, "label", key)
			delete(node.Labels, key)
			needsPatch = true
		}
	}

	// 2. Clean up repair taints.
	var taintsToKeep []v1.Taint
	var taintToRemove bool
	for _, taint := range node.Spec.Taints {
		if taint.Key == REPAIR_TAINT_KEY && taint.Effect == REPAIR_TAINT_EFFECT {
			taintToRemove = true
		} else {
			taintsToKeep = append(taintsToKeep, taint)
		}
	}

	if taintToRemove {
		logger.Info("Node is healthy, removing repair taint", "node", node.Name, "taint", REPAIR_TAINT_KEY)
		node.Spec.Taints = taintsToKeep
		needsPatch = true
	}

	// 3. Remove state machine annotations after a completed repair.
	if keys := nodeAnnotationsToPrune(node); len(keys) > 0 {
		for _, key := range keys {
			delete(node.Annotations, key)
		}
		needsPatch = true
	}

	if needsPatch {
		logger.Info("Applying cleanup patch to node", "node", node.Name)
		if err := r.Client.Patch(ctx, node, patch); err != nil {
			logger.Error(err, "Failed to apply cleanup patch to node")
			return ctrl.Result{}, err
		}
	} else {
		logger.Info("Node is healthy and has no NPD conditions to clean up.", "node", node.Name)
	}
	if err := r.stopLeaseHeartbeat(ctx, node.Name); err != nil {
		logger.Error(err, "Failed to stop lease heartbeat", "node", node.Name)
	}

	return ctrl.Result{}, nil
}

type ConditionChangedPredicate struct {
	predicate.Funcs
	log *zap.SugaredLogger
}

func (p ConditionChangedPredicate) Update(e event.UpdateEvent) bool {
	oldNode, ok := e.ObjectOld.(*v1.Node)
	if !ok {
		return false
	}
	newNode, ok := e.ObjectNew.(*v1.Node)
	if !ok {
		return false
	}
	oldConditions := getConditionMap(oldNode.Status.Conditions)
	newConditions := getConditionMap(newNode.Status.Conditions)
	for _, newCondition := range newConditions {
		oldCondition, exists := oldConditions[newCondition.Type]
		if !exists {
			p.log.Infow("CCM: New condition added to node.", "node", newNode.Name, "conditionType", newCondition.Type, "status", newCondition.Status, "reason", newCondition.Reason)
			return true
		}
		if oldCondition.Status != newCondition.Status || oldCondition.Reason != newCondition.Reason || oldCondition.Message != newCondition.Message {
			p.log.Infow("CCM: Node condition: "+string(newCondition.Type)+" changed", "node", newNode.Name, "conditionType", newCondition.Type, "oldStatus", oldCondition.Status, "newStatus", newCondition.Status, "oldReason", oldCondition.Reason, "newReason", newCondition.Reason, "oldHeartBeatTime", oldCondition.LastHeartbeatTime, "newHeartBeatTime", newCondition.LastHeartbeatTime, "oldTransitTime", oldCondition.LastTransitionTime, "newTransitTime", newCondition.LastHeartbeatTime)
			if _, ok := CONDITIONS[string(oldCondition.Type)]; ok {
				return true
			}
		}
	}
	var lastManager string
	var lastUpdateTime time.Time
	var fieldName string
	for _, field := range newNode.ManagedFields {
		if field.Time.After(lastUpdateTime) {
			lastUpdateTime = field.Time.Time
			lastManager = field.Manager
			fieldName = field.String()
		}
	}
	if lastManager != "" {
		p.log.Info("CCM: NPD Condition hasn't changed. Detected a change in Node object.", " manager: ", lastManager, " updateTime: ", lastUpdateTime, " fieldName:", fieldName)
	} else {
		p.log.Info("CCM: NPD Condition hasn't changed. No manager found for this update event.")
	}
	return false
}

func getConditionMap(conditions []v1.NodeCondition) map[v1.NodeConditionType]v1.NodeCondition {
	conditionMap := make(map[v1.NodeConditionType]v1.NodeCondition, len(conditions))
	for _, condition := range conditions {
		conditionMap[condition.Type] = condition
	}
	return conditionMap
}

func getRequeueDuration(logger logr.Logger, node *v1.Node) time.Duration {
	// Define the default requeue duration
	requeueDuration := 10 * time.Minute

	if val, ok := node.Labels[repairFrequencyLabel]; ok {
		// Parse the string value to an integer (base 10, 64-bit)
		if parsedValue, err := strconv.ParseInt(val, 10, 64); err == nil && parsedValue > 0 {
			// If parsing is successful and the value is positive, use it
			requeueDuration = time.Duration(parsedValue) * time.Minute
		} else {
			// Log a warning if the label value is invalid, but continue with the default
			logger.Info("CCM: Invalid value for repair frequency label on node: " + node.Name + " Frequency Lable has a value: " + val)
		}
	}

	return requeueDuration
}

func (p ConditionChangedPredicate) Create(e event.CreateEvent) bool {
	return true
}

// createRepairTaint creates a new Taint object with a timestamp as its value.
func CreateRepairTaint() v1.Taint {
	// Get the current time and format it as a string
	now := time.Now().UTC()
	const k8sTaintTimeFormat = "2006-01-02-15-04-05"

	// Format using a reference string
	timestamp := now.Format(k8sTaintTimeFormat)

	return v1.Taint{
		Key:    REPAIR_TAINT_KEY,
		Value:  timestamp, // Value is now a dynamic timestamp
		Effect: REPAIR_TAINT_EFFECT,
	}
}

func (r *NodeAutoRepairReconciler) ensureLeaseManager(logger logr.Logger) error {
	if r.leaseManager != nil {
		return nil
	}
	if r.ControllerID == "" {
		err := errors.New("controller ID not initialized")
		logger.Error(err, "CCM: cannot initialize lease manager without controller ID")
		return err
	}
	r.leaseManager = newRepairLeaseManager(r.Client, r.ControllerID)
	return nil
}

func (r *NodeAutoRepairReconciler) releaseLease(ctx context.Context, nodeName string) error {
	if r.leaseManager == nil {
		return nil
	}
	return r.leaseManager.Release(ctx, nodeName)
}

func (r *NodeAutoRepairReconciler) startLeaseHeartbeat(ctx context.Context, nodeName string, logger logr.Logger) error {
	if r.leaseManager == nil {
		err := errors.New("lease manager not initialized")
		logger.Error(err, "CCM: cannot start lease heartbeat without lease manager")
		return err
	}
	if r.leaseStopCh != nil {
		close(r.leaseStopCh)
	}
	stopCh := make(chan struct{})
	r.leaseStopCh = stopCh
	interval := r.leaseManager.renewInterval()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := r.leaseManager.Renew(context.Background(), nodeName); err != nil {
					r.logHeartbeatError(err, logger, nodeName)
				}
			case <-stopCh:
				return
			}
		}
	}()
	return nil
}

func (r *NodeAutoRepairReconciler) stopLeaseHeartbeat(ctx context.Context, nodeName string) error {
	if r.leaseStopCh != nil {
		close(r.leaseStopCh)
		r.leaseStopCh = nil
	}
	return r.releaseLease(ctx, nodeName)
}

func (r *NodeAutoRepairReconciler) logHeartbeatError(err error, logger logr.Logger, nodeName string) {
	logger.Error(err, "CCM: lease heartbeat failed", "node", nodeName)
}
