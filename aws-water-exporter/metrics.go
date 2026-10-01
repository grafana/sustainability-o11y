package main

import (
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

// MetricsCollector holds all the Prometheus metrics for the water exporter.
type MetricsCollector struct {
	RunsTotal             prometheus.Counter
	RecordsProcessedTotal prometheus.Counter
	ErrorsTotal           prometheus.Counter
	APICallsTotal         prometheus.Counter
	S3UploadsTotal        prometheus.Counter

	LastRunTimestamp   prometheus.Gauge
	ProcessingDuration prometheus.Gauge

	OperationDuration prometheus.HistogramVec
}

// NewMetricsCollector creates and registers all Prometheus metrics.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		RunsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "aws_water_exporter_runs_total",
			Help: "Total number of aws-water-exporter runs",
		}),
		RecordsProcessedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "aws_water_exporter_records_processed_total",
			Help: "Total number of water allocation records processed",
		}),
		ErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "aws_water_exporter_errors_total",
			Help: "Total number of errors encountered",
		}),
		APICallsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "aws_water_exporter_api_calls_total",
			Help: "Total number of AWS Sustainability API calls",
		}),
		S3UploadsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "aws_water_exporter_s3_uploads_total",
			Help: "Total number of S3 uploads",
		}),
		LastRunTimestamp: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "aws_water_exporter_last_run_timestamp",
			Help: "Timestamp of the last run",
		}),
		ProcessingDuration: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "aws_water_exporter_processing_duration_seconds",
			Help: "Duration of the last processing run in seconds",
		}),
		OperationDuration: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "aws_water_exporter_operation_duration_seconds",
				Help:    "Duration of different operations in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation"},
		),
	}
}

// Register registers all metrics in the collector with the given registerer.
func (mc *MetricsCollector) Register(registerer prometheus.Registerer) {
	registerer.MustRegister(mc.RunsTotal)
	registerer.MustRegister(mc.RecordsProcessedTotal)
	registerer.MustRegister(mc.ErrorsTotal)
	registerer.MustRegister(mc.APICallsTotal)
	registerer.MustRegister(mc.S3UploadsTotal)
	registerer.MustRegister(mc.LastRunTimestamp)
	registerer.MustRegister(mc.ProcessingDuration)
	registerer.MustRegister(&mc.OperationDuration)
}

func (mc *MetricsCollector) RecordRun() {
	mc.RunsTotal.Inc()
	mc.LastRunTimestamp.SetToCurrentTime()
}

func (mc *MetricsCollector) RecordError() {
	mc.ErrorsTotal.Inc()
}

func (mc *MetricsCollector) RecordProcessingDuration(d time.Duration) {
	mc.ProcessingDuration.Set(d.Seconds())
}

func (mc *MetricsCollector) RecordAPICall() {
	mc.APICallsTotal.Inc()
}

func (mc *MetricsCollector) RecordS3Upload() {
	mc.S3UploadsTotal.Inc()
}

func (mc *MetricsCollector) RecordOperationDuration(operation string, d time.Duration) {
	mc.OperationDuration.WithLabelValues(operation).Observe(d.Seconds())
}

// ProcessWaterRecords updates operational metrics for a fetched batch.
func (mc *MetricsCollector) ProcessWaterRecords(records []WaterRecord) {
	mc.RecordsProcessedTotal.Add(float64(len(records)))
}

// TimedOperation helps time operations and record their duration.
func (mc *MetricsCollector) TimedOperation(operation string) func() {
	start := time.Now()
	return func() {
		mc.RecordOperationDuration(operation, time.Since(start))
	}
}

// PushGatewayConfig holds configuration for pushing metrics to a Prometheus
// Push Gateway.
type PushGatewayConfig struct {
	URL     string
	Job     string
	Enabled bool
}

// PushMetrics pushes all collected metrics to the specified Push Gateway.
func (mc *MetricsCollector) PushMetrics(config PushGatewayConfig) error {
	if !config.Enabled {
		slog.Debug("Push Gateway not configured, skipping metrics push")
		return nil
	}

	slog.Info("Pushing metrics to Push Gateway", "url", config.URL, "job", config.Job)

	if _, err := url.Parse(config.URL); err != nil {
		return fmt.Errorf("invalid Push Gateway URL: %w", err)
	}

	pusher := push.New(config.URL, config.Job).Gatherer(prometheus.DefaultGatherer)
	pusher = pusher.Grouping("instance", "aws-water-exporter")
	pusher = pusher.Grouping("app", "aws-water-exporter")

	if err := pusher.Push(); err != nil {
		return fmt.Errorf("failed to push metrics to Push Gateway: %w", err)
	}

	slog.Info("Successfully pushed metrics to Push Gateway")
	return nil
}
