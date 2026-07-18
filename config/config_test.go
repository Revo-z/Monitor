package config

import (
	"testing"
	"time"
)

func TestDefaultInterval(t *testing.T) {
	if DefaultInterval != 0.5 {
		t.Errorf("DefaultInterval = %v, want 0.5", DefaultInterval)
	}
}

func TestIntervalRange(t *testing.T) {
	if MinInterval != 0.01 {
		t.Errorf("MinInterval = %v, want 0.01", MinInterval)
	}
	if MaxInterval != 2.0 {
		t.Errorf("MaxInterval = %v, want 2.0", MaxInterval)
	}
	if MinInterval >= MaxInterval {
		t.Errorf("MinInterval (%v) >= MaxInterval (%v)", MinInterval, MaxInterval)
	}
}

func TestTimeoutPositive(t *testing.T) {
	if Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", Timeout)
	}
	if Timeout <= 0 {
		t.Errorf("Timeout must be positive, got %v", Timeout)
	}
}

func TestEndMarker(t *testing.T) {
	if EndMarker != '#' {
		t.Errorf("EndMarker = %c, want '#'", EndMarker)
	}
}

func TestShortcutCommands(t *testing.T) {
	if len(ShortcutCommands) != 5 {
		t.Fatalf("len(ShortcutCommands) = %d, want 5", len(ShortcutCommands))
	}
	for i, sc := range ShortcutCommands {
		if sc.Label == "" {
			t.Errorf("ShortcutCommands[%d].Label is empty", i)
		}
		if sc.Cmd == "" {
			t.Errorf("ShortcutCommands[%d].Cmd is empty", i)
		}
	}
}

func TestWindowDefaults(t *testing.T) {
	if AppTitle == "" {
		t.Error("AppTitle must not be empty")
	}
	if DefaultWidth <= 0 {
		t.Errorf("DefaultWidth = %d, must be positive", DefaultWidth)
	}
	if DefaultHeight <= 0 {
		t.Errorf("DefaultHeight = %d, must be positive", DefaultHeight)
	}
}
