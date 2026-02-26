package controllers

import (
    "testing"
    "time"

    coordinationv1 "k8s.io/api/coordination/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLeaseExpired(t *testing.T) {
    m := &repairLeaseManager{leaseDuration: 2 * time.Minute}

    // nil lease -> expired
    if !m.leaseExpired(nil) {
        t.Fatalf("nil lease should be considered expired")
    }

    // missing duration -> expired
    l1 := &coordinationv1.Lease{Spec: coordinationv1.LeaseSpec{}}
    if !m.leaseExpired(l1) {
        t.Fatalf("lease without duration should be expired")
    }

    // with duration but no renew time -> expired
    dur := int32(120)
    l2 := &coordinationv1.Lease{Spec: coordinationv1.LeaseSpec{LeaseDurationSeconds: &dur}}
    if !m.leaseExpired(l2) {
        t.Fatalf("lease without renew time should be expired")
    }

    // renew time within duration -> not expired
    now := metav1.NewMicroTime(time.Now())
    l3 := &coordinationv1.Lease{Spec: coordinationv1.LeaseSpec{LeaseDurationSeconds: &dur, RenewTime: &now}}
    if m.leaseExpired(l3) {
        t.Fatalf("lease renewed just now should not be expired")
    }

    // renew time beyond duration -> expired
    past := metav1.NewMicroTime(time.Now().Add(-3 * time.Minute))
    l4 := &coordinationv1.Lease{Spec: coordinationv1.LeaseSpec{LeaseDurationSeconds: &dur, RenewTime: &past}}
    if !m.leaseExpired(l4) {
        t.Fatalf("lease with old renew time should be expired")
    }
}

func TestRenewIntervalFloor(t *testing.T) {
    // very small duration yields floor 30s interval
    m1 := &repairLeaseManager{leaseDuration: 10 * time.Second}
    if got := m1.renewInterval(); got != 30*time.Second {
        t.Fatalf("expected 30s floor, got %v", got)
    }
    // normal duration (e.g., 3m) -> 1m
    m2 := &repairLeaseManager{leaseDuration: 3 * time.Minute}
    if got := m2.renewInterval(); got < 59*time.Second || got > 61*time.Second {
        t.Fatalf("expected ~1m renew interval, got %v", got)
    }
}
