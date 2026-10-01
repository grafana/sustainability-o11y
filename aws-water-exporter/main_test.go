package main

import (
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sustainability/types"
)

func TestConfigDefaults(t *testing.T) {
	config := Config{
		AWSRegion:      "us-east-1",
		StartYear:      awsWaterHistoryStartYear,
		S3Bucket:       "grafanalabs-billing-carbon",
		S3Prefix:       "water",
		PushGatewayJob: "aws-water-exporter",
	}

	if config.S3Bucket != "grafanalabs-billing-carbon" {
		t.Errorf("Expected S3Bucket to be grafanalabs-billing-carbon, got %s", config.S3Bucket)
	}
	if config.PushGatewayJob != "aws-water-exporter" {
		t.Errorf("Expected PushGatewayJob to be aws-water-exporter, got %s", config.PushGatewayJob)
	}
}

func TestToWaterRecord(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	result := types.EstimatedWaterAllocation{
		ModelVersion: aws.String("v1.0.0"),
		TimePeriod: &types.TimePeriod{
			Start: &start,
		},
		DimensionsValues: map[string]string{
			"REGION":  "eu-south-2",
			"SERVICE": "AmazonEC2",
		},
		AllocationValues: map[string]types.WaterAllocation{
			"TOTAL_WATER_WITHDRAWALS": {
				Unit:  types.WaterAllocationUnitCubicMeters,
				Value: aws.Float64(1757.8),
			},
		},
	}

	record, ok := toWaterRecord(result)
	if !ok {
		t.Fatal("Expected toWaterRecord to succeed")
	}
	if record.Year != 2025 {
		t.Errorf("Expected Year 2025, got %d", record.Year)
	}
	if record.Region != "eu-south-2" {
		t.Errorf("Expected Region eu-south-2, got %s", record.Region)
	}
	if record.Service != "AmazonEC2" {
		t.Errorf("Expected Service AmazonEC2, got %s", record.Service)
	}
	if record.ModelVersion != "v1.0.0" {
		t.Errorf("Expected ModelVersion v1.0.0, got %s", record.ModelVersion)
	}
	if record.WithdrawalsM3 != 1757.8 {
		t.Errorf("Expected WithdrawalsM3 1757.8, got %f", record.WithdrawalsM3)
	}
}

func TestToWaterRecordMissingAllocation(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	result := types.EstimatedWaterAllocation{
		ModelVersion:     aws.String("v1.0.0"),
		TimePeriod:       &types.TimePeriod{Start: &start},
		DimensionsValues: map[string]string{},
		AllocationValues: map[string]types.WaterAllocation{},
	}

	if _, ok := toWaterRecord(result); ok {
		t.Fatal("Expected toWaterRecord to report ok=false when TOTAL_WATER_WITHDRAWALS is absent")
	}
}

func TestEncodeCSV(t *testing.T) {
	records := []WaterRecord{
		{Year: 2025, Region: "eu-south-2", Service: "AmazonEC2", ModelVersion: "v1.0.0", WithdrawalsM3: 1757.8},
		{Year: 2025, Region: "eu-central-1", Service: "AmazonS3", ModelVersion: "v1.0.0", WithdrawalsM3: 523.3},
	}

	body, err := encodeCSV(records)
	if err != nil {
		t.Fatalf("encodeCSV failed: %v", err)
	}

	out := string(body)
	if !strings.HasPrefix(out, "region,service,model_version,total_water_withdrawals_m3\n") {
		t.Errorf("Unexpected CSV header: %q", out)
	}
	// year must not appear in the body: it's a Hive partition column,
	// encoded only in the S3 key (year=YYYY/), never in the file itself.
	if strings.Contains(out, "2025,") {
		t.Errorf("Expected no year column in CSV body, got: %q", out)
	}
	if !strings.Contains(out, "eu-south-2,AmazonEC2,v1.0.0,1757.8") {
		t.Errorf("Expected row for eu-south-2 not found in: %q", out)
	}
}

func TestWaterTimePeriod(t *testing.T) {
	// Regression test for the off-by-one from the review: with the old
	// "now minus N years" logic, querying from 2026 with queryYears=2
	// produced a start of 2025 and silently dropped 2023/2024. The fixed
	// version takes startYear as an absolute year, so the full documented
	// history (2023 onward) is always requested regardless of the
	// current year.
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	start, end := waterTimePeriod(awsWaterHistoryStartYear, now)

	wantStart := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) {
		t.Errorf("expected start %v, got %v", wantStart, start)
	}
	if !end.Equal(wantEnd) {
		t.Errorf("expected end %v, got %v", wantEnd, end)
	}
}

func TestWaterTimePeriodClampsFutureStartYear(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	start, _ := waterTimePeriod(2030, now)

	wantStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) {
		t.Errorf("expected a startYear after now to clamp to now's year (%v), got %v", wantStart, start)
	}
}
