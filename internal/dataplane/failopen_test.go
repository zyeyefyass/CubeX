package dataplane

import (
	"context"
	"testing"
	"time"
)

func TestFailOpenController_StateTransitions(t *testing.T) {
	stateCh := make(chan FailOpenState, 2)
	controller := NewFailOpenController(func(state FailOpenState, reason string) {
		stateCh <- state
	})

	if controller.IsFailOpenActive() {
		t.Fatal("expected fail-open to be disabled initially")
	}

	rep := controller.GetStatusReport()
	if rep.Active || rep.CapsuleBadge != "green" {
		t.Fatalf("unexpected initial report: %+v", rep)
	}

	// 触发逃生
	err := controller.TriggerFailOpen(context.Background(), "core crashed")
	if err != nil {
		t.Fatalf("TriggerFailOpen failed: %v", err)
	}

	select {
	case st := <-stateCh:
		if st != FailOpenActive {
			t.Fatalf("expected state FailOpenActive, got %v", st)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for state change callback")
	}

	if !controller.IsFailOpenActive() {
		t.Fatal("expected fail-open to be active")
	}

	rep = controller.GetStatusReport()
	if !rep.Active || rep.CapsuleBadge != "amber" {
		t.Fatalf("unexpected amber report: %+v", rep)
	}

	// 恢复正常
	err = controller.RestoreNormal(context.Background(), "core recovered")
	if err != nil {
		t.Fatalf("RestoreNormal failed: %v", err)
	}

	select {
	case st := <-stateCh:
		if st != FailOpenDisabled {
			t.Fatalf("expected state FailOpenDisabled, got %v", st)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for state change callback")
	}

	if controller.IsFailOpenActive() {
		t.Fatal("expected fail-open to be disabled after restore")
	}
}
