# AWS Carbon Emissions Pipeline

## Overview

The AWS Customer Carbon Footprint Tool (CCFT) provides emissions data across accounts and services. Since AWS released programmatic access to the CCFT via AWS Data Exports, this data can be automatically written to S3 in structured form. Both market-based methodology (MBM) and location-based methodology (LBM) emissions are included.

This pipeline connects:

```
AWS CCFT (Data Exports)
        │
        ▼
    Amazon S3
        │
        ▼
   AWS Glue (crawler + catalog)
        │
        ▼
    Amazon Athena
        │
        ▼
    Grafana
```

## Prerequisites

- Terraform >= 1.0
- AWS account with Cost Explorer and AWS Data Exports enabled
- Permissions: `s3:*`, `glue:*`, `athena:*`, `bcm-data-exports:*`, `iam:*`

## Setup

### Step 1: Create an S3 bucket

Provision a dedicated S3 bucket to hold the carbon emissions exports.

```hcl
resource "aws_s3_bucket" "carbon_data_exports" {
  bucket = "carbon-data-exports"
}
```

### Step 2: Attach a bucket policy for Data Exports

Allow the AWS Data Exports and Billing services to write to the bucket.

```hcl
data "aws_iam_policy_document" "carbon_exports_policy" {
  statement {
    sid    = "EnableAWSDataExportsToWriteToS3AndCheckPolicy"
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = [
        "billingreports.amazonaws.com",
        "bcm-data-exports.amazonaws.com"
      ]
    }

    actions = [
      "s3:PutObject",      # allow AWS to write carbon exports
      "s3:GetBucketPolicy" # allow AWS to validate bucket policy
    ]

    resources = [
      "${aws_s3_bucket.carbon_data_exports.arn}/*",
      aws_s3_bucket.carbon_data_exports.arn
    ]

    # Optional: restrict access to your specific AWS account
    condition {
      test     = "StringLike"
      variable = "aws:SourceAccount"
      values   = [local.account_id]
    }

    condition {
      test     = "StringLike"
      variable = "aws:SourceArn"
      values   = [
        "arn:aws:cur:us-east-1:${local.account_id}:definition/*",
        "arn:aws:bcm-data-exports:us-east-1:${local.account_id}:export/*"
      ]
    }
  }
}

resource "aws_s3_bucket_policy" "carbon_data_exports_policy" {
  bucket = aws_s3_bucket.carbon_data_exports.id
  policy = data.aws_iam_policy_document.carbon_exports_policy.json
}
```

### Step 3: Enable Data Exports for the Carbon Emissions table

Configure AWS to export the Carbon Emissions table to S3. AWS will automatically refresh and deliver updated data on a regular cadence.

```hcl
resource "aws_bcmdataexports_export" "carbon" {
  export {
    name = "carbon-data-exports"
    data_query {
      query_statement = <<EOT
        SELECT
          last_refresh_timestamp,
          location,
          model_version,
          payer_account_id,
          product_code,
          region_code,
          total_mbm_emissions_unit,
          total_mbm_emissions_value,
          total_lbm_emissions_unit,
          total_lbm_emissions_value,
          usage_account_id,
          usage_period_end,
          usage_period_start
        FROM CARBON_EMISSIONS
      EOT

      table_configurations = {
        "CARBON_EMISSIONS" = {}
      }
    }

    destination_configurations {
      s3_destination {
        s3_bucket = aws_s3_bucket.carbon_data_exports.id
        s3_prefix = "exports"
        s3_region = "us-east-2"

        s3_output_configurations {
          overwrite   = "OVERWRITE_REPORT"
          format      = "PARQUET"
          compression = "PARQUET"
          output_type = "CUSTOM"
        }
      }
    }

    refresh_cadence {
      frequency = "SYNCHRONOUS"
    }
  }
}
```

### Step 4: Connect the bucket to AWS Glue

AWS Glue scans the S3 data, infers its schema, and registers it in the Glue Data Catalog for Athena to query.

First, create an IAM role that Glue can assume:

```hcl
resource "aws_iam_role" "glue_service_role" {
  name = "glue-carbon-crawler-service-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { Service = "glue.amazonaws.com" }
        Action    = "sts:AssumeRole"
      }
    ]
  })
}

resource "aws_iam_role_policy" "glue_role_policy" {
  name = "glue-carbon-crawler-policy"
  role = aws_iam_role.glue_service_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = ["s3:GetObject", "s3:ListBucket"]
        Resource = [
          aws_s3_bucket.carbon_data_exports.arn,
          "${aws_s3_bucket.carbon_data_exports.arn}/*"
        ]
      }
    ]
  })
}
```

Then create the Glue crawler:

```hcl
resource "aws_glue_catalog_database" "carbon" {
  name = "carbon"
}

resource "aws_glue_crawler" "carbon" {
  name          = "carbon-crawler"
  role          = aws_iam_role.glue_service_role.arn
  database_name = aws_glue_catalog_database.carbon.name

  s3_target {
    path = "s3://${aws_s3_bucket.carbon_data_exports.id}/exports/carbon/data"
  }

  schema_change_policy {
    delete_behavior = "LOG"
    update_behavior = "LOG"
  }

  schedule = "cron(00 08 15-26 * ? *)"
}
```

> The crawler schedule runs between the 15th and 26th of each month, aligned with AWS's monthly emissions data refresh cadence.

### Step 5: Query with Athena

With the schema registered in the Glue Data Catalog, the Carbon Emissions data is queryable via Athena using standard SQL. Glue stores only the table definitions; the underlying Parquet files remain in S3.

### Step 6: Visualize in Grafana

Connect Grafana to Athena using the [Athena data source plugin](https://grafana.com/grafana/plugins/grafana-athena-datasource/). Once connected, import the dashboard from [dashboard/carbon-emissions.json](../dashboard/carbon-emissions.json) to visualize emissions broken down by region, account, or service alongside existing performance and cost metrics.

## Metrics / Data Fields

| Field | Description |
|---|---|
| `total_mbm_emissions_value` | Market-based methodology emissions (for external reporting) |
| `total_lbm_emissions_value` | Location-based methodology emissions (for infrastructure decisions) |
| `region_code` | AWS region |
| `product_code` | AWS service |
| `usage_account_id` | AWS account |
| `usage_period_start` / `usage_period_end` | Billing period |

## Troubleshooting

- **No data in S3**: Ensure Cost Explorer and AWS Data Exports are enabled. Initial activation can take up to 24 hours.
- **Glue crawler finds no tables**: Verify the S3 path in the crawler matches the actual export prefix.
- **Athena query returns no rows**: Confirm the crawler has run at least once after data landed in S3.
- **Stale data**: CCFT data is refreshed monthly; queries within the same billing period will return the same values.
