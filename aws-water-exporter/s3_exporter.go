package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Exporter writes water withdrawal records to S3 as CSV, one object per
// year. CSV rather than Parquet: the whole dataset is on the order of a
// hundred rows a year (regions x services), so a columnar format buys
// nothing here, and Glue/Athena read CSV natively.
type S3Exporter struct {
	client *s3.Client
	bucket string
	prefix string
}

func NewS3Exporter(ctx context.Context, bucket, prefix string) (*S3Exporter, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return &S3Exporter{client: s3.NewFromConfig(cfg), bucket: bucket, prefix: prefix}, nil
}

// ExportRecords writes one CSV object per year, overwriting any existing
// object for that year. Overwrite, not append: a re-run before AWS finalizes
// a year's data should replace the partial figures, not duplicate rows.
func (e *S3Exporter) ExportRecords(ctx context.Context, records []WaterRecord) error {
	byYear := make(map[int][]WaterRecord)
	for _, r := range records {
		byYear[r.Year] = append(byYear[r.Year], r)
	}

	years := make([]int, 0, len(byYear))
	for year := range byYear {
		years = append(years, year)
	}
	sort.Ints(years)

	for _, year := range years {
		body, err := encodeCSV(byYear[year])
		if err != nil {
			return fmt.Errorf("failed to encode year %d: %w", year, err)
		}

		key := fmt.Sprintf("%s/year=%d/data.csv", e.prefix, year)
		if _, err := e.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(e.bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(body),
		}); err != nil {
			return fmt.Errorf("failed to upload %s: %w", key, err)
		}
	}

	return nil
}

func encodeCSV(records []WaterRecord) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if err := w.Write([]string{"year", "region", "service", "model_version", "total_water_withdrawals_m3"}); err != nil {
		return nil, err
	}

	for _, r := range records {
		row := []string{
			strconv.Itoa(r.Year),
			r.Region,
			r.Service,
			r.ModelVersion,
			strconv.FormatFloat(r.WithdrawalsM3, 'f', -1, 64),
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}
