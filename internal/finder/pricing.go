package finder

// hoursPerMonth is the AWS convention for monthly billing estimates (730h).
const hoursPerMonth = 730.0

// basePricePerHour is the hourly base charge for a load balancer by family and
// region. These are the fixed hourly ELB charges (LCU/capacity-unit costs are
// usage-based and near-zero for an idle LB, so we intentionally exclude them —
// the point is the money burned by an LB doing nothing).
//
// Values are approximate us-east-1 on-demand list prices as of 2024; override
// per region with the map below.
var basePricePerHour = map[Kind]float64{
	KindALB: 0.0225,
	KindNLB: 0.0225,
	KindCLB: 0.025,
}

// regionMultiplier scales the base price for a handful of pricier regions.
// Anything not listed uses 1.0 (us-east-1 baseline). This is deliberately a
// coarse static table — an idle-LB estimate does not need penny accuracy.
var regionMultiplier = map[string]float64{
	"us-east-1":      1.00,
	"us-east-2":      1.00,
	"us-west-1":      1.00,
	"us-west-2":      1.00,
	"eu-west-1":      1.00,
	"eu-central-1":   1.09,
	"eu-north-1":     1.00,
	"ap-southeast-1": 1.09,
	"ap-southeast-2": 1.11,
	"ap-northeast-1": 1.11,
	"ap-south-1":     1.00,
	"sa-east-1":      1.50,
}

// MonthlyWaste returns the estimated monthly USD cost of running a load balancer
// of the given kind in the given region. Only idle LBs should be passed here —
// for an active LB this figure is not "waste", just the running cost.
func MonthlyWaste(kind Kind, region string) float64 {
	base, ok := basePricePerHour[kind]
	if !ok {
		base = basePricePerHour[KindALB]
	}
	mult, ok := regionMultiplier[region]
	if !ok {
		mult = 1.0
	}
	return base * mult * hoursPerMonth
}
