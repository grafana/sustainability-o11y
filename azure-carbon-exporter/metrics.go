package main

import (
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

// MetricsCollector holds all the Prometheus metrics for the carbon exporter
type MetricsCollector struct {
	// Counters
	RunsTotal             prometheus.Counter
	RecordsProcessedTotal prometheus.Counter
	ErrorsTotal           prometheus.Counter
	CarbonAPICallsTotal   prometheus.Counter
	BigQueryUploadsTotal  prometheus.Counter

	// Gauges
	LastRunTimestamp             prometheus.Gauge
	ProcessingDuration           prometheus.Gauge
	LatestRecordTimestampSeconds prometheus.Gauge

	// Histograms
	OperationDuration prometheus.HistogramVec
}

// NewMetricsCollector creates and registers all Prometheus metrics
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		RunsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "azure_carbon_exporter_runs_total",
			Help: "Total number of azure-carbon-exporter runs",
		}),
		RecordsProcessedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "azure_carbon_exporter_records_processed_total",
			Help: "Total number of carbon emission records processed",
		}),
		ErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "azure_carbon_exporter_errors_total",
			Help: "Total number of errors encountered",
		}),
		CarbonAPICallsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "azure_carbon_exporter_api_calls_total",
			Help: "Total number of Azure Carbon API calls",
		}),
		BigQueryUploadsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "azure_carbon_exporter_bigquery_uploads_total",
			Help: "Total number of BigQuery uploads",
		}),
		LastRunTimestamp: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "azure_carbon_exporter_last_run_timestamp",
			Help: "Timestamp of the last run",
		}),
		ProcessingDuration: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "azure_carbon_exporter_processing_duration_seconds",
			Help: "Duration of the last processing run in seconds",
		}),
		LatestRecordTimestampSeconds: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "azure_carbon_exporter_latest_record_timestamp_seconds",
			Help: "Latest record date timestamp (seconds since epoch) observed while processing carbon emission records",
		}),
		OperationDuration: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "azure_carbon_exporter_operation_duration_seconds",
				Help:    "Duration of different operations in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation"},
		),
	}

	return mc
}

// Register registers all metrics in the collector with the given registerer
func (mc *MetricsCollector) Register(registerer prometheus.Registerer) {
	registerer.MustRegister(mc.RunsTotal)
	registerer.MustRegister(mc.RecordsProcessedTotal)
	registerer.MustRegister(mc.ErrorsTotal)
	registerer.MustRegister(mc.CarbonAPICallsTotal)
	registerer.MustRegister(mc.BigQueryUploadsTotal)
	registerer.MustRegister(mc.LastRunTimestamp)
	registerer.MustRegister(mc.ProcessingDuration)
	registerer.MustRegister(mc.LatestRecordTimestampSeconds)
	registerer.MustRegister(&mc.OperationDuration)
}

// RecordRun increments the runs counter and sets the timestamp
func (mc *MetricsCollector) RecordRun() {
	mc.RunsTotal.Inc()
	mc.LastRunTimestamp.SetToCurrentTime()
}

// RecordError increments the error counter
func (mc *MetricsCollector) RecordError() {
	mc.ErrorsTotal.Inc()
}

// RecordProcessingDuration sets the processing duration gauge
func (mc *MetricsCollector) RecordProcessingDuration(duration time.Duration) {
	mc.ProcessingDuration.Set(duration.Seconds())
}

// RecordCarbonAPICall increments the Carbon API calls counter
func (mc *MetricsCollector) RecordCarbonAPICall() {
	mc.CarbonAPICallsTotal.Inc()
}

// RecordBigQueryUpload increments the BigQuery uploads counter
func (mc *MetricsCollector) RecordBigQueryUpload() {
	mc.BigQueryUploadsTotal.Inc()
}

// RecordOperationDuration records the duration of a specific operation
func (mc *MetricsCollector) RecordOperationDuration(operation string, duration time.Duration) {
	mc.OperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// SetLatestRecordTimestamp sets the latest-record gauge from a time.Time
func (mc *MetricsCollector) SetLatestRecordTimestamp(t time.Time) {
	if t.IsZero() {
		return
	}
	mc.LatestRecordTimestampSeconds.Set(float64(t.Unix()))
}

// TimedOperation helps time operations and record their duration
func (mc *MetricsCollector) TimedOperation(operation string) func() {
	start := time.Now()
	return func() {
		mc.RecordOperationDuration(operation, time.Since(start))
	}
}

// ProcessCarbonEmissions processes carbon emissions records and updates operational metrics
func (mc *MetricsCollector) ProcessCarbonEmissions(records []CarbonRecord) {
	var latestTimestamp time.Time

	for _, record := range records {
		// Parse the date to find the latest record
		if recordTime := record.UsageMonth; !recordTime.IsZero() {
			if recordTime.After(latestTimestamp) {
				latestTimestamp = recordTime
			}
		}

		mc.RecordsProcessedTotal.Inc()
	}

	// Set the latest record timestamp
	mc.SetLatestRecordTimestamp(latestTimestamp)
}

// PushGatewayConfig holds configuration for pushing metrics to a Prometheus Push Gateway
type PushGatewayConfig struct {
	URL     string
	Job     string
	Enabled bool
}

// PushMetrics pushes all collected metrics to the specified Push Gateway
func (mc *MetricsCollector) PushMetrics(config PushGatewayConfig) error {
	if !config.Enabled {
		slog.Debug("Push Gateway not configured, skipping metrics push")
		return nil
	}

	slog.Info("Pushing metrics to Push Gateway", "url", config.URL, "job", config.Job)

	// Validate URL
	if _, err := url.Parse(config.URL); err != nil {
		return fmt.Errorf("invalid Push Gateway URL: %w", err)
	}

	// Create a Push Gateway client
	pusher := push.New(config.URL, config.Job).Gatherer(prometheus.DefaultGatherer)

	// Add some instance labels for better identification
	pusher = pusher.Grouping("instance", "azure-carbon-exporter")
	pusher = pusher.Grouping("app", "azure-carbon-exporter")

	// Push metrics
	if err := pusher.Push(); err != nil {
		return fmt.Errorf("failed to push metrics to Push Gateway: %w", err)
	}

	slog.Info("Successfully pushed metrics to Push Gateway")
	return nil
}
