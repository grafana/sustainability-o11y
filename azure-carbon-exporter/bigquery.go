package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/option"
)

// AzureCarbonRecord represents a carbon emission record for BigQuery
type AzureCarbonRecord struct {
	UsageMonth      bigquery.NullDate      `bigquery:"usage_month"`
	Scope           bigquery.NullString    `bigquery:"scope"`
	Location        bigquery.NullString    `bigquery:"location"`
	AccountID       bigquery.NullString    `bigquery:"account_id"`
	ResourceType    bigquery.NullString    `bigquery:"resource_type"`
	EmissionsKgCO2e bigquery.NullFloat64   `bigquery:"emissions_kg_co2e"`
	RecordID        bigquery.NullString    `bigquery:"record_id"`
	LastUpdated     bigquery.NullTimestamp `bigquery:"last_updated"`
}

// Predefined scope names for carbon emissions
var scopes = []string{"Scope 1", "Scope 2", "Scope 3"}

// carbonEmissionsSchema defines the explicit BigQuery schema for carbon emissions
var carbonEmissionsSchema = bigquery.Schema{
	{Name: "usage_month", Type: bigquery.DateFieldType, Required: true, Description: "First day of the month for the emissions data"},
	{Name: "scope", Type: bigquery.StringFieldType, Required: true, Description: "Emission scope: Scope 1, Scope 2, or Scope 3"},
	{Name: "location", Type: bigquery.StringFieldType, Required: false, Description: "Region/Country/Cloud region"},
	{Name: "account_id", Type: bigquery.StringFieldType, Required: false, Description: "Azure subscription ID (cloud account/business unit)"},
	{Name: "resource_type", Type: bigquery.StringFieldType, Required: false, Description: "Resource type like VM, Storage, Network"},
	{Name: "emissions_kg_co2e", Type: bigquery.FloatFieldType, Required: true, Description: "Carbon emissions for that dimension in kg CO2e"},
	{Name: "record_id", Type: bigquery.StringFieldType, Required: true, Description: "Unique identifier for deduplication"},
	{Name: "last_updated", Type: bigquery.TimestampFieldType, Required: true, Description: "When the record was last updated"},
}

// BigQueryConfig holds BigQuery-specific configuration
type BigQueryConfig struct {
	ProjectID       string
	DatasetID       string
	TableID         string
	CredentialsFile string
	CreateTable     bool
	BatchSize       int
	Enabled         bool
}

// BigQueryExporter handles exporting carbon data to BigQuery
type BigQueryExporter struct {
	client *bigquery.Client
	table  *bigquery.Table
	config BigQueryConfig
}

