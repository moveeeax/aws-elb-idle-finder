package finder

import "fmt"

// trafficThreshold is the summed request/flow count below which an LB with
// registered targets is still considered idle. A handful of health-check or
// scanner hits over two weeks should not save an otherwise-dead LB.
const trafficThreshold = 100.0

// Classify decides whether a single load balancer is idle, and why.
//
// Rules, in order:
//  1. No healthy targets registered  -> IDLE (nothing can serve traffic).
//  2. Metrics unavailable (Traffic<0) -> UNKNOWN (never flag without evidence).
//  3. Traffic below the threshold     -> IDLE (targets exist but nobody calls).
//  4. Otherwise                       -> ACTIVE.
//
// region is used only to price the waste estimate for idle results.
func Classify(lb LoadBalancer, region string) Finding {
	f := Finding{
		Name:           lb.Name,
		ARN:            lb.ARN,
		Kind:           lb.Kind,
		HealthyTargets: lb.HealthyTargets,
		Traffic:        lb.Traffic,
	}

	switch {
	case lb.HealthyTargets == 0:
		f.Status = StatusIdle
		if lb.RegisteredTargets == 0 {
			f.Reason = "no registered targets"
		} else {
			f.Reason = fmt.Sprintf("%d registered targets, none healthy", lb.RegisteredTargets)
		}
		f.MonthlyWasteUSD = MonthlyWaste(lb.Kind, region)
	case lb.Traffic < 0:
		f.Status = StatusUnknown
		f.Reason = "no CloudWatch datapoints for lookback window"
	case lb.Traffic < trafficThreshold:
		f.Status = StatusIdle
		f.Reason = fmt.Sprintf("near-zero traffic (%.0f over lookback)", lb.Traffic)
		f.MonthlyWasteUSD = MonthlyWaste(lb.Kind, region)
	default:
		f.Status = StatusActive
		f.Reason = fmt.Sprintf("%d healthy targets, %.0f traffic", lb.HealthyTargets, lb.Traffic)
	}
	return f
}
