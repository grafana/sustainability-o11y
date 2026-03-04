package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/carbonoptimization/armcarbonoptimization"
)

const (
	dateLayout = "2006-01-02"
	apiTimeout = 30 * time.Second
)

// CarbonRecord represents a simplified carbon emission record
type CarbonRecord struct {
	UsageMonth      time.Time `json:"usageMonth"`
	Location        string    `json:"location,omitempty"`     // Azure region/location (e.g., "centralus")
	ResourceType    string    `json:"resourceType,omitempty"` // Azure resource type (e.g., "virtualmachines")
	CarbonFootprint struct {
		Scope1 float64 `json:"scope1"`
		Scope2 float64 `json:"scope2"`
		Scope3 float64 `json:"scope3"`
	} `json:"carbonFootprint"`
}

// CarbonQueryResponse represents the response from carbon emissions query
type CarbonQueryResponse struct {
	Records []CarbonRecord `json:"records"`
}

// AzureCarbonClient handles Azure Carbon Optimization API calls
type AzureCarbonClient struct {
	client *armcarbonoptimization.CarbonServiceClient
}

// NewAzureCarbonClient creates a new Azure Carbon API client
func NewAzureCarbonClient(tenantID, clientID, clientSecret string) (*AzureCarbonClient, error) {
	var cred azcore.TokenCredential
	var err error

	// If all three credential parameters are provided, use ClientSecretCredential
	if tenantID != "" && clientID != "" && clientSecret != "" {
		slog.Info("Using Azure Service Principal authentication for Carbon API")
		cred, err = azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure service principal credential: %w", err)
		}
	} else {
		// Fall back to default Azure credential chain
		slog.Info("Using Azure default credential chain for Carbon API")
		cred, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure credential: %w", err)
		}
	}

	client, err := armcarbonoptimization.NewCarbonServiceClient(cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure Carbon Service client: %w", err)
	}

	return &AzureCarbonClient{
		client: client,
	}, nil
}

// GetAvailableDateRange queries the available date range for carbon emissions data
func (c *AzureCarbonClient) GetAvailableDateRange(ctx context.Context) (*armcarbonoptimization.DateRange, error) {
	reqCtx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	resp, err := c.client.QueryCarbonEmissionDataAvailableDateRange(reqCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query available date range: %w", err)
	}

	if resp.StartDate == nil || resp.EndDate == nil {
		return nil, fmt.Errorf("API returned nil date range")
	}

	parsedStart, err := time.Parse(dateLayout, *resp.StartDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start date: %w", err)
	}
	parsedEnd, err := time.Parse(dateLayout, *resp.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse end date: %w", err)
	}

	return &armcarbonoptimization.DateRange{
		Start: &parsedStart,
		End:   &parsedEnd,
	}, nil
}

// QueryResourceItemDetails queries carbon emissions data for individual resources with both location and resource type
func (c *AzureCarbonClient) QueryResourceItemDetails(ctx context.Context, dateRange *armcarbonoptimization.DateRange, subscriptions []string) (*CarbonQueryResponse, error) {
	if err := validateSubscriptions(subscriptions); err != nil {
		return nil, err
	}

	sdkSubscriptions := convertSubscriptionsToPtrs(subscriptions)
	recordMap := make(map[string]*CarbonRecord)

	if err := c.queryAndProcessResourceDetails(ctx, dateRange, sdkSubscriptions, recordMap); err != nil {
		return nil, fmt.Errorf("failed to query resource details: %w", err)
	}

	records := convertMapToCarbonRecords(recordMap)
	slog.Debug("Total records", "total_records", len(records))

	return &CarbonQueryResponse{Records: records}, nil
}

func validateSubscriptions(subscriptions []string) error {
	if len(subscriptions) == 0 {
		return fmt.Errorf("at least one subscription ID must be provided")
	}
	return nil
}

func convertSubscriptionsToPtrs(subscriptions []string) []*string {
	sdkSubscriptions := make([]*string, len(subscriptions))
	for i, sub := range subscriptions {
		sdkSubscriptions[i] = &sub
	}
	return sdkSubscriptions
}

var allEmissionScopes = []armcarbonoptimization.EmissionScopeEnum{
	armcarbonoptimization.EmissionScopeEnumScope1,
	armcarbonoptimization.EmissionScopeEnumScope2,
	armcarbonoptimization.EmissionScopeEnumScope3,
}

