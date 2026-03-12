# AWS Carbon Emissions Pipeline

## Overview

The AWS Customer Carbon Footprint Tool (CCFT) provides emissions data across accounts and services. Since AWS released programmatic access to the CCFT via AWS Data Exports, this data can be automatically written to S3 in structured form. Both market-based methodology (MBM) and location-based methodology (LBM) emissions are included, along with Scope 1, 2, and 3 breakdowns.

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

- Terraform >= 1.5
- AWS account with Cost Explorer and AWS Data Exports enabled
- Permissions: `s3:*`, `glue:*`, `athena:*`, `bcm-data-exports:*`, `iam:*`

## Using the Terraform module

This repo provides a reusable Terraform module that sets up the full pipeline. See [terraform/modules/aws-carbon-pipeline/](../terraform/modules/aws-carbon-pipeline/) for the module and [terraform/examples/aws-carbon-pipeline/](../terraform/examples/aws-carbon-pipeline/) for a usage example.

Copy `terraform.tfvars.example` to `terraform.tfvars`, fill in your values, then:

```bash
terraform init
terraform apply
```

The module creates all resources described in the steps below.

## Setup

### Step 1: Create an S3 bucket

Provision a dedicated S3 bucket to hold the carbon emissions exports. The module also applies bucket hygiene defaults automatically:

- **Public access block** — all public access is blocked
- **Lifecycle rules** — aborts incomplete multipart uploads after 7 days and deletes expired object delete markers to prevent silent storage accumulation
- **CloudWatch request metrics** — enabled by default, opt out with `enable_request_metrics = false`

```hcl
resource "aws_s3_bucket" "carbon" {
  bucket        = var.bucket_name
  force_destroy = false
}

resource "aws_s3_bucket_public_access_block" "carbon" {
  bucket                  = aws_s3_bucket.carbon.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_lifecycle_configuration" "carbon" {
  bucket = aws_s3_bucket.carbon.id

  rule {
    id     = "abort incomplete multi-part uploads"
    status = "Enabled"
    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }

  rule {
    id     = "delete expired object delete markers"
    status = "Enabled"
    expiration {
      expired_object_delete_marker = true
    }
  }
}
```

### Step 2: Attach a bucket policy for Data Exports

Allow the AWS Data Exports and Billing services to write to the bucket. Note: the `SourceArn` conditions reference `us-east-1` because the CUR service is only available in that region regardless of where your other resources are deployed.

```hcl
data "aws_iam_policy_document" "carbon_bucket_policy" {
  statement {
    sid    = "EnableAWSDataExportsToWriteToS3AndCheckPolicy"
    effect = "Allow"

    principals {
      type = "Service"
      identifiers = [
        "billingreports.amazonaws.com",
        "bcm-data-exports.amazonaws.com"
      ]
    }

    actions = [
      "s3:PutObject",
      "s3:GetBucketPolicy"
    ]

    resources = [
      "${aws_s3_bucket.carbon.arn}/*",
      aws_s3_bucket.carbon.arn
    ]

    condition {
      test     = "StringLike"
      variable = "aws:SourceAccount"
      values   = [var.account_id]
    }

    condition {
      test     = "StringLike"
      variable = "aws:SourceArn"
      values = [
        # CUR and BCM Data Exports are only available in us-east-1 — this is not configurable
        "arn:aws:cur:us-east-1:${var.account_id}:definition/*",
        "arn:aws:bcm-data-exports:us-east-1:${var.account_id}:export/*"
      ]
    }
  }
}
```

### Step 3: Enable Data Exports for the Carbon Emissions table

Configure AWS to export the Carbon Emissions table to S3 in Parquet format. The query includes the full emissions breakdown: MBM, LBM, and Scope 1/2/3 fields.

