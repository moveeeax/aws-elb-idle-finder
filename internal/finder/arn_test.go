package finder

import "testing"

func TestArnSuffix(t *testing.T) {
	tests := map[string]string{
		"arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/my-lb/50dc6c495c0c9188": "app/my-lb/50dc6c495c0c9188",
		"arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/nlb/abc":                "net/nlb/abc",
		"no-marker-here": "no-marker-here",
	}
	for in, want := range tests {
		if got := arnSuffix(in); got != want {
			t.Errorf("arnSuffix(%q) = %q, want %q", in, got, want)
		}
	}
}
