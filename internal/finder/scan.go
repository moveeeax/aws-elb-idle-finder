package finder

import (
	"context"
	"sort"
	"time"
)

// Client is the narrow AWS surface the scanner depends on. It is implemented by
// the real aws-sdk-go-v2 wrapper in awsclient.go and by mocks in tests, so the
// scan logic never touches live AWS.
type Client interface {
	// ListLoadBalancers returns every ALB/NLB/CLB in the region with its
	// registered/healthy target counts already resolved.
	ListLoadBalancers(ctx context.Context) ([]LoadBalancer, error)
	// Traffic returns the summed request/flow count for a load balancer over
	// the lookback window, or -1 if CloudWatch returned no datapoints.
	Traffic(ctx context.Context, lb LoadBalancer, lookback time.Duration) (float64, error)
}

// Scanner ties a Client to a region and clock and produces Reports.
type Scanner struct {
	Client Client
	Region string
	Now    func() time.Time // injectable clock; defaults to time.Now
}

// Scan enumerates load balancers, gathers traffic, classifies each, and returns
// a fully populated Report sorted by descending monthly waste then name.
func (s *Scanner) Scan(ctx context.Context, lookback time.Duration) (Report, error) {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}

	lbs, err := s.Client.ListLoadBalancers(ctx)
	if err != nil {
		return Report{}, err
	}

	rep := Report{
		Region:      s.Region,
		Lookback:    lookback.String(),
		GeneratedAt: now(),
	}

	for _, lb := range lbs {
		// Only spend a CloudWatch call when targets exist — a no-target LB is
		// already idle and its traffic is irrelevant to the verdict.
		if lb.HealthyTargets > 0 {
			t, err := s.Client.Traffic(ctx, lb, lookback)
			if err != nil {
				return Report{}, err
			}
			lb.Traffic = t
		}
		f := Classify(lb, s.Region)
		rep.Findings = append(rep.Findings, f)
		if f.Status == StatusIdle {
			rep.IdleCount++
			rep.TotalWasteUSD += f.MonthlyWasteUSD
		}
	}

	sort.SliceStable(rep.Findings, func(i, j int) bool {
		if rep.Findings[i].MonthlyWasteUSD != rep.Findings[j].MonthlyWasteUSD {
			return rep.Findings[i].MonthlyWasteUSD > rep.Findings[j].MonthlyWasteUSD
		}
		return rep.Findings[i].Name < rep.Findings[j].Name
	})

	return rep, nil
}
