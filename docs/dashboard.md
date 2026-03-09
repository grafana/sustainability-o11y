# Carbon Emissions Dashboard

## Overview

The Carbon Emissions Report dashboard visualises cloud carbon emissions across GCP, AWS, and Azure in a single view. It shows total emissions, breakdowns by scope, region, and service, and supports both location-based and market-based methodology comparisons.

The dashboard is built with the [Grafana Foundation SDK](https://github.com/grafana/grafana-foundation-sdk) and all data source connections and table names are configured via template variables — no code changes are needed to point it at your own data.

## Required plugins

Install these Grafana data source plugins before importing the dashboard:

| Plugin | Purpose | Install |
|--------|---------|---------|
| [BigQuery](https://grafana.com/grafana/plugins/grafana-bigquery-datasource/) | GCP and Azure emissions data | `grafana cli plugins install grafana-bigquery-datasource` |
| [Athena](https://grafana.com/grafana/plugins/grafana-athena-datasource/) | AWS emissions data | `grafana cli plugins install grafana-athena-datasource` |
| [Infinity](https://grafana.com/grafana/plugins/yesoreyeram-infinity-datasource/) | Static coordinate data for map panels | `grafana cli plugins install yesoreyeram-infinity-datasource` |

On Grafana Cloud, these plugins can be installed from the **Administration → Plugins** page.

## Prerequisites

Before configuring the dashboard, set up the data pipelines for the cloud providers you want to monitor:

- **AWS** — See [aws-pipeline.md](aws-pipeline.md). You need an Athena database and table name from the Glue catalog.
- **GCP** — See [gcp-pipeline.md](gcp-pipeline.md). You need a BigQuery project ID and dataset name.
- **Azure** — See [azure-pipeline.md](azure-pipeline.md). You need the BigQuery dataset name used when running the exporter.

You only need to configure the providers you are using. Panels for providers without a configured data source will show errors, which can be ignored.

## Configuring the dashboard

Once the dashboard is imported, open **Dashboard settings → Variables** (or use the dropdowns at the top of the dashboard) and fill in the following:

### Data source variables

These are picker dropdowns — select the matching data source connection you have configured in Grafana:

| Variable | Description |
|----------|-------------|
| `bigquery_ds` | Your BigQuery data source connection (used for GCP and Azure panels) |
| `athena_ds` | Your Athena data source connection (used for AWS panels) |
| `infinity_ds` | Your Infinity data source connection (used for map panels) |

### BigQuery configuration

| Variable | Description | Example |
|----------|-------------|---------|
| `bigquery_project` | The GCP project ID where your BigQuery datasets live | `my-gcp-project` |
| `gcp_dataset` | BigQuery dataset containing GCP carbon footprint data | `gcp_carbon_footprint` |
| `azure_dataset` | BigQuery dataset containing Azure carbon emissions data | `azure_carbon_emissions` |

The GCP dataset name defaults to `gcp_carbon_footprint`, which is the default created by the [GCP Terraform module](gcp-pipeline.md). The Azure dataset name should match the `--bigquery.dataset` flag used when running the [azure-carbon-exporter](azure-pipeline.md).

### Athena configuration

| Variable | Description | Example |
|----------|-------------|---------|
| `athena_database` | The Glue/Athena database name for AWS carbon data | `carbon` |
| `athena_table` | The table name within the Athena database | `v3_0_0_data` |

The database name matches the `glue_database_name` variable in the [AWS Terraform module](aws-pipeline.md) (defaults to `carbon`). The table name is the one created by the Glue crawler — by default this is derived from the S3 prefix and is typically `v3_0_0_data` for the AWS Carbon Footprint Tool v3 schema.

## Generating the dashboard JSON

The dashboard is defined as code using the Grafana Foundation SDK. To generate the JSON for import:

```bash
cd dashboard
go run ./export
```

This outputs `carbon_emissions_report.json` in the `dashboard/` directory, which can be imported in Grafana.
