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

- Terraform >= 1.0
- A GCP organization with billing account access
- Permissions on GCP project: `resourcemanager.projects.update`,
`serviceusage.services.enable`,
`bigquery.transfers.update`
- Permissions on billing account: `billing.accounts.getCarbonInformation`
- The BigQuery Data Transfer API enabled: `gcloud services enable bigquerydatatransfer.googleapis.com`

## Setup

### Step 1: Create a service account for carbon data exporting

Create a dedicated service account that will authenticate with GCP to fetch carbon emissions data and write it into BigQuery.

```hcl
resource "google_service_account" "gcp_climate_data" {
  account_id   = "gcp-climate-data"
  display_name = "gcp-climate-data"
  description  = "Service account for GCP climate data exporting to BigQuery"
}
```

### Step 2: Add roles and grant the required org-level and billing account roles

GCP's Carbon Footprint transfer requires two custom roles granted at the organization level. These are distinct from standard BigQuery roles and must be set explicitly at the org level rather than the project level.

```hcl
resource "google_organization_iam_custom_role" "bigtable_transfer_climate_data_project" {
  role_id     = "BigtableTransferClimateProject"
  org_id      = data.google_organization.grafana.org_id
  title       = "Bigtable Transfer for Climate Data (project permissions)"
  description = "Least privilege for transferring GCP climate data into Bigtable (project permissions)."
  permissions = [
    "bigquery.transfers.update",
    "resourcemanager.projects.update",
    "serviceusage.services.enable"
  ]
}

resource "google_organization_iam_custom_role" "bigtable_transfer_climate_data_billing_account" {
  role_id     = "BigtableTransferClimateBillingAccount"
  org_id      = data.google_organization.grafana.org_id
  title       = "Bigtable Transfer for Climate Data (billing account permissions)"
  description = "Least privilege for transferring GCP climate data into Bigtable (billing account permissions)."
  permissions = [
    "billing.accounts.getCarbonInformation"
  ]
}

resource "google_organization_iam_member" "climate_project_role" {
  org_id = data.google_organization.org.org_id
  role   = google_organization_iam_custom_role.bigtable_transfer_climate_data_project.id
  member = google_service_account.gcp_climate_data.member
}

resource "google_organization_iam_member" "climate_billing_role" {
  org_id = data.google_organization.org.org_id
  role   = google_organization_iam_custom_role.bigtable_transfer_climate_data_billing_account.id
  member = google_service_account.gcp_climate_data.member
}
```

### Step 3: Create a BigQuery dataset for carbon footprint data

Provision a dedicated BigQuery dataset to receive the carbon emissions exports. This dataset is the destination for the monthly transfer and allows Grafana to query it directly.

Grant the service account `OWNER` access, and grant the BigQuery Data Transfer service account `dataEditor` access so it can deliver data into the dataset.

```hcl
resource "google_bigquery_dataset" "gcp_carbon_footprint" {
  dataset_id  = "gcp_carbon_footprint"
  description = "Monthly export of GCP Carbon footprint data"

  access {
    role          = "OWNER"
    user_by_email = google_service_account.gcp_climate_data.email
  }

  access {
    role          = "roles/bigquery.dataEditor"
    user_by_email = "service-${local.project_number}@gcp-sa-bigquerydatatransfer.iam.gserviceaccount.com"
  }
}
```

### Step 4: Configure the BigQuery Data Transfer

With the dataset and permissions in place, configure the Data Transfer job. The `data_source_id` identifies GCP's built-in Carbon Footprint source, and `billing_accounts` scopes the export to your organization's billing account.

```hcl
resource "google_bigquery_data_transfer_config" "gcp_carbon_footprint_transfer" {
  display_name           = "gcp_carbon_footprint"
  data_source_id         = "61cede5a-0000-2440-ad42-883d24f8f7b8"
  destination_dataset_id = google_bigquery_dataset.gcp_carbon_footprint.dataset_id
  service_account_name   = google_service_account.gcp_climate_data.email

  params = {
    billing_accounts = local.billing_account_id
  }
}
```

> GCP exports each month's carbon data on the 15th of the following month. The transfer runs automatically on that cadence once configured.

**Known issue: enrollment step required before first apply**

Terraform may fail on first apply with:

