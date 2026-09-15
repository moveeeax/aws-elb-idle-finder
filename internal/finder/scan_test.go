package finder

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockClient is an in-memory Client for deterministic scan tests.
type mockClient struct {
	lbs     []LoadBalancer
	traffic map[string]float64
	listErr error
}

func (m *mockClient) ListLoadBalancers(context.Context) ([]LoadBalancer, error) {
	return m.lbs, m.listErr
}

func (m *mockClient) Traffic(_ context.Context, lb LoadBalancer, _ time.Duration) (float64, error) {
	if v, ok := m.traffic[lb.Name]; ok {
		return v, nil
	}
	return -1, nil
}

func fixedNow() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

func TestScanReport(t *testing.T) {
	m := &mockClient{
		lbs: []LoadBalancer{
			{Name: "dead", Kind: KindALB, RegisteredTargets: 0, HealthyTargets: 0},
			{Name: "quiet", Kind: KindNLB, RegisteredTargets: 2, HealthyTargets: 2},
			{Name: "busy", Kind: KindALB, RegisteredTargets: 3, HealthyTargets: 3},
			{Name: "clb-empty", Kind: KindCLB, RegisteredTargets: 0, HealthyTargets: 0},
		},
		traffic: map[string]float64{
			"quiet": 5,     // below threshold -> idle
			"busy":  99999, // active
		},
	}
	sc := &Scanner{Client: m, Region: "us-east-1", Now: fixedNow}

	rep, err := sc.Scan(context.Background(), 14*24*time.Hour)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if rep.IdleCount != 3 {
		t.Errorf("idle count = %d, want 3", rep.IdleCount)
	}
	if !rep.GeneratedAt.Equal(fixedNow()) {
		t.Errorf("GeneratedAt = %v, want injected clock", rep.GeneratedAt)
	}
	if rep.TotalWasteUSD <= 0 {
		t.Errorf("total waste should be positive, got %.2f", rep.TotalWasteUSD)
	}

	// Findings must be sorted by descending waste; the active "busy" LB (zero
	// waste) sorts last.
	last := rep.Findings[len(rep.Findings)-1]
	if last.Name != "busy" || last.Status != StatusActive {
		t.Errorf("expected active 'busy' last, got %s (%s)", last.Name, last.Status)
	}
	for i := 1; i < len(rep.Findings); i++ {
		if rep.Findings[i-1].MonthlyWasteUSD < rep.Findings[i].MonthlyWasteUSD {
			t.Errorf("findings not sorted by descending waste at %d", i)
		}
	}
}

func TestScanListError(t *testing.T) {
	m := &mockClient{listErr: errors.New("boom")}
	sc := &Scanner{Client: m, Region: "us-east-1", Now: fixedNow}
	if _, err := sc.Scan(context.Background(), time.Hour); err == nil {
		t.Fatal("expected error from ListLoadBalancers")
	}
}
