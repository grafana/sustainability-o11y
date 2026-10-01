package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// awsWaterHistoryStartYear is the first year AWS publishes water allocation
// data for, per the Sustainability API docs. It's the default --start-year:
// an absolute year, not a rolling window, so it never needs bumping as
// calendar years pass (see the comment on FetchWaterWithdrawals).
const awsWaterHistoryStartYear = 2023

// Config holds the configuration for the exporter.
type Config struct {
	// AWS
	AWSRegion string
	StartYear int

	// S3 destination
	S3Bucket string
	S3Prefix string

	// Prometheus Push Gateway
	PushGatewayURL string
	PushGatewayJob string

	// Logging
	LogLevel string

	// DryRun skips the S3 export and logs the fetched records instead.
	// Use this to validate the AWS Sustainability API response shape
	// (DimensionsValues keys, units, etc.) before writing anywhere.
	DryRun bool
}

func main() {
	ctx := context.Background()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	var config Config

	flag.StringVar(&config.AWSRegion, "aws.region", "us-east-1", "AWS region for the Sustainability API endpoint")
	flag.IntVar(&config.StartYear, "start-year", awsWaterHistoryStartYear, "Earliest calendar year to query (absolute year, not a rolling window — AWS water history starts January 2023)")

	flag.StringVar(&config.S3Bucket, "s3.bucket", "grafanalabs-billing-carbon", "S3 bucket to write water withdrawal data to")
	flag.StringVar(&config.S3Prefix, "s3.prefix", "water", "S3 key prefix for exported data")

	flag.StringVar(&config.PushGatewayURL, "prom.pushgateway.url", "", "Prometheus Push Gateway URL (optional)")
	flag.StringVar(&config.PushGatewayJob, "prom.pushgateway.job", "aws-water-exporter", "Prometheus Push Gateway job name")

	flag.StringVar(&config.LogLevel, "log-level", "info", "Log level: debug, info, warn, error")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Fetch and log records without writing to S3")

	flag.Parse()

	var logLevel slog.Level
	switch strings.ToLower(config.LogLevel) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	if len(os.Args) == 1 {
		fmt.Println(`AWS Water Withdrawals Exporter
================================
Exports AWS Sustainability water withdrawal estimates to S3. Usage:`)
		flag.PrintDefaults()
		fmt.Println(`Example:
./aws-water-exporter \
  --s3.bucket=grafanalabs-billing-carbon \
  --s3.prefix=water \
  --prom.pushgateway.url=http://pushgateway:9091`)
		return
	}

	if err := run(ctx, config); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, config Config) error {
	metrics := NewMetricsCollector()
	metrics.Register(prometheus.DefaultRegisterer)

	// Push metrics on the way out regardless of outcome, then let run()
	// return normally — os.Exit from inside a defer would terminate the
	// process before this function's own return value ever reached main(),
	// silently swallowing the real error.
	defer func() {
		pushConfig := PushGatewayConfig{
			URL:     config.PushGatewayURL,
			Job:     config.PushGatewayJob,
			Enabled: config.PushGatewayURL != "",
		}
		if err := metrics.PushMetrics(pushConfig); err != nil {
			slog.Error("Failed to push metrics", "error", err)
		}
	}()

	runStart := time.Now()
	metrics.RecordRun()
	defer func() {
		metrics.RecordProcessingDuration(time.Since(runStart))
	}()

	client, err := NewWaterAllocationClient(ctx, config.AWSRegion)
	if err != nil {
		metrics.RecordError()
		return fmt.Errorf("failed to create AWS Sustainability client: %w", err)
	}

	slog.Info("Fetching water withdrawal estimates", "start_year", config.StartYear)
	fetchTimer := metrics.TimedOperation("fetch_water_withdrawals")
	metrics.RecordAPICall()
	records, err := client.FetchWaterWithdrawals(ctx, config.StartYear)
	fetchTimer()
	if err != nil {
		metrics.RecordError()
		return fmt.Errorf("failed to fetch water withdrawals: %w", err)
	}
	slog.Info("Fetched water withdrawal estimates", "records", len(records))
	metrics.ProcessWaterRecords(records)

	if config.DryRun {
		slog.Info("Dry run: skipping S3 export, logging records instead")
		for _, r := range records {
			slog.Info("record",
				"year", r.Year,
				"region", r.Region,
				"service", r.Service,
				"model_version", r.ModelVersion,
				"withdrawals_m3", r.WithdrawalsM3,
			)
		}
		slog.Info("Water withdrawal dry run completed", "duration", time.Since(runStart))
		return nil
	}

	exporter, err := NewS3Exporter(ctx, config.S3Bucket, config.S3Prefix)
	if err != nil {
		metrics.RecordError()
		return fmt.Errorf("failed to create S3 exporter: %w", err)
	}

	exportTimer := metrics.TimedOperation("export_to_s3")
	err = exporter.ExportRecords(ctx, records)
	exportTimer()
	if err != nil {
		metrics.RecordError()
		return fmt.Errorf("failed to export records to S3: %w", err)
	}
	metrics.RecordS3Upload()

	slog.Info("Water withdrawal export completed", "duration", time.Since(runStart))
	return nil
}