```
Error: Error creating Config: googleapi: Error 400: BigQuery DataTransfer is not enabled
for 61cede5a-0000-2440-ad42-883d24f8f7b8.
```

This happens because the Carbon Footprint data source must be enrolled via the `EnrollDataSources` API before a transfer config can be created. This enrollment step is not documented in Google's public guides but is required. The Terraform Google provider does not yet expose a resource for this enrollment step (tracked in [hashicorp/terraform-provider-google#20217](https://github.com/hashicorp/terraform-provider-google/issues/20217)).

Enable it via gcloud before running `terraform apply`:

```bash
gcloud services enable bigquerydatatransfer.googleapis.com
```

### Step 5: Visualize in Grafana

Connect Grafana to BigQuery using the BigQuery data source plugin, pointing to your GCP project and the `gcp_carbon_footprint` dataset. Import the carbon monitoring dashboard to visualize GCP emissions broken down by project, region, and service alongside other cloud providers.

## Metrics / Data Fields

### BigQuery table schema


The transfer creates a `carbon_footprint` table in the destination dataset, partitioned by `usage_month`.

| Column | Type | Description |
|--------|------|-------------|
| `usage_month` | DATE | First day of the billing month (e.g., `2025-01-01`) |
| `billing_account_id` | STRING | The ID of the billing account | 
| `project.id` | STRING | Contains GCP project ID |
| `service.description` | STRING | Contains description of GCP service (e.g., `Compute Engine`, `Cloud Storage`)  |
| `location.location` | STRING | Contains GCP location (e.g., `us-central1-a`, `europe-west1-b`) |
| `location.region ` | STRING | Contains GCP region (e.g., `us-central1`, `europe-west1`) |
| `carbon_footprint_kgCO2e.scope1` | FLOAT | Contains Scope 1 total carbon emissions in kg CO₂e |
| `carbon_footprint_kgCO2e.scope2.location_based` | FLOAT | Contains location-based  Scope 2 total carbon emissions in kg CO₂e |
| `carbon_footprint_kgCO2e.scope2.market_based` | FLOAT | Contains market-based Scope 2 total carbon emissions in kg CO₂e |
| `carbon_footprint_kgCO2e.scope3` | FLOAT | Contains Scope 3 total carbon emissions in kg CO₂e |
| `carbon_footprint_total_kgCO2e.location_based` | FLOAT | The total emissions for all 3 scopes kg CO₂e for the account, project, service, location, and month in kg of CO2 equivalent.|
| `carbon_footprint_total_kgCO2e.market_based` | FLOAT | The total emissions for all 3 scopes for the account, project, service, location, and month in kg of CO2 equivalent. |
| `carbon_model_version ` | INT | Version of carbon model that produced this output. This value is updated whenever the model is changed. |

[Source](https://docs.cloud.google.com/carbon-footprint/docs/data-schema)



### Sample query

```sql
SELECT
  SUM(`carbon_footprint_total_kgCO2e`.`location_based`) / 1000 as total_carbon_footprint,
  -- usage_month,
  --`location`.`location`,
  `location`.`region`,
  --`project`.`id`,
  --`project`.`number`,
  --`service`.`description`,
  --`service`.`id`
FROM
  `your-gcp-project.gcp_carbon_footprint.carbon_footprint`
  GROUP BY
    -- usage_month,
    location.region
    --service.description
  ORDER BY total_carbon_footprint DESC
  LIMIT 15;

```

## Troubleshooting

- **`EnrollDataSources` error on first apply**: Enable the BigQuery Data Transfer API via `gcloud services enable bigquerydatatransfer.googleapis.com` before applying Terraform.
- **Transfer config created but no data arrives**: Verify the service account has both org-level IAM roles (`BigtableTransferClimateProject` and `BigtableTransferClimateBillingAccount`). Missing either role silently prevents data delivery.
- **Data Transfer service account permission denied**: Ensure the `service-<project-number>@gcp-sa-bigquerydatatransfer.iam.gserviceaccount.com` account has `roles/bigquery.dataEditor` on the dataset.
- **Stale data**: GCP carbon data is published monthly on the 15th of the following month. Queries within the same billing period return the same values until the next data refresh.
- **Missing projects or regions**: Carbon Footprint data is scoped to the billing account. Confirm the `billing_accounts` param matches your organization's billing account ID.
