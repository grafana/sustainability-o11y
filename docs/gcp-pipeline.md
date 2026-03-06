# GCP Carbon Emissions Pipeline

## Overview

The GCP Carbon Footprint API provides emissions data per project, region, and service across your organization. This pipeline uses GCP's built-in BigQuery Data Transfer Service to export that data monthly into BigQuery, where it can be queried by Grafana alongside emissions data from other cloud providers.

This pipeline connects:

```
GCP Carbon Footprint API
          │
          ▼
BigQuery Data Transfer Service
(monthly export, runs on the 15th)
          │
          ▼
      BigQuery
 (partitioned by usage_month)
          │
          ▼
      Grafana
 (BigQuery data source)
```

## Prerequisites

- Terraform >= 1.5
- A GCP project with billing account access
- Permissions on GCP project: `resourcemanager.projects.update`, `serviceusage.services.enable`, `bigquery.transfers.update`
- Permissions on billing account: `billing.accounts.getCarbonInformation`
- The BigQuery Data Transfer API enabled: `gcloud services enable bigquerydatatransfer.googleapis.com`

## Setup

### Step 1: Enable the BigQuery Data Transfer API

Before the first `terraform apply`, the Carbon Footprint data source must be enrolled. The Terraform Google provider does not yet support this step ([hashicorp/terraform-provider-google#20217](https://github.com/hashicorp/terraform-provider-google/issues/20217)).

```bash
gcloud services enable bigquerydatatransfer.googleapis.com --project=<project-id>
```

### Step 2: Deploy the module

Use the example in [`terraform/examples/gcp-carbon-pipeline`](../terraform/examples/gcp-carbon-pipeline) as a starting point:

```hcl
module "gcp_carbon_pipeline" {
  source = "../../modules/gcp-carbon-pipeline"
 
  project_id          = "my-gcp-project"
  billing_account_ids = ["ABCDEF-123456-ABCDEF"]
  org_id = "123456789012"

  # Optional — override module defaults
  dataset_id         = "gcp_carbon_footprint"
  dataset_location   = "us"
  service_account_id = "gcp-climate-data"
}
```

> GCP exports each month's carbon data on the 15th of the following month. The transfer runs automatically on that cadence once configured. You can also backfill data going back to 2021 with a one time transfer job.

### Step 3: Connect Grafana to BigQuery

Enable the Google BigQuery data source for Grafana with [these docs](https://grafana.com/grafana/plugins/grafana-bigquery-datasource/). 

To grant to an existing service account read access to the dataset, set `grafana_service_account_email` in the module:

```hcl
grafana_service_account_email = "grafana@my-gcp-project.iam.gserviceaccount.com"
```

If you'd like to create one within the module then set `grafana_bigquery_data_source` like this:
```hcl
grafana_bigquery_data_source = true
```

Which will create a service account and add the permissions to view. The service account email will be shared in the output.


Then you can import the carbon monitoring dashboard to visualize GCP emissions broken down by project, region, and service alongside other cloud providers.

## Module reference

### Inputs

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `project_id` | `string` | required | GCP project ID where BigQuery and Data Transfer resources will be created |
| `billing_account_ids` | `list(string)` | required | One or more GCP billing account IDs to scope the carbon footprint export |
| `org_id` | `string` | `null` | GCP organization ID. When set, creates org-level custom roles and IAM bindings required for the Data Transfer. Leave null if roles are managed externally |
| `dataset_id` | `string` | `"gcp_carbon_footprint"` | BigQuery dataset ID to create for the carbon footprint export |
| `dataset_location` | `string` | `"us"` | Location for the BigQuery dataset |
| `service_account_id` | `string` | `"gcp-climate-data"` | Account ID for the service account that runs the Data Transfer |
| `grafana_bigquery_data_source` | `bool` | `false` | When true, creates a dedicated `grafana-bigquery-datasource` service account and grants it `dataViewer` and `jobUser` roles |
| `grafana_service_account_email` | `string` | `null` | Email of an existing Grafana service account to grant BigQuery dataViewer access. Leave null to skip |
| `additional_dataset_access` | `list(object)` | `[]` | Additional IAM bindings to add to the BigQuery dataset. Each object requires `role` and `user_by_email` |

### Outputs

| Output | Description |
|--------|-------------|
| `grafana_service_account_email` | Email of the created Grafana service account. Null if `grafana_bigquery_data_source` is false |

## Metrics / Data Fields

The transfer creates a `carbon_footprint` table in the destination dataset, partitioned by `usage_month`.

| Column | Type | Description |
|--------|------|-------------|
| `usage_month` | DATE | First day of the billing month (e.g., `2025-01-01`) |
| `billing_account_id` | STRING | The ID of the billing account |
| `project.id` | STRING | GCP project ID |
| `service.description` | STRING | GCP service (e.g., `Compute Engine`, `Cloud Storage`) |
| `location.location` | STRING | GCP location (e.g., `us-central1-a`, `europe-west1-b`) |
| `location.region` | STRING | GCP region (e.g., `us-central1`, `europe-west1`) |
| `carbon_footprint_kgCO2e.scope1` | FLOAT | Scope 1 total carbon emissions in kg CO₂e |
| `carbon_footprint_kgCO2e.scope2.location_based` | FLOAT | Location-based Scope 2 total carbon emissions in kg CO₂e |
| `carbon_footprint_kgCO2e.scope2.market_based` | FLOAT | Market-based Scope 2 total carbon emissions in kg CO₂e |
| `carbon_footprint_kgCO2e.scope3` | FLOAT | Scope 3 total carbon emissions in kg CO₂e |
| `carbon_footprint_total_kgCO2e.location_based` | FLOAT | Total emissions for all 3 scopes, location-based, in kg CO₂e |
| `carbon_footprint_total_kgCO2e.market_based` | FLOAT | Total emissions for all 3 scopes, market-based, in kg CO₂e |
| `carbon_model_version` | INT | Version of the carbon model that produced this output |

[Source](https://cloud.google.com/carbon-footprint/docs/data-schema)


## Troubleshooting

- **`EnrollDataSources` error on first apply**: Enable the BigQuery Data Transfer API via `gcloud services enable bigquerydatatransfer.googleapis.com --project=<project-id>` before applying Terraform.
- **Transfer config created but no data arrives**: Verify the service account has both org-level IAM roles (`BigQueryTransferClimateProject` and `BigQueryTransferClimateBillingAccount`). Missing either role silently prevents data delivery.
- **Data Transfer service account permission denied**: Ensure the `service-<project-number>@gcp-sa-bigquerydatatransfer.iam.gserviceaccount.com` account has `WRITER` access on the dataset.
- **Stale data**: GCP carbon data is published monthly on the 15th of the following month. Queries within the same billing period return the same values until the next data refresh.
- **Missing projects or regions**: Carbon Footprint data is scoped to the billing account. Confirm the `billing_account_ids` values match your organization's billing account IDs.
