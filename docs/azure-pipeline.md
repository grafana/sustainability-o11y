# Azure Carbon Emissions Pipeline

## Overview

The Azure Carbon Optimization API provides emissions data per resource, region, and scope across Azure subscriptions. This pipeline uses a custom exporter to pull that data and write it to BigQuery, where it can be queried by Grafana alongside emissions data from other cloud providers.

This pipeline connects:

```
Azure Carbon Optimization API
           │
           ▼
  azure-carbon-exporter
  (Docker / binary, run as a job)
           │
           ▼
       BigQuery
  (MERGE operations, partitioned
   by usage_month)
           │
           ▼
       Grafana
  (BigQuery data source)
```

## Prerequisites

- Azure subscription(s) with the [Carbon Optimization API](https://learn.microsoft.com/en-us/rest/api/carbon/carbon-service) enabled
- An Azure service principal or other credential with access to the Carbon Optimization API
- A GCP project with a BigQuery dataset pre-created (the exporter creates the table, but not the dataset)
- Docker or Go 1.21+ to run the exporter

## Setup

### Step 1: Create an Azure service principal

Create an Azure AD application and service principal, then assign it the **Carbon Optimization Reader** role at the subscription level. This is the minimum permission required by the Carbon Optimization API.

```hcl
resource "azuread_application" "azure_carbon_export" {
  display_name = "azure-carbon-export"
}

resource "azuread_service_principal" "azure_carbon_export" {
  client_id = azuread_application.azure_carbon_export.client_id
}

resource "azuread_application_password" "azure_carbon_export" {
  application_id = azuread_application.azure_carbon_export.id
}

resource "azurerm_role_assignment" "azure_carbon_export_reader" {
  scope                = "/subscriptions/${var.subscription_id}"
  role_definition_name = "Carbon Optimization Reader"
  principal_id         = azuread_service_principal.azure_carbon_export.id
}
```

### Step 2: Create a BigQuery dataset

The exporter automatically creates the table, but the dataset must be provisioned in advance. Grant the service account running the exporter `OWNER` and `roles/bigquery.user` on the dataset so it can create and write to the table.

```hcl
resource "google_bigquery_dataset" "azure_carbon_emissions" {
  dataset_id  = "azure_carbon_emissions"
  description = "Azure carbon emissions data exported to BigQuery"
}
```

### Step 3: Run the exporter

The exporter is a short-lived job (not a long-running server). Run it on a schedule (e.g., via cron, Kubernetes CronJob, or CI) to keep BigQuery up to date. Azure carbon data is published monthly, so weekly or monthly runs are sufficient.

#### Using Docker

Build the image from source (see [`azure-carbon-exporter/`](../azure-carbon-exporter/)):

```bash
cd azure-carbon-exporter && make build-image
```

Then run it:

```bash
docker run \
  -e AZURE_TENANT_ID=your-tenant-id \
  -e AZURE_CLIENT_ID=your-client-id \
  -e AZURE_CLIENT_SECRET=your-client-secret \
  azure-carbon-exporter:latest \
  --subscriptions=subscription-id-1,subscription-id-2 \
  --bigquery.project-id=my-gcp-project \
  --bigquery.dataset=azure_carbon_emissions_prod \
  --bigquery.table=azure_carbon_data_prod \
  --bigquery.credentials=/path/to/gcp-credentials.json
```

#### Using environment variables for all configuration

```bash
export AZURE_TENANT_ID=your-tenant-id
export AZURE_CLIENT_ID=your-client-id
export AZURE_CLIENT_SECRET=your-client-secret
export BIGQUERY_PROJECT_ID=my-gcp-project
export BIGQUERY_DATASET=azure_carbon_emissions_prod
export BIGQUERY_TABLE=azure_carbon_data_prod
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/gcp-credentials.json

./azure-carbon-exporter --subscriptions=subscription-id-1,subscription-id-2
```

#### Key configuration options

| Flag | Default | Description |
|------|---------|-------------|
| `--subscriptions` | required | Comma-separated list of Azure subscription IDs |
| `--query-months` | `2` | Months of history to query per run; set to `0` for full history (useful for initial load) |
| `--bigquery.create-table` | `true` | Create the BigQuery table if it doesn't exist |
| `--bigquery.batch-size` | `100` | Batch size for BigQuery streaming inserts |
| `--prom.pushgateway.url` | — | Optional Prometheus Push Gateway URL for operational metrics |
| `--log-level` | `info` | Log level: `debug`, `info`, `warn`, `error` |

> By default the exporter queries only the last 2 months. This is safe to re-run because BigQuery writes use MERGE (UPSERT) semantics — existing records are updated rather than duplicated.

### Step 4: Connect Grafana to BigQuery

Add the BigQuery data source in Grafana pointing to the GCP project and dataset. The dashboard queries use standard SQL against the table created by the exporter.

### Step 5: Visualize in Grafana

Import the carbon monitoring dashboard from the `grafana-foundation-sdk` to visualize Azure emissions broken down by scope, region, and resource type alongside other cloud providers.

## Metrics / Data Fields

### BigQuery table schema

| Column | Type | Description |
|--------|------|-------------|
| `usage_month` | DATE | First day of the billing month (e.g., `2025-01-01`) |
| `scope` | STRING | Emission scope: `Scope 1`, `Scope 2`, or `Scope 3` |
| `location` | STRING | Azure region, normalized to lowercase (e.g., `west europe`, `east us`) |
| `account_id` | STRING | Azure subscription ID |
| `resource_type` | STRING | Azure resource type, extracted from the full type path (e.g., `virtualmachines`, `storageaccounts`) |
| `emissions_kg_co2e` | FLOAT64 | Carbon emissions in kg CO₂e |
| `record_id` | STRING | Unique key used for deduplication (account + month + scope + location + resource type) |
| `last_updated` | TIMESTAMP | When the record was last written |

The table is partitioned by `usage_month` for efficient time-range queries.

### Emission scopes

| Scope | Description |
|-------|-------------|
| `Scope 1` | Direct emissions from Azure-owned or controlled sources |
| `Scope 2` | Indirect emissions from purchased electricity and energy |
| `Scope 3` | All other indirect emissions in the value chain |

## Troubleshooting

- **API returns no data**: Ensure the Carbon Optimization API is enabled for the subscription and the service principal has the correct role assignment. Note that Carbon Optimization data may lag by 1–2 months.
- **`Not found: Dataset` error**: The BigQuery dataset must be created before running the exporter. The exporter creates the table but not the dataset.
- **Duplicate records**: Re-runs are safe — the exporter uses `MERGE` (UPSERT) operations keyed on `record_id`, so records are updated rather than duplicated.
- **Missing regions or resource types**: The exporter normalizes locations to lowercase and extracts the short resource type (the part after `/`). Verify filters in Grafana match the normalized values.
- **Stale data**: Azure carbon data is published on a monthly cadence. Queries within the same billing period return the same values until the next data refresh.