// NewBigQueryExporter creates a new BigQuery exporter
func NewBigQueryExporter(ctx context.Context, config BigQueryConfig) (*BigQueryExporter, error) {
	if !config.Enabled {
		return nil, nil
	}

	var client *bigquery.Client
	var err error

	if config.CredentialsFile != "" {
		// Use specified credentials file
		client, err = bigquery.NewClient(ctx, config.ProjectID, option.WithCredentialsFile(config.CredentialsFile))
	} else {
		// Use Application Default Credentials
		slog.Info("No credentials file specified, using Application Default Credentials")
		client, err = bigquery.NewClient(ctx, config.ProjectID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client: %w", err)
	}

	slog.Debug("BigQuery client initialized successfully")

	dataset := client.Dataset(config.DatasetID)
	table := dataset.Table(config.TableID)

	exporter := &BigQueryExporter{
		client: client,
		table:  table,
		config: config,
	}

	// Create table if needed (dataset should already exist via Terraform)
	if config.CreateTable {
		if err := exporter.createTableIfNeeded(ctx); err != nil {
			client.Close()
			return nil, fmt.Errorf("failed to create table (ensure dataset exists via Terraform): %w", err)
		}
	}

	return exporter, nil
}

// createTableIfNeeded creates a BigQuery table if it doesn't exist
func (e *BigQueryExporter) createTableIfNeeded(ctx context.Context) error {
	slog.Info("Checking if table exists", "table", fmt.Sprintf("%s.%s.%s", e.config.ProjectID, e.config.DatasetID, e.config.TableID))

	// Create a context with timeout to prevent hanging
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := e.table.Metadata(timeoutCtx)
	if err == nil {
		slog.Debug("Table already exists")
		return nil
	}

	slog.Info("Table not found, creating new table", "error", err.Error())

	tableMetadata := &bigquery.TableMetadata{
		Schema:      carbonEmissionsSchema,
		Description: "Azure carbon emissions data",
		TimePartitioning: &bigquery.TimePartitioning{
			Field: "usage_month",
			Type:  bigquery.MonthPartitioningType,
		},
	}

	// Use timeout context for creation as well
	createCtx, createCancel := context.WithTimeout(ctx, 30*time.Second)
	defer createCancel()

	if err := e.table.Create(createCtx, tableMetadata); err != nil {
		// Check if it's a dataset not found error
		if strings.Contains(err.Error(), "Not found: Dataset") {
			return fmt.Errorf("failed to create table - dataset '%s' does not exist. Please ensure the dataset is created via Terraform first: %w", e.config.DatasetID, err)
		}
		return fmt.Errorf("failed to create table: %w", err)
	}

	slog.Info("Created table", "table", fmt.Sprintf("%s.%s.%s", e.config.ProjectID, e.config.DatasetID, e.config.TableID))
	return nil
}

// ExportRecords exports carbon records to BigQuery
func (e *BigQueryExporter) ExportRecords(ctx context.Context, records []CarbonRecord, subscriptionIDs []string) error {
	if e == nil || !e.config.Enabled {
		return nil
	}

	if len(records) == 0 {
		slog.Debug("No records to export to BigQuery")
		return nil
	}

	slog.Debug("Starting BigQuery export", "records", len(records), "subscriptions", len(subscriptionIDs))

	// Convert carbon records to BigQuery format - one row per scope per subscription per month per location per resource type
	bqRecords := make([]*AzureCarbonRecord, 0, len(records)*len(subscriptionIDs)*3) // 3 scopes per record
	now := time.Now()

	for _, record := range records {
		// Convert usage month to first day of month for usage_month field
		usageMonth := time.Date(record.UsageMonth.Year(), record.UsageMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

		for _, subscriptionID := range subscriptionIDs {
			// Create separate records for each scope
			for i, scope := range scopes {
				emissions := getScopeEmissions(record.CarbonFootprint, i)

				// Create a unique record ID based on the natural key
				recordIDParts := []string{
					subscriptionID,
					usageMonth.Format("2006-01"),
					strings.ReplaceAll(scope, " ", ""),
					"loc_" + strings.ToLower(strings.ReplaceAll(record.Location, " ", "")),
					"svc_" + strings.ToLower(strings.ReplaceAll(record.ResourceType, " ", "")),
				}
				recordID := strings.Join(recordIDParts, "_")

				bqRecord := &AzureCarbonRecord{
					UsageMonth: bigquery.NullDate{
						Date:  civil.DateOf(usageMonth),
						Valid: true,
					},
					Scope: bigquery.NullString{
						StringVal: scope,
						Valid:     true,
					},
					Location: bigquery.NullString{
						StringVal: record.Location,
						Valid:     record.Location != "",
					},
					AccountID: bigquery.NullString{
						StringVal: subscriptionID,
						Valid:     subscriptionID != "",
					},
					ResourceType: bigquery.NullString{
						StringVal: record.ResourceType,
						Valid:     record.ResourceType != "",
					},
					EmissionsKgCO2e: bigquery.NullFloat64{
						Float64: emissions,
						Valid:   true,
					},
					RecordID: bigquery.NullString{
						StringVal: recordID,
						Valid:     true,
					},
					LastUpdated: bigquery.NullTimestamp{
						Timestamp: now,
						Valid:     true,
					},
				}
				bqRecords = append(bqRecords, bqRecord)
			}
		}
	}

	// Use MERGE operation to handle duplicates
	return e.mergeRecords(ctx, bqRecords)
}

// mergeRecords uses a persistent staging table and MERGE to handle duplicates
func (e *BigQueryExporter) mergeRecords(ctx context.Context, records []*AzureCarbonRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Use persistent staging table
	stagingTableID := fmt.Sprintf("%s_staging", e.config.TableID)
	dataset := e.client.Dataset(e.config.DatasetID)
	stagingTable := dataset.Table(stagingTableID)

	// Ensure staging table exists
	if err := e.ensureStagingTableExists(ctx, stagingTable, stagingTableID); err != nil {
		return fmt.Errorf("failed to ensure staging table exists: %w", err)
	}

	// Truncate staging table before use
	if err := e.truncateStagingTable(ctx, stagingTableID); err != nil {
		return fmt.Errorf("failed to truncate staging table: %w", err)
	}

	// Insert records into staging table
	if err := e.streamRecordsToTable(ctx, stagingTable, records); err != nil {
		return fmt.Errorf("failed to insert records into staging table: %w", err)
	}

	// Build and execute MERGE query
	mergeQuery := e.buildCarbonMergeQuery(stagingTableID)
	slog.Debug("Executing MERGE query", "staging_table", stagingTableID)

	query := e.client.Query(mergeQuery)
	mergeJob, err := query.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to start MERGE job: %w", err)
	}

	mergeStatus, err := mergeJob.Wait(ctx)
	if err != nil {
		return fmt.Errorf("MERGE job failed: %w", err)
	}

	if mergeStatus.Err() != nil {
		return fmt.Errorf("MERGE job completed with errors: %w", mergeStatus.Err())
	}

	slog.Info("MERGE operation completed successfully", "staging_table", stagingTableID)
	return nil
}

// streamRecordsToTable streams records to a specific BigQuery table in batches
func (e *BigQueryExporter) streamRecordsToTable(ctx context.Context, table *bigquery.Table, records []*AzureCarbonRecord) error {
	inserter := table.Inserter()
	inserter.IgnoreUnknownValues = true
	inserter.SkipInvalidRows = true

	batchSize := e.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}

	totalRecords := len(records)
	slog.Debug("Streaming records to BigQuery table", "total_records", totalRecords, "batch_size", batchSize)

	// Use errgroup for parallel batch processing with concurrency limit
	g, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, 4) // Limit concurrency to 4 parallel inserts

	batchCount := 0
	for i := 0; i < totalRecords; i += batchSize {
		end := i + batchSize
		if end > totalRecords {
			end = totalRecords
		}

		batch := records[i:end]
		batchNum := batchCount + 1
		batchCount++

		// Acquire semaphore
		sem <- struct{}{}

		g.Go(func() error {
			defer func() { <-sem }() // Release semaphore

			// Convert to slice of interface{} as required by BigQuery inserter
			rows := make([]interface{}, len(batch))
			for j, record := range batch {
				rows[j] = record
			}

			// Insert the batch
			err := inserter.Put(ctx, rows)
			if err != nil {
				// Handle insertion errors with detailed logging
				if multiError, ok := err.(bigquery.PutMultiError); ok {
					for _, putError := range multiError {
						slog.Error("BigQuery insertion error",
							"batch", batchNum,
							"row_index", putError.RowIndex,
							"error", putError.Error())
					}
				}
				return fmt.Errorf("failed to insert batch %d: %w", batchNum, err)
			}

			slog.Debug("Inserted batch to BigQuery", "batch", batchNum, "records_in_batch", len(batch))
			return nil
		})
	}

	// Wait for all batches to complete
	if err := g.Wait(); err != nil {
		return err
	}

	slog.Info("Successfully exported records to BigQuery",
		"total_records", totalRecords,
		"total_batches", batchCount,
		"target_table", fmt.Sprintf("%s.%s.%s", table.ProjectID, table.DatasetID, table.TableID))

	return nil
}

