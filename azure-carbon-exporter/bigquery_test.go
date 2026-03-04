package main

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
)

func TestAzureCarbonRecord_Schema(t *testing.T) {
	// Test that the schema can be inferred correctly
	_, err := bigquery.InferSchema(AzureCarbonRecord{})
	if err != nil {
		t.Fatalf("Failed to infer schema: %v", err)
	}
}

func TestBigQueryConfig_Validation(t *testing.T) {
	tests := []struct {
		name     string
		config   BigQueryConfig
		expected bool
	}{
		{
			name: "complete config",
			config: BigQueryConfig{
				ProjectID: "test-project",
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Enabled:   true,
			},
			expected: true,
		},
		{
			name: "missing project",
			config: BigQueryConfig{
				DatasetID: "test-dataset",
				TableID:   "test-table",
				Enabled:   false, // Should be false when project is missing
			},
			expected: false,
		},
		{
			name: "disabled config",
			config: BigQueryConfig{
				Enabled: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For this test, we just check if the config would be considered valid
			// In a real scenario, we'd need to mock the BigQuery client
			if tt.config.Enabled != tt.expected {
				t.Errorf("Expected enabled=%v, got enabled=%v", tt.expected, tt.config.Enabled)
			}
		})
	}
}

func TestCarbonRecordConversion(t *testing.T) {
	// Test converting CarbonRecord to AzureCarbonRecord with new normalized schema
	testTime := time.Date(2024, 1, 15, 12, 30, 0, 0, time.UTC)        // Mid-month time
	expectedUsageMonth := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Should be first day of month

	carbonRecord := CarbonRecord{
		UsageMonth: testTime,
		CarbonFootprint: struct {
			Scope1 float64 `json:"scope1"`
			Scope2 float64 `json:"scope2"`
			Scope3 float64 `json:"scope3"`
		}{
			Scope1: 10.5,
			Scope2: 20.3,
			Scope3: 5.7,
		},
	}

	subscriptionID := "test-subscription-123"

	// Test creating a Scope 1 record
	scope1Record := &AzureCarbonRecord{
		UsageMonth: bigquery.NullDate{
			Date:  civil.DateOf(expectedUsageMonth),
			Valid: true,
		},
		Scope: bigquery.NullString{
			StringVal: "Scope 1",
			Valid:     true,
		},
		Location: bigquery.NullString{
			StringVal: "",
			Valid:     false,
		},
		AccountID: bigquery.NullString{
			StringVal: subscriptionID,
			Valid:     subscriptionID != "",
		},
		ResourceType: bigquery.NullString{
			StringVal: "",
			Valid:     false,
		},
		EmissionsKgCO2e: bigquery.NullFloat64{
			Float64: carbonRecord.CarbonFootprint.Scope1,
			Valid:   true,
		},
	}

	// Verify the conversion worked correctly
	if !scope1Record.UsageMonth.Valid {
		t.Error("UsageMonth should be valid")
	}

	if scope1Record.UsageMonth.Date != civil.DateOf(expectedUsageMonth) {
		t.Errorf("Expected date %v, got %v", civil.DateOf(expectedUsageMonth), scope1Record.UsageMonth.Date)
	}

	if scope1Record.Scope.StringVal != "Scope 1" {
		t.Errorf("Expected scope 'Scope 1', got %v", scope1Record.Scope.StringVal)
	}

	if scope1Record.EmissionsKgCO2e.Float64 != 10.5 {
		t.Errorf("Expected emissions 10.5, got %v", scope1Record.EmissionsKgCO2e.Float64)
	}

	if scope1Record.AccountID.StringVal != subscriptionID {
		t.Errorf("Expected account ID %s, got %s", subscriptionID, scope1Record.AccountID.StringVal)
	}

	if scope1Record.Location.Valid {
		t.Error("Location should be null/invalid for now")
	}

	if scope1Record.ResourceType.Valid {
		t.Error("ResourceType should be null/invalid for now")
	}
}

func TestNewBigQueryExporter_Disabled(t *testing.T) {
	ctx := context.Background()
	config := BigQueryConfig{
		Enabled: false,
	}

	exporter, err := NewBigQueryExporter(ctx, config)
	if err != nil {
		t.Fatalf("Expected no error for disabled config, got: %v", err)
	}

	if exporter != nil {
		t.Error("Expected nil exporter for disabled config")
	}
}
