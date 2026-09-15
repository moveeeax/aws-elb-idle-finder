# aws-elb-idle-finder

[![ci](https://github.com/moveeeax/aws-elb-idle-finder/actions/workflows/ci.yml/badge.svg)](https://github.com/moveeeax/aws-elb-idle-finder/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/go-1.24+-00ADD8?logo=go)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

Scan a region's load balancers (ALB / NLB / classic ELB) and flag the **idle**
ones — no healthy targets, or near-zero traffic — with an estimated **monthly $
waste** so you can delete them with evidence in hand.

An ALB or NLB with no healthy targets still bills ~$16/mo plus LCU. They pile up
from torn-down stacks nobody cleaned up. This finds them.

## What it does

For every load balancer in a region it:

- resolves registered + healthy target counts (target-group health for ELBv2,
  registered instances for classic),
- pulls the traffic counter from CloudWatch over a lookback window
  (`RequestCount` for ALB/CLB, `ActiveFlowCount` for NLB),
- classifies each as **IDLE**, **ACTIVE**, or **UNKNOWN**, and
- prices the idle ones from a static, region-adjustable table.

It exits **1** when any idle load balancer is found, so you can drop it into CI
as a cost gate.

## How it works

Classification is deliberately simple and evidence-based:

| Condition | Verdict |
|---|---|
| No healthy targets | `IDLE` (nothing can serve traffic) |
| Healthy targets, but CloudWatch returned no datapoints | `UNKNOWN` (never flag blindly) |
| Healthy targets, traffic below the threshold (default 100 over the window) | `IDLE` |
| Healthy targets and real traffic | `ACTIVE` |

The waste estimate uses the fixed hourly ELB charge × a coarse region
multiplier × 730h. Usage-based LCU costs are excluded on purpose — the point is
the money burned by a balancer doing nothing.

## Install

```sh
go install github.com/moveeeax/aws-elb-idle-finder/cmd/aws-elb-idle-finder@latest
```

Or build from source: `go build ./cmd/aws-elb-idle-finder`.

## Usage

Credentials come from the standard AWS chain (env, shared config, SSO, IAM
role). Needs read-only `elasticloadbalancing:Describe*` and
`cloudwatch:GetMetricStatistics`.

```
aws-elb-idle-finder [flags]

  --region        AWS region to scan (default: from AWS config/env)
  --lookback      traffic lookback window, e.g. 14d or 168h (default 14d)
  --json          emit JSON instead of a table
  --no-exit-code  always exit 0 even when idle LBs are found
```

Human report:

```console
$ aws-elb-idle-finder --region eu-central-1 --lookback 14d
STATUS  KIND         NAME         HEALTHY  TRAFFIC  WASTE/MO  REASON
IDLE    classic      legacy-clb   0        0        $19.89    no registered targets
IDLE    network      batch-nlb    2        3        $17.90    near-zero traffic (3 over lookback)
IDLE    application  old-staging  0        0        $17.90    no registered targets
ACTIVE  application  prod-api     6        1200000  -         6 healthy targets, 1200000 traffic

eu-central-1 (336h0m0s lookback): 3 idle load balancer(s), ~$55.70/mo wasted.
```

JSON for pipelines:

```sh
aws-elb-idle-finder --region eu-central-1 --json | jq '.findings[] | select(.status=="IDLE")'
```

See the report format without any AWS account:

```sh
go run ./examples/demo
```

## In CI

```yaml
- run: aws-elb-idle-finder --region eu-central-1   # exits 1 if idle LBs exist
```

## Development

```sh
go test ./...        # unit tests, no live AWS
go test -race ./...
```

The scanner talks to AWS only through a small `Client` interface, so the
classifier and report logic are tested against an in-memory mock.

## License

MIT — see [LICENSE](LICENSE).
