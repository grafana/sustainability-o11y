package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sustainability"
	"github.com/aws/aws-sdk-go-v2/service/sustainability/types"
)

// WaterRecord is one (year, region, service) data point as returned by the
// AWS Sustainability API.
type WaterRecord struct {
	Year          int
	Region        string
	Service       string
	ModelVersion  string
	WithdrawalsM3 float64
}

// WaterAllocationClient wraps the AWS Sustainability API's water allocation
// operation.
//
// Credentials come entirely from the default SDK credential chain. The pod's
// IRSA setup (terraform/security/aws/management, mirroring yace.tf) sets
// AWS_ROLE_ARN and AWS_WEB_IDENTITY_TOKEN_FILE; config.LoadDefaultConfig picks
// those up via AssumeRoleWithWebIdentity automatically. No custom STS code
// belongs here.
type WaterAllocationClient struct {
	api *sustainability.Client
}

func NewWaterAllocationClient(ctx context.Context, region string) (*WaterAllocationClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return &WaterAllocationClient{api: sustainability.NewFromConfig(cfg)}, nil
}

// FetchWaterWithdrawals returns one row per (year, region, service)
// combination, covering queryYears calendar years ending with the current
// year. Granularity is fixed to YEARLY_CALENDAR because it's the only
// granularity the API supports for water allocation.
func (c *WaterAllocationClient) FetchWaterWithdrawals(ctx context.Context, queryYears int) ([]WaterRecord, error) {
	if queryYears <= 0 {
		queryYears = 1
	}

	now := time.Now().UTC()
	start := time.Date(now.Year()-queryYears+1, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, time.UTC)

	input := &sustainability.GetEstimatedWaterAllocationInput{
		TimePeriod: &types.TimePeriod{
			Start: &start,
			End:   &end,
		},
		Granularity: types.TimeGranularityYearlyCalendar,
		GroupBy:     []types.Dimension{types.DimensionRegion, types.DimensionService},
	}

	var records []WaterRecord
	paginator := sustainability.NewGetEstimatedWaterAllocationPaginator(c.api, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch water allocation page: %w", err)
		}

		for _, result := range page.Results {
			record, ok := toWaterRecord(result)
			if !ok {
				continue
			}
			records = append(records, record)
		}
	}

	return records, nil
}

// toWaterRecord converts one API result into a WaterRecord.
//
// NOTE: the mapping from types.Dimension enum values ("REGION", "SERVICE") to
// DimensionsValues map keys is inferred from the SDK's naming convention, not
// confirmed against a live response — the API hasn't been callable from this
// environment. Verify this against a real GetEstimatedWaterAllocation
// response before relying on it.
func toWaterRecord(result types.EstimatedWaterAllocation) (WaterRecord, bool) {
	if result.TimePeriod == nil || result.TimePeriod.Start == nil {
		return WaterRecord{}, false
	}

	allocation, ok := result.AllocationValues[string(types.WaterAllocationTypeTotalWaterWithdrawals)]
	if !ok {
		return WaterRecord{}, false
	}

	return WaterRecord{
		Year:          result.TimePeriod.Start.Year(),
		Region:        result.DimensionsValues[string(types.DimensionRegion)],
		Service:       result.DimensionsValues[string(types.DimensionService)],
		ModelVersion:  aws.ToString(result.ModelVersion),
		WithdrawalsM3: aws.ToFloat64(allocation.Value),
	}, true
}
