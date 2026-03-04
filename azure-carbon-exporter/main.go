package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/carbonoptimization/armcarbonoptimization"
	"github.com/prometheus/client_golang/prometheus"
)

// Config holds the configuration for the exporter
type Config struct {
	// Azure credentials
	TenantID     string
	ClientID     string
	ClientSecret string

	// Query parameters
	Subscriptions []string
	QueryMonths   int

	// Prometheus Push Gateway
	PushGatewayURL string
	PushGatewayJob string

	// BigQuery configuration
	BigQuery BigQueryConfig

	// Logging
	LogLevel string
}

// getCredential resolves credentials from command-line flags or environment variables
func getCredential(flagValue, envVarName string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(envVarName)
}

func main() {
	ctx := context.Background()

	// Initialize structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Parse command-line flags
	var config Config

	// Azure credentials
	flag.StringVar(&config.TenantID, "azure.tenant-id", "", "Azure Tenant ID")
	flag.StringVar(&config.ClientID, "azure.client-id", "", "Azure Client ID")
	flag.StringVar(&config.ClientSecret, "azure.client-secret", "", "Azure Client Secret")

	// Query parameters
	subscriptionsFlag := flag.String("subscriptions", "", "Comma-separated list of subscription IDs to query (required)")
	flag.IntVar(&config.QueryMonths, "query-months", 2, "Number of months to query from the end of available date range (default: 2, set to 0 to query full available range)")

	// Prometheus Push Gateway
	flag.StringVar(&config.PushGatewayURL, "prom.pushgateway.url", "", "Prometheus Push Gateway URL (optional)")
	flag.StringVar(&config.PushGatewayJob, "prom.pushgateway.job", "azure-carbon-exporter", "Prometheus Push Gateway job name")

	// BigQuery configuration
	flag.StringVar(&config.BigQuery.ProjectID, "bigquery.project-id", "", "BigQuery project ID (optional)")
	flag.StringVar(&config.BigQuery.DatasetID, "bigquery.dataset", "", "BigQuery dataset ID (optional)")
	flag.StringVar(&config.BigQuery.TableID, "bigquery.table", "", "BigQuery table ID (optional)")
	flag.StringVar(&config.BigQuery.CredentialsFile, "bigquery.credentials", "", "Path to BigQuery credentials JSON file (optional)")
	flag.BoolVar(&config.BigQuery.CreateTable, "bigquery.create-table", true, "Create BigQuery table if it doesn't exist")
	flag.IntVar(&config.BigQuery.BatchSize, "bigquery.batch-size", 100, "BigQuery batch size for streaming inserts")

	// Logging
	flag.StringVar(&config.LogLevel, "log-level", "info", "Log level: debug, info, warn, error")

	flag.Parse()

	// Parse subscriptions
	if *subscriptionsFlag != "" {
		config.Subscriptions = strings.Split(*subscriptionsFlag, ",")
		for i, subscription := range config.Subscriptions {
			config.Subscriptions[i] = strings.TrimSpace(subscription)
		}
	}

	// Set log level
	var logLevel slog.Level
	switch strings.ToLower(config.LogLevel) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
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

	// Show help if no arguments provided
	if len(os.Args) == 1 {
		fmt.Println(`Azure Carbon Emissions Exporter
===============================
This tool exports Azure carbon emissions data as Prometheus metrics. Usage:`)
		flag.PrintDefaults()
		fmt.Println(`Examples:
# Basic usage with service principal
./azure-carbon-exporter
--azure.tenant-id=xxx \\
--azure.client-id=xxx \\
--azure.client-secret=xxx

# Query specific subscriptions
./azure-carbon-exporter \\
--subscriptions=sub1,sub2,sub3

# With Push Gateway
./azure-carbon-exporter \\
--prom.pushgateway.url=http://pushgateway:9091 \\
--prom.pushgateway.job=carbon-emissions

# With BigQuery export
./azure-carbon-exporter \\
--bigquery.project-id=my-project \\
--bigquery.dataset=carbon_emissions \\
--bigquery.table=azure_carbon_data \\
--bigquery.credentials=/path/to/credentials.json`)
		return
	}

	// Resolve Azure credentials
	resolvedTenantID := getCredential(config.TenantID, "AZURE_TENANT_ID")
	resolvedClientID := getCredential(config.ClientID, "AZURE_CLIENT_ID")
	resolvedClientSecret := getCredential(config.ClientSecret, "AZURE_CLIENT_SECRET")

	// Resolve BigQuery credentials and enable BigQuery if configured
	config.BigQuery.ProjectID = getCredential(config.BigQuery.ProjectID, "BIGQUERY_PROJECT_ID")
	config.BigQuery.DatasetID = getCredential(config.BigQuery.DatasetID, "BIGQUERY_DATASET")
	config.BigQuery.TableID = getCredential(config.BigQuery.TableID, "BIGQUERY_TABLE")
	config.BigQuery.CredentialsFile = getCredential(config.BigQuery.CredentialsFile, "GOOGLE_APPLICATION_CREDENTIALS")

	// Enable BigQuery if all required parameters are provided
	config.BigQuery.Enabled = config.BigQuery.ProjectID != "" && config.BigQuery.DatasetID != "" && config.BigQuery.TableID != ""

	if config.BigQuery.Enabled {
		slog.Info("BigQuery export enabled",
			"project", config.BigQuery.ProjectID,
			"dataset", config.BigQuery.DatasetID,
			"table", config.BigQuery.TableID)
	} else {
		slog.Info("BigQuery export disabled - missing required configuration")
	}

	if err := run(ctx, config, resolvedTenantID, resolvedClientID, resolvedClientSecret); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, config Config, tenantID, clientID, clientSecret string) error {
	// Initialize metrics collector
	metrics := NewMetricsCollector()
	metrics.Register(prometheus.DefaultRegisterer)

	// Track desired process exit code; default is success (0)
	exitCode := 0
	defer func() {
		// Push metrics to Push Gateway if configured
		pushConfig := PushGatewayConfig{
			URL:     config.PushGatewayURL,
			Job:     config.PushGatewayJob,
			Enabled: config.PushGatewayURL != "" && config.PushGatewayJob != "",
		}
		if err := metrics.PushMetrics(pushConfig); err != nil {
			slog.Error("Failed to push metrics", "error", err)
		}
		// Ensure non-zero exit when requested
		os.Exit(exitCode)
	}()

	// Create Azure Carbon API client
	client, err := NewAzureCarbonClient(tenantID, clientID, clientSecret)
	if err != nil {
		return fmt.Errorf("failed to create Azure Carbon client: %w", err)
	}

	// Create BigQuery exporter if enabled
	var bqExporter *BigQueryExporter
	if config.BigQuery.Enabled {
		bqExporter, err = NewBigQueryExporter(ctx, config.BigQuery)
		if err != nil {
			return fmt.Errorf("failed to create BigQuery exporter: %w", err)
		}
		defer func() {
			if err := bqExporter.Close(); err != nil {
				slog.Error("Failed to close BigQuery client", "error", err)
			}
		}()
	}

	// Create exporter
	exporter := &CarbonExporter{
		client:     client,
		metrics:    metrics,
		config:     config,
		bqExporter: bqExporter,
	}

	// Record the start of a new run
	metrics.RecordRun()
	runStart := time.Now()
	defer func() {
		metrics.RecordProcessingDuration(time.Since(runStart))
	}()

	// Perform single scrape and exit
	slog.Info("Starting carbon emissions export")
	exporter.processCarbonEmissions(ctx)
	slog.Info("Carbon emissions export completed")

	return nil
}