```hcl
resource "aws_bcmdataexports_export" "carbon" {
  export {
    name = var.export_name

    data_query {
      query_statement = "SELECT last_refresh_timestamp, location, model_version, payer_account_id, product_code, region_code, total_lbm_emissions_unit, total_lbm_emissions_value, total_mbm_emissions_unit, total_mbm_emissions_value, total_scope_1_emissions_unit, total_scope_1_emissions_value, total_scope_2_lbm_emissions_unit, total_scope_2_lbm_emissions_value, total_scope_2_mbm_emissions_unit, total_scope_2_mbm_emissions_value, total_scope_3_lbm_emissions_unit, total_scope_3_lbm_emissions_value, total_scope_3_mbm_emissions_unit, total_scope_3_mbm_emissions_value, usage_account_id, usage_period_end, usage_period_start FROM CARBON_EMISSIONS"

      table_configurations = {
        "CARBON_EMISSIONS" = {}
      }
    }

    destination_configurations {
      s3_destination {
        s3_bucket = aws_s3_bucket.carbon.id
        s3_prefix = var.s3_prefix
        s3_region = var.region

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

AWS Glue scans the S3 data, infers its schema, and registers it in the Glue Data Catalog for Athena to query. The IAM role uses the AWS-managed `AWSGlueServiceRole` policy (for CloudWatch and catalog access) plus a scoped S3 policy for the carbon bucket.

```hcl
resource "aws_iam_role" "glue_service_role" {
  name               = var.glue_role_name
  assume_role_policy = data.aws_iam_policy_document.glue_assume_role.json
}

resource "aws_iam_role_policy_attachment" "glue_service_role" {
  role       = aws_iam_role.glue_service_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSGlueServiceRole"
}

resource "aws_glue_catalog_database" "carbon" {
  name = var.glue_database_name
}

resource "aws_glue_crawler" "carbon" {
  name          = var.glue_crawler_name
  role          = aws_iam_role.glue_service_role.arn
  database_name = aws_glue_catalog_database.carbon.name
  schedule      = var.crawler_schedule

  s3_target {
    path = "s3://${aws_s3_bucket.carbon.id}/${var.s3_prefix}/${var.export_name}/data"
  }

  schema_change_policy {
    delete_behavior = "LOG"
    update_behavior = "LOG"
  }
}
```

> The crawler schedule defaults to `cron(00 08 15-26 * ? *)`, running between the 15th and 26th of each month, aligned with AWS's monthly emissions data refresh cadence.

### Step 5: Query with Athena

With the schema registered in the Glue Data Catalog, the Carbon Emissions data is queryable via Athena using standard SQL. Glue stores only the table definitions; the underlying Parquet files remain in S3.

### Step 6: Visualize in Grafana

Connect Grafana to Athena using the [Athena data source plugin](https://grafana.com/grafana/plugins/grafana-athena-datasource/). Once connected, build dashboards that break down emissions by region, account, or service alongside existing performance and cost metrics.

## Data Fields

| Field | Description |
|---|---|
| `total_mbm_emissions_value` | Market-based methodology emissions (for external reporting) |
| `total_lbm_emissions_value` | Location-based methodology emissions (for infrastructure decisions) |
| `total_scope_1_emissions_value` | Direct emissions from owned/controlled sources |
| `total_scope_2_mbm_emissions_value` | Indirect MBM emissions from purchased energy |
| `total_scope_2_lbm_emissions_value` | Indirect LBM emissions from purchased energy |
| `total_scope_3_mbm_emissions_value` | MBM value chain emissions |
| `total_scope_3_lbm_emissions_value` | LBM value chain emissions |
| `region_code` | AWS region |
| `product_code` | AWS service |
| `usage_account_id` | AWS account |
| `usage_period_start` / `usage_period_end` | Billing period |

## Troubleshooting

- **No data in S3**: Ensure Cost Explorer and AWS Data Exports are enabled. Initial activation can take up to 24 hours.
- **Glue crawler finds no tables**: Verify the S3 path in the crawler matches the actual export prefix.
- **Athena query returns no rows**: Confirm the crawler has run at least once after data landed in S3.
- **Stale data**: CCFT data is refreshed monthly; queries within the same billing period will return the same values.
