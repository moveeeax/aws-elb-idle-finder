package finder

import "testing"

func TestClassify(t *testing.T) {
	tests := []struct {
		name       string
		lb         LoadBalancer
		wantStatus Status
		wantWaste  bool // expect a non-zero waste estimate
	}{
		{
			name:       "no registered targets is idle",
			lb:         LoadBalancer{Name: "dead-alb", Kind: KindALB, RegisteredTargets: 0, HealthyTargets: 0, Traffic: -1},
			wantStatus: StatusIdle,
			wantWaste:  true,
		},
		{
			name:       "registered but none healthy is idle",
			lb:         LoadBalancer{Name: "unhealthy-alb", Kind: KindALB, RegisteredTargets: 3, HealthyTargets: 0, Traffic: -1},
			wantStatus: StatusIdle,
			wantWaste:  true,
		},
		{
			name:       "healthy targets but no metrics is unknown",
			lb:         LoadBalancer{Name: "no-metrics", Kind: KindNLB, RegisteredTargets: 2, HealthyTargets: 2, Traffic: -1},
			wantStatus: StatusUnknown,
			wantWaste:  false,
		},
		{
			name:       "healthy targets zero traffic is idle",
			lb:         LoadBalancer{Name: "quiet-alb", Kind: KindALB, RegisteredTargets: 2, HealthyTargets: 2, Traffic: 0},
			wantStatus: StatusIdle,
			wantWaste:  true,
		},
		{
			name:       "healthy targets low traffic is idle",
			lb:         LoadBalancer{Name: "trickle-alb", Kind: KindALB, RegisteredTargets: 1, HealthyTargets: 1, Traffic: trafficThreshold - 1},
			wantStatus: StatusIdle,
			wantWaste:  true,
		},
		{
			name:       "healthy targets real traffic is active",
			lb:         LoadBalancer{Name: "busy-alb", Kind: KindALB, RegisteredTargets: 4, HealthyTargets: 4, Traffic: 50000},
			wantStatus: StatusActive,
			wantWaste:  false,
		},
		{
			name:       "traffic exactly at threshold is active",
			lb:         LoadBalancer{Name: "edge-alb", Kind: KindALB, RegisteredTargets: 1, HealthyTargets: 1, Traffic: trafficThreshold},
			wantStatus: StatusActive,
			wantWaste:  false,
		},
		{
			name:       "classic lb no instances is idle",
			lb:         LoadBalancer{Name: "old-clb", Kind: KindCLB, RegisteredTargets: 0, HealthyTargets: 0, Traffic: -1},
			wantStatus: StatusIdle,
			wantWaste:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.lb, "us-east-1")
			if got.Status != tt.wantStatus {
				t.Fatalf("status = %s, want %s (reason: %s)", got.Status, tt.wantStatus, got.Reason)
			}
			if tt.wantWaste && got.MonthlyWasteUSD <= 0 {
				t.Errorf("expected non-zero waste, got %.4f", got.MonthlyWasteUSD)
			}
			if !tt.wantWaste && got.MonthlyWasteUSD != 0 {
				t.Errorf("expected zero waste, got %.4f", got.MonthlyWasteUSD)
			}
			if got.Reason == "" {
				t.Error("reason must not be empty")
			}
		})
	}
}