// queryAndProcessResourceDetails queries individual resource carbon emissions with both location and resource type
// Note: ItemDetailsReport only supports one month at a time, so we query each month separately
func (c *AzureCarbonClient) queryAndProcessResourceDetails(ctx context.Context, dateRange *armcarbonoptimization.DateRange, sdkSubscriptions []*string, recordMap map[string]*CarbonRecord) error {
	slog.Debug("Querying resource-based carbon emissions (with location and resource type)")

	// ItemDetailsReport requires start and end dates to be equal (one month at a time)
	currentDate := *dateRange.Start
	endDate := *dateRange.End

	for currentDate.Before(endDate) || currentDate.Equal(endDate) {
		// Create a single-day date range for this month
		monthDateRange := &armcarbonoptimization.DateRange{
			Start: &currentDate,
			End:   &currentDate,
		}

		// Query each scope for this month
		for _, scope := range allEmissionScopes {
			queryParams := &armcarbonoptimization.ItemDetailsQueryFilter{
				ReportType:       to.Ptr(armcarbonoptimization.ReportTypeEnumItemDetailsReport),
				CategoryType:     to.Ptr(armcarbonoptimization.CategoryTypeEnumResource),
				CarbonScopeList:  []*armcarbonoptimization.EmissionScopeEnum{to.Ptr(scope)},
				SubscriptionList: sdkSubscriptions,
				DateRange:        monthDateRange,
				OrderBy:          to.Ptr(armcarbonoptimization.OrderByColumnEnumLatestMonthEmissions),
				SortDirection:    to.Ptr(armcarbonoptimization.SortDirectionEnumDesc),
				PageSize:         to.Ptr[int32](5000), // Max page size
			}

			reqCtx, cancel := context.WithTimeout(ctx, apiTimeout)
			resp, err := c.client.QueryCarbonEmissionReports(reqCtx, queryParams, nil)
			cancel()

			if err != nil {
				slog.Warn("Failed to query resource details for scope", "scope", scope, "date", currentDate.Format(dateLayout), "error", err)
				continue
			}

			slog.Debug("Parsing resource-based carbon data for scope", "scope", scope, "date", currentDate.Format(dateLayout), "records_count", len(resp.Value))
			if err := processResourceEmissionRecords(resp.Value, scope, currentDate, recordMap); err != nil {
				slog.Warn("Failed to process resource emission records", "scope", scope, "date", currentDate.Format(dateLayout), "error", err)
				continue
			}
		}

		// Move to next month
		currentDate = currentDate.AddDate(0, 1, 0)
	}

	return nil
}

// processResourceEmissionRecords processes resource-based emission records (with both location and resource type)
func processResourceEmissionRecords(records []armcarbonoptimization.CarbonEmissionDataClassification, scope armcarbonoptimization.EmissionScopeEnum, queryDate time.Time, recordMap map[string]*CarbonRecord) error {
	for _, sdkRecord := range records {
		resourceDetail, ok := sdkRecord.(*armcarbonoptimization.ResourceCarbonEmissionItemDetailData)
		if !ok {
			continue
		}

		// Validate required fields
		if resourceDetail.Location == nil || resourceDetail.ResourceType == nil || resourceDetail.LatestMonthEmissions == nil {
			slog.Warn("Skipping resource record with missing required fields")
			continue
		}

		// Normalize and set the location and resource type
		location := strings.ToLower(strings.TrimSpace(*resourceDetail.Location))
		resourceType := *resourceDetail.ResourceType
		// Extract the part after "/" (e.g., "microsoft.web/serverfarms" -> "serverfarms")
		if parts := strings.Split(resourceType, "/"); len(parts) > 1 {
			resourceType = strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))
		}

		// Use the query date (first day of month) as the usage month
		usageMonth := time.Date(queryDate.Year(), queryDate.Month(), 1, 0, 0, 0, 0, time.UTC)

		// Create a unique key that includes both location and resource type
		recordKey := fmt.Sprintf("%s_%s_%s", usageMonth.Format("2006-01-02"), location, resourceType)

		record := getOrCreateResourceRecord(usageMonth, location, resourceType, recordMap, recordKey)
		assignScope(scope, record, *resourceDetail.LatestMonthEmissions)
	}
	return nil
}

// getOrCreateResourceRecord gets or creates a carbon record for a specific resource
func getOrCreateResourceRecord(usageMonth time.Time, location, resourceType string, recordMap map[string]*CarbonRecord, recordKey string) *CarbonRecord {
	if record, exists := recordMap[recordKey]; exists {
		return record
	}

	record := &CarbonRecord{
		UsageMonth:   usageMonth,
		Location:     location,
		ResourceType: resourceType,
	}
	recordMap[recordKey] = record
	return record
}

func convertMapToCarbonRecords(recordMap map[string]*CarbonRecord) []CarbonRecord {
	records := make([]CarbonRecord, 0, len(recordMap))
	for _, record := range recordMap {
		records = append(records, *record)
	}

	return records
}

func assignScope(scope armcarbonoptimization.EmissionScopeEnum, record *CarbonRecord, emissions float64) {
	switch scope {
	case armcarbonoptimization.EmissionScopeEnumScope1:
		record.CarbonFootprint.Scope1 += emissions
	case armcarbonoptimization.EmissionScopeEnumScope2:
		record.CarbonFootprint.Scope2 += emissions
	case armcarbonoptimization.EmissionScopeEnumScope3:
		record.CarbonFootprint.Scope3 += emissions
	}
}
