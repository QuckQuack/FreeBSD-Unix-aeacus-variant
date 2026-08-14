package releaseflow

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRun_stops_before_cleanup_when_setup_fails(t *testing.T) {
	// Given
	wantErr := errors.New("service failed")
	var calls []string
	stages := []Stage{
		{Name: "desktop", Run: func() error {
			calls = append(calls, "desktop")
			return nil
		}},
		{Name: "service", Run: func() error {
			calls = append(calls, "service")
			return wantErr
		}},
		{Name: "cleanup", Run: func() error {
			calls = append(calls, "cleanup")
			return nil
		}},
	}

	// When
	err := Run(stages)

	// Then
	if !reflect.DeepEqual(calls, []string{"desktop", "service"}) {
		t.Fatalf("Run() calls = %v, want setup stages through first failure", calls)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), "service") {
		t.Fatalf("Run() error = %q, want failing stage name", err)
	}
}