// buildCarbonMergeQuery builds the MERGE SQL statement for carbon emissions data
func (e *BigQueryExporter) buildCarbonMergeQuery(stagingTableID string) string {
	targetTable := fmt.Sprintf("`%s.%s.%s`", e.config.ProjectID, e.config.DatasetID, e.config.TableID)
	stagingTable := fmt.Sprintf("`%s.%s.%s`", e.config.ProjectID, e.config.DatasetID, stagingTableID)

	return fmt.Sprintf(`
MERGE %s T
USING (
  SELECT * EXCEPT(row_num)
  FROM (
    SELECT *,
           ROW_NUMBER() OVER (
             PARTITION BY record_id
             ORDER BY last_updated DESC, emissions_kg_co2e DESC
           ) as row_num
    FROM %s
  ) ranked
  WHERE ranked.row_num = 1
) S
ON T.record_id = S.record_id
WHEN MATCHED THEN UPDATE SET
    usage_month = S.usage_month,
    scope = S.scope,
    location = S.location,
    account_id = S.account_id,
    resource_type = S.resource_type,
    emissions_kg_co2e = S.emissions_kg_co2e,
    last_updated = S.last_updated
WHEN NOT MATCHED THEN INSERT (
    usage_month,
    scope,
    location,
    account_id,
    resource_type,
    emissions_kg_co2e,
    record_id,
    last_updated
) VALUES (
    S.usage_month,
    S.scope,
    S.location,
    S.account_id,
    S.resource_type,
    S.emissions_kg_co2e,
    S.record_id,
    S.last_updated
)`, targetTable, stagingTable)
}

