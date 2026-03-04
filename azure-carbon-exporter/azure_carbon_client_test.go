package main

import (
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/carbonoptimization/armcarbonoptimization"
)

func TestValidateSubscriptions(t *testing.T) {
	tests := []struct {
		name          string
		subscriptions []string
		wantErr       bool
	}{
		{
			name:          "valid subscriptions",
			subscriptions: []string{"sub1", "sub2"},
			wantErr:       false,
		},
		{
			name:          "empty subscriptions",
			subscriptions: []string{},
			wantErr:       true,
		},
		{
			name:          "nil subscriptions",
			subscriptions: nil,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSubscriptions(tt.subscriptions)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSubscriptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConvertSubscriptionsToPtrs(t *testing.T) {
	tests := []struct {
		name          string
		subscriptions []string
		want          int // length of result
	}{
		{
			name:          "convert multiple subscriptions",
			subscriptions: []string{"sub1", "sub2", "sub3"},
			want:          3,
		},
		{
			name:          "convert single subscription",
			subscriptions: []string{"sub1"},
			want:          1,
		},
		{
			name:          "empty slice",
			subscriptions: []string{},
			want:          0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertSubscriptionsToPtrs(tt.subscriptions)
			if len(result) != tt.want {
				t.Errorf("convertSubscriptionsToPtrs() length = %v, want %v", len(result), tt.want)
			}

			// Verify pointers point to correct values
			for i, sub := range tt.subscriptions {
				if result[i] == nil || *result[i] != sub {
					t.Errorf("convertSubscriptionsToPtrs()[%d] = %v, want %v", i, result[i], sub)
				}
			}
		})
	}
}

func TestAllEmissionScopes(t *testing.T) {
	expectedScopes := []armcarbonoptimization.EmissionScopeEnum{
		armcarbonoptimization.EmissionScopeEnumScope1,
		armcarbonoptimization.EmissionScopeEnumScope2,
		armcarbonoptimization.EmissionScopeEnumScope3,
	}

	if len(allEmissionScopes) != len(expectedScopes) {
		t.Errorf("allEmissionScopes length = %v, want %v", len(allEmissionScopes), len(expectedScopes))
	}

	for i, scope := range allEmissionScopes {
		if scope != expectedScopes[i] {
			t.Errorf("allEmissionScopes[%d] = %v, want %v", i, scope, expectedScopes[i])
		}
	}
}

func TestAssignScope(t *testing.T) {
	tests := []struct {
		name      string
		scope     armcarbonoptimization.EmissionScopeEnum
		emissions float64
		checkFunc func(*CarbonRecord) bool
	}{
		{
			name:      "assign scope 1",
			scope:     armcarbonoptimization.EmissionScopeEnumScope1,
			emissions: 123.45,
			checkFunc: func(r *CarbonRecord) bool { return r.CarbonFootprint.Scope1 == 123.45 },
		},
		{
			name:      "assign scope 2",
			scope:     armcarbonoptimization.EmissionScopeEnumScope2,
			emissions: 678.90,
			checkFunc: func(r *CarbonRecord) bool { return r.CarbonFootprint.Scope2 == 678.90 },
		},
		{
			name:      "assign scope 3",
			scope:     armcarbonoptimization.EmissionScopeEnumScope3,
			emissions: 234.56,
			checkFunc: func(r *CarbonRecord) bool { return r.CarbonFootprint.Scope3 == 234.56 },
		},
		{
			name:      "aggregate scope 1",
			scope:     armcarbonoptimization.EmissionScopeEnumScope1,
			emissions: 50.0,
			checkFunc: func(r *CarbonRecord) bool {
				// First assignment sets to 50.0
				// Second assignment should add (tested separately)
				return r.CarbonFootprint.Scope1 == 50.0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &CarbonRecord{}
			assignScope(tt.scope, record, tt.emissions)

			if !tt.checkFunc(record) {
				t.Errorf("assignScope() did not set correct value for %v", tt.scope)
			}
		})
	}

	// Test aggregation separately
	t.Run("aggregate multiple assignments", func(t *testing.T) {
		record := &CarbonRecord{}
		assignScope(armcarbonoptimization.EmissionScopeEnumScope1, record, 100.0)
		assignScope(armcarbonoptimization.EmissionScopeEnumScope1, record, 50.0)
		if record.CarbonFootprint.Scope1 != 150.0 {
			t.Errorf("assignScope() aggregation failed: got %v, want 150.0", record.CarbonFootprint.Scope1)
		}
	})
}

func TestGetOrCreateResourceRecord(t *testing.T) {
	tests := []struct {
		name         string
		usageMonth   time.Time
		location     string
		resourceType string
		recordKey    string
		wantNew      bool
	}{
		{
			name:         "create new record",
			usageMonth:   time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC),
			location:     "west europe",
			resourceType: "virtualmachines",
			recordKey:    "2024-08-01_west europe_virtualmachines",
			wantNew:      true,
		},
		{
			name:         "return existing record",
			usageMonth:   time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC),
			location:     "west europe",
			resourceType: "virtualmachines",
			recordKey:    "2024-08-01_west europe_virtualmachines",
			wantNew:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recordMap := make(map[string]*CarbonRecord)

			// Add existing record if specified
			if !tt.wantNew {
				recordMap[tt.recordKey] = &CarbonRecord{
					UsageMonth:   tt.usageMonth,
					Location:     tt.location,
					ResourceType: tt.resourceType,
				}
			}

			record := getOrCreateResourceRecord(tt.usageMonth, tt.location, tt.resourceType, recordMap, tt.recordKey)

			if record == nil {
				t.Error("getOrCreateResourceRecord() returned nil record")
				return
			}

			if tt.wantNew && len(recordMap) != 1 {
				t.Errorf("getOrCreateResourceRecord() should create new record, map length = %v", len(recordMap))
			}

			if !tt.wantNew && len(recordMap) != 1 {
				t.Errorf("getOrCreateResourceRecord() should reuse existing record, map length = %v", len(recordMap))
			}

			if record.Location != tt.location {
				t.Errorf("getOrCreateResourceRecord() location = %v, want %v", record.Location, tt.location)
			}

			if record.ResourceType != tt.resourceType {
				t.Errorf("getOrCreateResourceRecord() resourceType = %v, want %v", record.ResourceType, tt.resourceType)
			}
		})
	}
}

