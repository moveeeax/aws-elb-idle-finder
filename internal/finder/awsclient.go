package finder

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	elbv1 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

// elbv2API, elbv1API and cwAPI are the SDK method subsets we use, so the wrapper
// itself stays testable and the deps stay explicit.
type elbv2API interface {
	DescribeLoadBalancers(context.Context, *elbv2.DescribeLoadBalancersInput, ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error)
	DescribeTargetGroups(context.Context, *elbv2.DescribeTargetGroupsInput, ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error)
	DescribeTargetHealth(context.Context, *elbv2.DescribeTargetHealthInput, ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error)
}

type elbv1API interface {
	DescribeLoadBalancers(context.Context, *elbv1.DescribeLoadBalancersInput, ...func(*elbv1.Options)) (*elbv1.DescribeLoadBalancersOutput, error)
}

type cwAPI interface {
	GetMetricStatistics(context.Context, *cloudwatch.GetMetricStatisticsInput, ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricStatisticsOutput, error)
}

// AWSClient implements Client against real AWS services.
type AWSClient struct {
	elbv2 elbv2API
	elbv1 elbv1API
	cw    cwAPI
}

// NewAWSClient builds an AWSClient from a loaded aws.Config.
func NewAWSClient(cfg aws.Config) *AWSClient {
	return &AWSClient{
		elbv2: elbv2.NewFromConfig(cfg),
		elbv1: elbv1.NewFromConfig(cfg),
		cw:    cloudwatch.NewFromConfig(cfg),
	}
}

// ListLoadBalancers enumerates ELBv2 (ALB/NLB) and classic LBs with resolved
// registered/healthy target counts.
func (c *AWSClient) ListLoadBalancers(ctx context.Context) ([]LoadBalancer, error) {
	var out []LoadBalancer

	v2, err := c.listV2(ctx)
	if err != nil {
		return nil, err
	}
	out = append(out, v2...)

	v1, err := c.listV1(ctx)
	if err != nil {
		return nil, err
	}
	out = append(out, v1...)

	return out, nil
}

func (c *AWSClient) listV2(ctx context.Context) ([]LoadBalancer, error) {
	var out []LoadBalancer
	var marker *string
	for {
		page, err := c.elbv2.DescribeLoadBalancers(ctx, &elbv2.DescribeLoadBalancersInput{Marker: marker})
		if err != nil {
			return nil, err
		}
		for _, lb := range page.LoadBalancers {
			kind := KindALB
			if lb.Type == elbv2types.LoadBalancerTypeEnumNetwork {
				kind = KindNLB
			}
			reg, healthy, err := c.v2Targets(ctx, aws.ToString(lb.LoadBalancerArn))
			if err != nil {
				return nil, err
			}
			out = append(out, LoadBalancer{
				Name:              aws.ToString(lb.LoadBalancerName),
				ARN:               aws.ToString(lb.LoadBalancerArn),
				Kind:              kind,
				RegisteredTargets: reg,
				HealthyTargets:    healthy,
				Traffic:           -1,
			})
		}
		if page.NextMarker == nil {
			break
		}
		marker = page.NextMarker
	}
	return out, nil
}

func (c *AWSClient) v2Targets(ctx context.Context, lbARN string) (registered, healthy int, err error) {
	tgs, err := c.elbv2.DescribeTargetGroups(ctx, &elbv2.DescribeTargetGroupsInput{LoadBalancerArn: aws.String(lbARN)})
	if err != nil {
		return 0, 0, err
	}
	for _, tg := range tgs.TargetGroups {
		th, err := c.elbv2.DescribeTargetHealth(ctx, &elbv2.DescribeTargetHealthInput{TargetGroupArn: tg.TargetGroupArn})
		if err != nil {
			return 0, 0, err
		}
		for _, d := range th.TargetHealthDescriptions {
			registered++
			if d.TargetHealth != nil && d.TargetHealth.State == elbv2types.TargetHealthStateEnumHealthy {
				healthy++
			}
		}
	}
	return registered, healthy, nil
}

func (c *AWSClient) listV1(ctx context.Context) ([]LoadBalancer, error) {
	var out []LoadBalancer
	var marker *string
	for {
		page, err := c.elbv1.DescribeLoadBalancers(ctx, &elbv1.DescribeLoadBalancersInput{Marker: marker})
		if err != nil {
			return nil, err
		}
		for _, lb := range page.LoadBalancerDescriptions {
			// Classic LBs expose only registered instances via the API; without
			// per-instance health here we treat registered as the healthy proxy
			// and rely on traffic to catch the near-zero case.
			reg := len(lb.Instances)
			out = append(out, LoadBalancer{
				Name:              aws.ToString(lb.LoadBalancerName),
				Kind:              KindCLB,
				RegisteredTargets: reg,
				HealthyTargets:    reg,
				Traffic:           -1,
			})
		}
		if page.NextMarker == nil {
			break
		}
		marker = page.NextMarker
	}
	return out, nil
}

// Traffic sums the appropriate CloudWatch counter over the lookback window.
func (c *AWSClient) Traffic(ctx context.Context, lb LoadBalancer, lookback time.Duration) (float64, error) {
	namespace, metric, dimName, dimValue := metricSpec(lb)
	if metric == "" {
		return -1, nil
	}
	end := time.Now()
	start := end.Add(-lookback)
	res, err := c.cw.GetMetricStatistics(ctx, &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metric),
		Dimensions: []cwtypes.Dimension{{Name: aws.String(dimName), Value: aws.String(dimValue)}},
		StartTime:  aws.Time(start),
		EndTime:    aws.Time(end),
		Period:     aws.Int32(86400),
		Statistics: []cwtypes.Statistic{cwtypes.StatisticSum},
	})
	if err != nil {
		return 0, err
	}
	if len(res.Datapoints) == 0 {
		return -1, nil
	}
	var total float64
	for _, dp := range res.Datapoints {
		total += aws.ToFloat64(dp.Sum)
	}
	return total, nil
}

// metricSpec returns the CloudWatch namespace/metric/dimension for a LB's
// traffic counter. dimValue for ELBv2 is the ARN suffix ("app/name/id").
func metricSpec(lb LoadBalancer) (namespace, metric, dimName, dimValue string) {
	switch lb.Kind {
	case KindALB:
		return "AWS/ApplicationELB", "RequestCount", "LoadBalancer", arnSuffix(lb.ARN)
	case KindNLB:
		return "AWS/NetworkELB", "ActiveFlowCount", "LoadBalancer", arnSuffix(lb.ARN)
	case KindCLB:
		return "AWS/ELB", "RequestCount", "LoadBalancerName", lb.Name
	default:
		return "", "", "", ""
	}
}