// ensureStagingTableExists creates the persistent staging table if it doesn't exist
func (e *BigQueryExporter) ensureStagingTableExists(ctx context.Context, stagingTable *bigquery.Table, stagingTableID string) error {
	// Create a context with timeout to prevent hanging
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := stagingTable.Metadata(timeoutCtx)
	if err == nil {
		slog.Debug("Staging table already exists", "staging_table", stagingTableID)
		return nil
	}

	slog.Info("Creating persistent staging table", "staging_table", stagingTableID)

	stagingMetadata := &bigquery.TableMetadata{
		Schema:      carbonEmissionsSchema,
		Description: "Persistent staging table for carbon emissions MERGE operations",
	}

	// Use timeout context for creation as well
	createCtx, createCancel := context.WithTimeout(ctx, 30*time.Second)
	defer createCancel()

	if err := stagingTable.Create(createCtx, stagingMetadata); err != nil {
		return fmt.Errorf("failed to create staging table: %w", err)
	}

	slog.Info("Created persistent staging table", "staging_table", stagingTableID)
	return nil
}

// truncateStagingTable clears all data from the staging table
func (e *BigQueryExporter) truncateStagingTable(ctx context.Context, stagingTableID string) error {
	truncateQuery := fmt.Sprintf("TRUNCATE TABLE `%s.%s.%s`",
		e.config.ProjectID, e.config.DatasetID, stagingTableID)

	slog.Debug("Truncating staging table", "staging_table", stagingTableID)

	// Use timeout context for truncation
	truncateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	query := e.client.Query(truncateQuery)
	truncateJob, err := query.Run(truncateCtx)
	if err != nil {
		return fmt.Errorf("failed to start TRUNCATE job: %w", err)
	}

	truncateStatus, err := truncateJob.Wait(truncateCtx)
	if err != nil {
		return fmt.Errorf("TRUNCATE job failed: %w", err)
	}

	if truncateStatus.Err() != nil {
		return fmt.Errorf("TRUNCATE job completed with errors: %w", truncateStatus.Err())
	}

	slog.Debug("Staging table truncated successfully", "staging_table", stagingTableID)
	return nil
}

// getScopeEmissions returns the emissions value for a specific scope index
func getScopeEmissions(carbonFootprint struct {
	Scope1 float64 `json:"scope1"`
	Scope2 float64 `json:"scope2"`
	Scope3 float64 `json:"scope3"`
}, scopeIndex int) float64 {
	switch scopeIndex {
	case 0:
		return carbonFootprint.Scope1
	case 1:
		return carbonFootprint.Scope2
	case 2:
		return carbonFootprint.Scope3
	default:
		return 0
	}
}

// Close closes the BigQuery client
func (e *BigQueryExporter) Close() error {
	if e != nil && e.client != nil {
		return e.client.Close()
	}
	return nil
}
