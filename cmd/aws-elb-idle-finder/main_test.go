package main

import (
	"testing"
	"time"
)

func TestParseLookback(t *testing.T) {
	ok := map[string]time.Duration{
		"14d":  14 * 24 * time.Hour,
		"1d":   24 * time.Hour,
		"168h": 168 * time.Hour,
		"30m":  30 * time.Minute,
	}
	for in, want := range ok {
		got, err := parseLookback(in)
		if err != nil {
			t.Errorf("parseLookback(%q) error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseLookback(%q) = %v, want %v", in, got, want)
		}
	}

	for _, bad := range []string{"", "d", "0d", "-3d", "abc", "12x", "0"} {
		if _, err := parseLookback(bad); err == nil {
			t.Errorf("parseLookback(%q) expected error", bad)
		}
	}
}