func TestConvertMapToCarbonRecords(t *testing.T) {
	// Create test data
	recordMap := make(map[string]*CarbonRecord)

	dates := []string{"2024-08-03", "2024-08-01", "2024-08-02"}

	for _, date := range dates {
		parsedTime, _ := time.Parse(dateLayout, date)
		recordMap[date] = &CarbonRecord{
			UsageMonth: parsedTime,
			CarbonFootprint: struct {
				Scope1 float64 `json:"scope1"`
				Scope2 float64 `json:"scope2"`
				Scope3 float64 `json:"scope3"`
			}{
				Scope1: 100.0,
				Scope2: 200.0,
				Scope3: 300.0,
			},
		}
	}

	result := convertMapToCarbonRecords(recordMap)

	// Check length
	if len(result) != len(recordMap) {
		t.Errorf("convertMapToCarbonRecords() length = %v, want %v", len(result), len(recordMap))
	}

	// Verify all records are present (order doesn't matter)
	dateSet := make(map[string]bool)
	for _, record := range result {
		dateSet[record.UsageMonth.Format(dateLayout)] = true
	}

	for _, date := range dates {
		if !dateSet[date] {
			t.Errorf("convertMapToCarbonRecords() missing date %v", date)
		}
	}
}

func TestProcessResourceEmissionRecords(t *testing.T) {
	// Create mock resource emission records
	queryDate := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	records := []armcarbonoptimization.CarbonEmissionDataClassification{
		&armcarbonoptimization.ResourceCarbonEmissionItemDetailData{
			Location:             to.Ptr("west europe"),
			ResourceType:         to.Ptr("microsoft.compute/virtualmachines"),
			LatestMonthEmissions: to.Ptr(123.45),
		},
		&armcarbonoptimization.ResourceCarbonEmissionItemDetailData{
			Location:             to.Ptr("east us"),
			ResourceType:         to.Ptr("microsoft.storage/storageaccounts"),
			LatestMonthEmissions: to.Ptr(678.90),
		},
	}

	recordMap := make(map[string]*CarbonRecord)
	scope := armcarbonoptimization.EmissionScopeEnumScope1

	err := processResourceEmissionRecords(records, scope, queryDate, recordMap)
	if err != nil {
		t.Errorf("processResourceEmissionRecords() error = %v", err)
	}

	// Check that records were created (normalized keys)
	if len(recordMap) != 2 {
		t.Errorf("processResourceEmissionRecords() created %v records, want 2", len(recordMap))
	}

	// Check specific record values
	expectedKey1 := "2024-08-01_west europe_virtualmachines"
	if record, exists := recordMap[expectedKey1]; exists {
		if record.CarbonFootprint.Scope1 != 123.45 {
			t.Errorf("processResourceEmissionRecords() Scope1 = %v, want 123.45", record.CarbonFootprint.Scope1)
		}
		if record.Location != "west europe" {
			t.Errorf("processResourceEmissionRecords() Location = %v, want west europe", record.Location)
		}
		if record.ResourceType != "virtualmachines" {
			t.Errorf("processResourceEmissionRecords() ResourceType = %v, want virtualmachines", record.ResourceType)
		}
	} else {
		t.Errorf("processResourceEmissionRecords() did not create record for key %v", expectedKey1)
	}
}

