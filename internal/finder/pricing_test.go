package finder

import "testing"

func TestMonthlyWaste(t *testing.T) {
	// us-east-1 ALB baseline: 0.0225 * 730 = 16.425
	got := MonthlyWaste(KindALB, "us-east-1")
	if got < 16.0 || got > 17.0 {
		t.Errorf("us-east-1 ALB waste = %.3f, want ~16.4", got)
	}

	// eu-central-1 carries a >1 multiplier, so it must exceed baseline.
	if MonthlyWaste(KindALB, "eu-central-1") <= got {
		t.Error("eu-central-1 should price above us-east-1")
	}

	// Unknown region falls back to 1.0 multiplier (== baseline).
	if MonthlyWaste(KindALB, "moon-base-1") != got {
		t.Error("unknown region should use baseline multiplier")
	}

	// Classic LBs are pricier per hour than ALBs at baseline.
	if MonthlyWaste(KindCLB, "us-east-1") <= got {
		t.Error("classic LB should price above ALB at baseline")
	}
}
