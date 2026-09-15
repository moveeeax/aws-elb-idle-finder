package finder

import "strings"

// arnSuffix extracts the CloudWatch LoadBalancer dimension value from an ELBv2
// ARN. For an ARN ".../loadbalancer/app/my-lb/50dc6c495c0c9188" it returns
// "app/my-lb/50dc6c495c0c9188" — the substring after "loadbalancer/".
func arnSuffix(arn string) string {
	const marker = ":loadbalancer/"
	if i := strings.Index(arn, marker); i >= 0 {
		return arn[i+len(marker):]
	}
	return arn
}
