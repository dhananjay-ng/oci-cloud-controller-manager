package controllers

import (
    "context"
    "testing"
    "time"

    "github.com/go-logr/logr"
    v1 "k8s.io/api/core/v1"
)

// Test that handleUnhealthyNode throttles when last-repair-end is within cooldown window
func TestHandleUnhealthyNode_Throttled(t *testing.T) {
	now := time.Now().UTC()
	node := &v1.Node{}
	node.Name = "nar-throttle-node"
	node.Annotations = map[string]string{
		narLastRepairEndAnnotation: now.Add(-30 * time.Minute).Format(time.RFC3339),
	}
	// a sample unhealthy condition (type doesn't matter for this direct call)
	cond := &v1.NodeCondition{Type: v1.NodeReady, Status: v1.ConditionFalse}

    r := &NodeAutoRepairReconciler{}
    logger := logr.Discard()

	res, err := r.handleUnhealthyNode(context.Background(), logger, node, []*v1.NodeCondition{cond})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.RequeueAfter <= 0 {
		t.Fatalf("expected positive RequeueAfter due to throttling, got %v", res.RequeueAfter)
	}
	// remaining should be less than cooldown (default 60m)
	if res.RequeueAfter >= repairCoolDown {
		t.Fatalf("expected RequeueAfter < cooldown %v, got %v", repairCoolDown, res.RequeueAfter)
	}
}

// Test that cleanup logic will not propose pruning last-repair-end annotation
func TestNodeAnnotationsToPrune_DoesNotIncludeLastRepairEnd(t *testing.T) {
    node := &v1.Node{}
    node.Name = "nar-cleanup-node"
    node.Annotations = map[string]string{
        narStateAnnotationKey:         "Draining",
        narRepairIDAnnotationKey:      "rid-123",
        narRepairOriginAnnotationKey:  "controller-a",
        narLastTransitionAnnotation:   time.Now().UTC().Format(time.RFC3339),
        narAttemptsAnnotationKey:      "2",
        narRebootIssuedAnnotationKey:  "true",
        narStateMetadataAnnotationKey: "{}",
        narLastRepairEndAnnotation:    time.Now().UTC().Format(time.RFC3339),
    }

    keys := nodeAnnotationsToPrune(node)
    // Ensure last-repair-end is not among pruned keys
    for _, k := range keys {
        if k == narLastRepairEndAnnotation {
            t.Fatalf("narLastRepairEndAnnotation should not be pruned, but found in prune set")
        }
    }
    // And ensure working-state annotations are included
    want := map[string]bool{
        narStateAnnotationKey:         true,
        narRepairIDAnnotationKey:      true,
        narRepairOriginAnnotationKey:  true,
        narLastTransitionAnnotation:   true,
        narAttemptsAnnotationKey:      true,
        narRebootIssuedAnnotationKey:  true,
        narStateMetadataAnnotationKey: true,
    }
    for _, k := range keys {
        delete(want, k)
    }
    if len(want) != 0 {
        t.Fatalf("expected working-state annotations to be pruned, missing: %v", want)
    }
}

