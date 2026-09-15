// Command aws-elb-idle-finder scans a region's load balancers and flags idle
// ones with an estimated monthly waste. Exit code 1 when idle LBs are found, so
// it doubles as a CI cost gate.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/moveeeax/aws-elb-idle-finder/internal/finder"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return // -h already printed usage; clean exit
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
}

func run(args []string, stdout, stderr *os.File) error {
	fs := flag.NewFlagSet("aws-elb-idle-finder", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		region   = fs.String("region", "", "AWS region to scan (default: from AWS config/env)")
		lookback = fs.String("lookback", "14d", "traffic lookback window (e.g. 14d, 168h)")
		asJSON   = fs.Bool("json", false, "emit JSON instead of a table")
		noExit   = fs.Bool("no-exit-code", false, "always exit 0 even when idle LBs are found")
	)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: aws-elb-idle-finder [flags]\n\nScan ALB/NLB/CLB load balancers and flag idle ones.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	dur, err := parseLookback(*lookback)
	if err != nil {
		return err
	}

	ctx := context.Background()
	opts := []func(*config.LoadOptions) error{}
	if *region != "" {
		opts = append(opts, config.WithRegion(*region))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}
	if cfg.Region == "" {
		return fmt.Errorf("no region: pass --region or set AWS_REGION")
	}

	sc := &finder.Scanner{Client: finder.NewAWSClient(cfg), Region: cfg.Region}
	rep, err := sc.Scan(ctx, dur)
	if err != nil {
		return err
	}

	if *asJSON {
		if err := rep.WriteJSON(stdout); err != nil {
			return err
		}
	} else {
		if err := rep.WriteTable(stdout); err != nil {
			return err
		}
	}

	if rep.IdleCount > 0 && !*noExit {
		os.Exit(1)
	}
	return nil
}

// parseLookback accepts a Go duration or a "<n>d" days shorthand.
func parseLookback(s string) (time.Duration, error) {
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s[:len(s)-1], "%d", &days); err != nil || days <= 0 {
			return 0, fmt.Errorf("invalid lookback %q", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid lookback %q", s)
	}
	return d, nil
}