// CarbonExporter handles the carbon emissions data processing
type CarbonExporter struct {
	client     *AzureCarbonClient
	metrics    *MetricsCollector
	config     Config
	bqExporter *BigQueryExporter
}

// scrape performs a single scrape of carbon emissions data
func (e *CarbonExporter) processCarbonEmissions(ctx context.Context) {
	scrapeStart := time.Now()
	defer func() {
		e.metrics.RecordProcessingDuration(time.Since(scrapeStart))
	}()

	slog.Info("Starting carbon emissions scrape")
	e.metrics.RecordRun()

	// Get available date range from Azure
	dateRangeTimer := e.metrics.TimedOperation("get_date_range")
	availableDateRange, err := e.client.GetAvailableDateRange(ctx)
	dateRangeTimer()

	if err != nil {
		slog.Error("Failed to get available date range", "error", err)
		e.metrics.RecordError()
		return
	}

	// Calculate date range: query the last N months from the end of available range
	var dateRange *armcarbonoptimization.DateRange
	queryMonths := e.config.QueryMonths

	if queryMonths <= 0 {
		// queryMonths=0 means query full available range
		dateRange = availableDateRange
		slog.Info("Using full available date range",
			"start_date", dateRange.Start.Format("2006-01-02"),
			"end_date", dateRange.End.Format("2006-01-02"))
	} else {
		// Start from the end month and go back N months (e.g., if end is March and N=2, start is February)
		endMonth := *availableDateRange.End
		startMonth := endMonth.AddDate(0, -queryMonths+1, 0) // Go back N-1 months to get N months total

		// Ensure we don't go before the available start date
		if startMonth.Before(*availableDateRange.Start) {
			startMonth = *availableDateRange.Start
		}

		dateRange = &armcarbonoptimization.DateRange{
			Start: &startMonth,
			End:   &endMonth,
		}

		slog.Info("Using incremental date range",
			"start_date", dateRange.Start.Format("2006-01-02"),
			"end_date", dateRange.End.Format("2006-01-02"),
			"query_months", queryMonths,
			"available_start", availableDateRange.Start.Format("2006-01-02"),
			"available_end", availableDateRange.End.Format("2006-01-02"))
	}

	// Query carbon emissions data
	queryTimer := e.metrics.TimedOperation("query_emissions")
	var response *CarbonQueryResponse

	response, err = e.client.QueryResourceItemDetails(ctx, dateRange, e.config.Subscriptions)
	queryTimer()

	// Record API call
	e.metrics.RecordCarbonAPICall()

	if err != nil {
		slog.Error("Failed to query carbon emissions", "error", err)
		e.metrics.RecordError()
		return
	}

	slog.Info("Successfully queried carbon emissions",
		"records", len(response.Records),
		"start_date", dateRange.Start,
		"end_date", dateRange.End)

	// Process carbon emissions data (for operational metrics only)
	processTimer := e.metrics.TimedOperation("process_records")
	e.metrics.ProcessCarbonEmissions(response.Records)
	processTimer()

	// Export to BigQuery if enabled
	if e.bqExporter != nil {
		bqTimer := e.metrics.TimedOperation("bigquery_export")
		err = e.bqExporter.ExportRecords(ctx, response.Records, e.config.Subscriptions)
		bqTimer()

		if err != nil {
			slog.Error("Failed to export records to BigQuery", "error", err)
			e.metrics.RecordError()
		} else {
			slog.Info("Successfully exported records to BigQuery", "records", len(response.Records))
			e.metrics.RecordBigQueryUpload()
		}
	}

	pushConfig := PushGatewayConfig{
		URL:     e.config.PushGatewayURL,
		Job:     e.config.PushGatewayJob,
		Enabled: e.config.PushGatewayURL != "",
	}

	if err := e.metrics.PushMetrics(pushConfig); err != nil {
		slog.Error("Failed to push metrics", "error", err)
		e.metrics.RecordError()
	}

	slog.Info("Carbon emissions processing completed", "duration", time.Since(scrapeStart))
}
