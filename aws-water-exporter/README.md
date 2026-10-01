# AWS Water Withdrawal Exporter

An exporter that fetches AWS Sustainability water withdrawal estimates and writes them to S3 as CSV, for querying via Athena.

## Features

- **AWS Sustainability API Integration**: Calls `GetEstimatedWaterAllocation`, grouped by region and service
- **S3/Athena Export**: Writes one CSV object per calendar year, partitioned by year in the S3 key — not BigQuery, since this data is a permanent multi-year record rather than a rolling metric subject to retention
- **Operational Metrics**: Exposes operational metrics for monitoring exporter health
- **Dry Run**: Fetch and log records without writing to S3, for validating API responses before touching production data

## Why this exists

Water withdrawals aren't one of the tables AWS pushes via Data Exports (unlike carbon emissions), so there's no S3 drop to point Athena at directly. The AWS Sustainability API is the only source — this exporter calls it on a schedule (via a Kubernetes CronJob in production) and lands the result in S3 itself. See [grafana/sustainability-o11y#36](https://github.com/grafana/sustainability-o11y/issues/36).

## Quick Start

### Prerequisites

- Go 1.26+ (for building from source)
- AWS credentials with `sustainability:GetEstimatedWaterAllocation` and `sustainability:GetEstimatedWaterAllocationDimensionValues` on the management account (the API only returns org-wide data when called from the payer account)
- An S3 bucket the credentials can write to

### Installation

#### From Source
```bash
go build -o aws-water-exporter .
```

#### Using Docker
```bash
docker build -t aws-water-exporter .
```

## Usage

### Basic Usage

Credentials come entirely from the default AWS SDK credential chain — there are no `--aws.*` credential flags. In production this is cross-account IRSA into the management account; locally, whatever your AWS CLI profile or environment variables resolve to.

```bash
AWS_PROFILE=management ./aws-water-exporter \
  --s3.bucket=grafanalabs-billing-carbon \
  --s3.prefix=water
```

### Validating Against the Live API

`--dry-run` fetches and logs every record without writing to S3 — use this to confirm the API response shape before pointing at production data:

```bash
AWS_PROFILE=management ./aws-water-exporter --dry-run --log-level=debug
```

### With Push Gateway

```bash
./aws-water-exporter \
  --s3.bucket=grafanalabs-billing-carbon \
  --prom.pushgateway.url=http://pushgateway:9091
```

## Configuration Options

| Parameter | Default | Description |
|-----------|---------|-------------|
| `--aws.region` | `us-east-1` | AWS region for the Sustainability API endpoint |
| `--start-year` | `2023` | Earliest calendar year to query. An absolute year, not a rolling window — AWS's water history starts January 2023, and a relative "N years back" window silently loses history once the most recent year isn't published yet (see `client.go`'s `waterTimePeriod`) |
| `--s3.region` | `us-east-1` | AWS region of the S3 bucket. Pinned explicitly rather than inferred from `AWS_REGION` — in production, IRSA injects that as the *cluster's* region, which isn't necessarily the bucket's region and previously caused a cross-region `PermanentRedirect` on upload |
| `--s3.bucket` | `grafanalabs-billing-carbon` | S3 bucket to write water withdrawal data to |
| `--s3.prefix` | `water` | S3 key prefix for exported data |
| `--prom.pushgateway.url` | - | Prometheus Push Gateway URL (optional) |
| `--prom.pushgateway.job` | `aws-water-exporter` | Push Gateway job name |
| `--log-level` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `--dry-run` | `false` | Fetch and log records without writing to S3 |

## Metrics

- `aws_water_exporter_runs_total` - Total number of exporter runs
- `aws_water_exporter_records_processed_total` - Total number of water allocation records processed
- `aws_water_exporter_errors_total` - Total number of errors encountered
- `aws_water_exporter_api_calls_total` - Total number of AWS Sustainability API calls
- `aws_water_exporter_s3_uploads_total` - Total number of S3 uploads
- `aws_water_exporter_last_run_timestamp` - Timestamp of the last run
- `aws_water_exporter_processing_duration_seconds` - Duration of the last processing run in seconds
- `aws_water_exporter_operation_duration_seconds` - Duration of different operations (histogram with `operation` label)

## Data Output

One CSV object per year, at `s3://<bucket>/<prefix>/year=<YYYY>/data.csv`:

```csv
region,service,model_version,total_water_withdrawals_m3
eu-south-2,AmazonEC2,v1.0.0,1757.8
eu-central-1,AmazonS3,v1.0.0,523.3
```

`year` deliberately does **not** appear as a column in the file — it's a Hive partition column, encoded only in the S3 key. A partitioned Glue table derives its value from the path; declaring it in the file as well would shift every other column over by one position when read back.

Each run overwrites the object for every year it fetches — safe to re-run, since a run before AWS finalizes a year's data should replace partial figures, not duplicate rows.

## Cadence

AWS publishes a given year's water data by June of the following year, with `YEARLY_CALENDAR` as the only supported granularity — monthly or quarterly requests return a `ValidationException`. A monthly CronJob schedule is more than sufficient; most runs are a no-op since the underlying data hasn't changed.

## API Reference

The exporter uses the AWS Sustainability API:
- **Service**: `sustainability` (API version `2018-05-10`)
- **Operation**: `GetEstimatedWaterAllocation`
- **Authentication**: AWS SigV4 via the default credential chain
- **Documentation**: [docs.aws.amazon.com/sustainability/latest/APIReference/API_GetEstimatedWaterAllocation.html](https://docs.aws.amazon.com/sustainability/latest/APIReference/API_GetEstimatedWaterAllocation.html)

## Development

### Building
```bash
go build ./...
```

### Testing
```bash
go test ./...
```

### Linting
```bash
go vet ./...
gofmt -l .
```
