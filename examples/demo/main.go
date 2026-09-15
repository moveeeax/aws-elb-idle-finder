// Command demo runs the finder against an in-memory fake AWS client so you can
// see the report format without any AWS credentials. It is illustrative only —
// the real CLI lives in cmd/aws-elb-idle-finder.
//
//	go run ./examples/demo
package main

import (
	"context"
	"os"
	"time"

	"github.com/moveeeax/aws-elb-idle-finder/internal/finder"
)

// fakeClient returns a fixed set of load balancers with canned traffic so the
// output is deterministic.
type fakeClient struct{}

func (fakeClient) ListLoadBalancers(context.Context) ([]finder.LoadBalancer, error) {
	return []finder.LoadBalancer{
		{Name: "prod-api", Kind: finder.KindALB, RegisteredTargets: 6, HealthyTargets: 6},
		{Name: "old-staging", Kind: finder.KindALB, RegisteredTargets: 0, HealthyTargets: 0},
		{Name: "batch-nlb", Kind: finder.KindNLB, RegisteredTargets: 2, HealthyTargets: 2},
		{Name: "legacy-clb", Kind: finder.KindCLB, RegisteredTargets: 0, HealthyTargets: 0},
	}, nil
}

func (fakeClient) Traffic(_ context.Context, lb finder.LoadBalancer, _ time.Duration) (float64, error) {
	switch lb.Name {
	case "prod-api":
		return 1_200_000, nil // busy -> ACTIVE
	case "batch-nlb":
		return 3, nil // trickle -> IDLE
	default:
		return -1, nil
	}
}

func main() {
	sc := &finder.Scanner{
		Client: fakeClient{},
		Region: "eu-central-1",
		Now:    func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	rep, err := sc.Scan(context.Background(), 14*24*time.Hour)
	if err != nil {
		panic(err)
	}
	_ = rep.WriteTable(os.Stdout)
}
