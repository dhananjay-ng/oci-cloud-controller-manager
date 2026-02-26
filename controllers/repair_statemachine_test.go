package controllers

import (
    "testing"
    "time"

    v1 "k8s.io/api/core/v1"
)

func TestRetryDelayBackoffAndCap(t *testing.T) {
    // Base from env default is 10s; cap is 5m by default
    sm := &nodeRepairStateMachine{node: &v1.Node{}}
    // verify exponential growth and cap
    tests := []struct{
        attempt int
        min     time.Duration
        max     time.Duration
    }{
        {1, 10 * time.Second, 10 * time.Second},
        {2, 20 * time.Second, 20 * time.Second},
        {3, 40 * time.Second, 40 * time.Second},
        {4, 80 * time.Second, 80 * time.Second},
        {5, 160 * time.Second, 160 * time.Second},
        // shifted attempts beyond 5 should cap at base<<5 = 320s but then overall cap 300s applies
        {6, 300 * time.Second, 300 * time.Second},
        {10, 300 * time.Second, 300 * time.Second},
    }
    for _, tt := range tests {
        d := sm.retryDelay(stateCordoning, tt.attempt)
        if d < tt.min || d > tt.max {
            t.Fatalf("attempt %d expected %v..%v got %v", tt.attempt, tt.min, tt.max, d)
        }
    }
}

func TestStateTimedOut_TrueWhenExceedsTimeout(t *testing.T) {
    sm := &nodeRepairStateMachine{node: &v1.Node{}}
    sm.tracker = newRepairStateTracker("rid-1")
    entry := sm.tracker.begin(stateDraining)
    // backdate start time to exceed default draining timeout (10m)
    entry.StartTime = time.Now().UTC().Add(-20 * time.Minute).Format(time.RFC3339)
    if !sm.stateTimedOut(stateDraining) {
        t.Fatalf("expected draining state to be timed out")
    }
}
