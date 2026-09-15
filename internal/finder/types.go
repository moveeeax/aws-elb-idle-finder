// Package finder scans AWS load balancers and classifies idle ones.
package finder

import "time"

// Kind is the load balancer family.
type Kind string

const (
	KindALB Kind = "application" // ELBv2 application load balancer
	KindNLB Kind = "network"     // ELBv2 network load balancer
	KindCLB Kind = "classic"     // classic (v1) load balancer
)

// Status is the idle classification of a load balancer.
type Status string

const (
	// StatusIdle means the LB has no healthy targets, or near-zero traffic over
	// the lookback window — a strong candidate for deletion.
	StatusIdle Status = "IDLE"
	// StatusActive means the LB has healthy targets and measurable traffic.
	StatusActive Status = "ACTIVE"
	// StatusUnknown means we could not collect enough evidence (no metrics
	// returned) to decide; treated as not-idle so we never flag blindly.
	StatusUnknown Status = "UNKNOWN"
)

// LoadBalancer is the normalized view of an ALB/NLB/CLB the scanner works with.
type LoadBalancer struct {
	Name string
	ARN  string // empty for classic LBs
	Kind Kind

	// HealthyTargets is the count of registered targets reporting healthy
	// across all target groups (ELBv2) or registered instances (classic).
	HealthyTargets int
	// RegisteredTargets is the total registered target/instance count.
	RegisteredTargets int

	// Traffic is the summed request/flow count over the lookback window from
	// CloudWatch. -1 means "no datapoints returned" (metrics unavailable).
	Traffic float64
}

// Finding is the classification result for one load balancer.
type Finding struct {
	Name            string  `json:"name"`
	ARN             string  `json:"arn,omitempty"`
	Kind            Kind    `json:"kind"`
	Status          Status  `json:"status"`
	Reason          string  `json:"reason"`
	HealthyTargets  int     `json:"healthy_targets"`
	Traffic         float64 `json:"traffic"`
	MonthlyWasteUSD float64 `json:"monthly_waste_usd"`
}

// Report is the full scan output.
type Report struct {
	Region        string    `json:"region"`
	Lookback      string    `json:"lookback"`
	GeneratedAt   time.Time `json:"generated_at"`
	Findings      []Finding `json:"findings"`
	IdleCount     int       `json:"idle_count"`
	TotalWasteUSD float64   `json:"total_monthly_waste_usd"`
}
