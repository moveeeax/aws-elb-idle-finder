package finder

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func sampleReport() Report {
	return Report{
		Region:      "us-east-1",
		Lookback:    "336h0m0s",
		GeneratedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Findings: []Finding{
			{Name: "dead", Kind: KindALB, Status: StatusIdle, Reason: "no registered targets", MonthlyWasteUSD: 16.42, Traffic: -1},
			{Name: "busy", Kind: KindALB, Status: StatusActive, Reason: "4 healthy targets", HealthyTargets: 4, Traffic: 5000},
		},
		IdleCount:     1,
		TotalWasteUSD: 16.42,
	}
}

func TestWriteTable(t *testing.T) {
	var buf bytes.Buffer
	if err := sampleReport().WriteTable(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"STATUS", "IDLE", "dead", "$16.42", "1 idle load balancer"} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing %q\n%s", want, out)
		}
	}
	// A -1 traffic sentinel must render as "-", never a negative number.
	if strings.Contains(out, "-1.00") || strings.Contains(out, "\t-1\t") {
		t.Errorf("negative traffic leaked into table:\n%s", out)
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := sampleReport().WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	var round Report
	if err := json.Unmarshal(buf.Bytes(), &round); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if round.IdleCount != 1 || len(round.Findings) != 2 {
		t.Errorf("round-trip mismatch: %+v", round)
	}
}