func TestProcessResourceEmissionRecords_InvalidRecord(t *testing.T) {
	// Test with invalid record type (not ResourceCarbonEmissionItemDetailData)
	queryDate := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	records := []armcarbonoptimization.CarbonEmissionDataClassification{
		&armcarbonoptimization.CarbonEmissionOverallSummaryData{
			// Just create an empty overall summary data - it should be skipped
		},
	}

	recordMap := make(map[string]*CarbonRecord)
	scope := armcarbonoptimization.EmissionScopeEnumScope1

	err := processResourceEmissionRecords(records, scope, queryDate, recordMap)
	if err != nil {
		t.Errorf("processResourceEmissionRecords() should not error on invalid record type, got: %v", err)
	}

	// Should not create any records
	if len(recordMap) != 0 {
		t.Errorf("processResourceEmissionRecords() should not create records for invalid type, got %v records", len(recordMap))
	}
}

func TestProcessResourceEmissionRecords_NilFields(t *testing.T) {
	// Test with nil required fields
	queryDate := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	records := []armcarbonoptimization.CarbonEmissionDataClassification{
		&armcarbonoptimization.ResourceCarbonEmissionItemDetailData{
			Location:             nil, // Missing location
			ResourceType:         to.Ptr("microsoft.compute/virtualmachines"),
			LatestMonthEmissions: to.Ptr(123.45),
		},
	}

	recordMap := make(map[string]*CarbonRecord)
	scope := armcarbonoptimization.EmissionScopeEnumScope1

	err := processResourceEmissionRecords(records, scope, queryDate, recordMap)
	if err != nil {
		t.Errorf("processResourceEmissionRecords() should not error on nil fields, got: %v", err)
	}

	// Should not create any records
	if len(recordMap) != 0 {
		t.Errorf("processResourceEmissionRecords() should not create records with nil fields, got %v records", len(recordMap))
	}
}

// Integration-style test (without real API calls)
func TestCarbonRecordFlow(t *testing.T) {
	// Test the complete flow from raw data to records with aggregation

	queryDate := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	recordMap := make(map[string]*CarbonRecord)

	// Mock data that simulates multiple resources with same location/resource type
	mockRecords := []armcarbonoptimization.CarbonEmissionDataClassification{
		&armcarbonoptimization.ResourceCarbonEmissionItemDetailData{
			Location:             to.Ptr("west europe"),
			ResourceType:         to.Ptr("microsoft.compute/virtualmachines"),
			LatestMonthEmissions: to.Ptr(100.0),
		},
		&armcarbonoptimization.ResourceCarbonEmissionItemDetailData{
			Location:             to.Ptr("west europe"), // Same location/resource type
			ResourceType:         to.Ptr("microsoft.compute/virtualmachines"),
			LatestMonthEmissions: to.Ptr(50.0), // Should be aggregated
		},
		&armcarbonoptimization.ResourceCarbonEmissionItemDetailData{
			Location:             to.Ptr("east us"), // Different location
			ResourceType:         to.Ptr("microsoft.compute/virtualmachines"),
			LatestMonthEmissions: to.Ptr(150.0),
		},
	}

	// Process Scope1 records
	err := processResourceEmissionRecords(mockRecords, armcarbonoptimization.EmissionScopeEnumScope1, queryDate, recordMap)
	if err != nil {
		t.Fatalf("processResourceEmissionRecords() failed: %v", err)
	}

	// Convert to slice
	result := convertMapToCarbonRecords(recordMap)

	// Verify results - should have 2 unique location/resource type combinations
	if len(result) != 2 {
		t.Errorf("Expected 2 records, got %v", len(result))
	}

	// Find the west europe record and verify aggregation
	var westEuropeRecord *CarbonRecord
	for i := range result {
		if result[i].Location == "west europe" {
			westEuropeRecord = &result[i]
			break
		}
	}

	if westEuropeRecord == nil {
		t.Fatal("Expected to find west europe record")
	}

	// Should have aggregated emissions (100.0 + 50.0 = 150.0)
	if westEuropeRecord.CarbonFootprint.Scope1 != 150.0 {
		t.Errorf("Expected aggregated Scope1 emissions 150.0, got %v", westEuropeRecord.CarbonFootprint.Scope1)
	}
}
