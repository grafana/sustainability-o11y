# Azure Carbon Emissions Exporter

An exporter that fetches Azure carbon emissions data using the official Azure SDK and provides a simplified JSON API for carbon footprint analysis.

## Features

- **Azure SDK Integration**: Uses the official Azure Carbon Optimization SDK for reliable API access
- **Scope Breakdown**: Provides detailed breakdown of emissions by Scope 1, 2, and 3
- **Resource-Level Data**: Queries individual Azure resources with both location and resource type dimensions
- **Flexible Authentication**: Supports Azure Service Principal and default credential chain
- **Operational Metrics**: Exposes operational metrics for monitoring exporter health
- **Date Range Queries**: Automatically queries available date ranges from the API with configurable lookback period
- **BigQuery Export**: Optional export of carbon emissions data to Google BigQuery for analysis and reporting

## Quick Start

### Prerequisites

- Go 1.21+ (for building from source)
- Azure credentials with Carbon Optimization API access
- BigQuery dataset pre-created via Terraform (for BigQuery export)

### Installation

#### From Source
```bash
make build-binary
```

#### Using Docker
```bash
make build
```

## Usage

### Basic Usage

#### With CLI Flags
```bash
./azure-carbon-exporter \
  --azure.tenant-id=your-tenant-id \
  --azure.client-id=your-client-id \
  --azure.client-secret=your-client-secret
```

#### With Environment Variables
```bash
export AZURE_TENANT_ID=your-tenant-id
export AZURE_CLIENT_ID=your-client-id
export AZURE_CLIENT_SECRET=your-client-secret

./azure-carbon-exporter
```

#### With Azure CLI Authentication
```bash
az login
./azure-carbon-exporter
```

### Advanced Configuration

#### Query Specific Subscriptions
```bash
./azure-carbon-exporter \
  --subscriptions=subscription-id-1,subscription-id-2,subscription-id-3
```

#### With BigQuery Export
```bash
./azure-carbon-exporter \
  --azure.tenant-id=your-tenant-id \
  --azure.client-id=your-client-id \
  --azure.client-secret=your-client-secret \
  --subscriptions=subscription-id-1,subscription-id-2 \
  --bigquery.project-id=my-gcp-project \
  --bigquery.dataset=carbon_emissions \
  --bigquery.table=azure_carbon_data \
  --bigquery.credentials=/path/to/gcp-credentials.json
```

### Docker Usage

#### Basic Docker Run
```bash
docker run \
  -e AZURE_TENANT_ID=your-tenant-id \
  -e AZURE_CLIENT_ID=your-client-id \
  -e AZURE_CLIENT_SECRET=your-client-secret \
  azure-carbon-exporter:latest \
  --subscriptions="your-subscription-id" \
  --log-level=debug
```

## Configuration Options

### Required Parameters

- **Azure Authentication**: One of the following:
  - `--azure.tenant-id`, `--azure.client-id`, `--azure.client-secret`
  - Environment variables: `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`
  - Azure CLI authentication (`az login`)
  - Managed Identity (when running on Azure)

### Optional Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `--subscriptions` | Required | Comma-separated list of subscription IDs to query |
| `--query-months` | `2` | Number of months to query from the end of available date range (set to `0` to query full available range) |
| `--prom.pushgateway.url` | - | Prometheus Push Gateway URL (optional) |
| `--prom.pushgateway.job` | `azure-carbon-exporter` | Push Gateway job name |
| `--bigquery.project-id` | - | BigQuery project ID (enables BigQuery export) |
| `--bigquery.dataset` | - | BigQuery dataset ID |
| `--bigquery.table` | - | BigQuery table ID |
| `--bigquery.credentials` | - | Path to BigQuery credentials JSON file (or use GOOGLE_APPLICATION_CREDENTIALS env var) |
| `--bigquery.create-table` | `true` | Create BigQuery table if it doesn't exist |
| `--bigquery.batch-size` | `100` | BigQuery batch size for streaming inserts |
| `--bigquery.enabled` | `true` | Enable BigQuery export (set to false to disable even if credentials are provided) |
| `--log-level` | `info` | Log level: `debug`, `info`, `warn`, `error` |

## Metrics

The exporter provides the following Prometheus metrics for operational monitoring:

### Operational Metrics

- `azure_carbon_exporter_runs_total` - Total number of exporter runs
- `azure_carbon_exporter_records_processed_total` - Total number of carbon emission records processed
- `azure_carbon_exporter_errors_total` - Total number of errors encountered
- `azure_carbon_exporter_api_calls_total` - Total number of Azure Carbon API calls
- `azure_carbon_exporter_bigquery_uploads_total` - Total number of BigQuery uploads
- `azure_carbon_exporter_last_run_timestamp` - Timestamp of the last run
- `azure_carbon_exporter_processing_duration_seconds` - Duration of the last processing run in seconds
- `azure_carbon_exporter_latest_record_timestamp_seconds` - Latest record date timestamp observed while processing
- `azure_carbon_exporter_operation_duration_seconds` - Duration of different operations (histogram with `operation` label)

## Query Optimization

By default, the exporter queries only the last 2 months of available data to reduce Azure API calls and BigQuery operation costs. This is suitable for regular incremental updates since Azure carbon data is published monthly. The `--query-months` flag controls this behavior:

- **Default (`--query-months=2`)**: Queries the last 2 months from the end of available date range
- **Full range (`--query-months=0`)**: Queries all available historical data (useful for initial data load)

The exporter uses MERGE operations in BigQuery, so re-querying recent months is safe and will update any changed data.

## BigQuery Integration

When BigQuery export is enabled, the exporter will automatically create a table with the following schema and export carbon emissions data to it:

### Table Schema

| Column | Type | Description |
|--------|------|-------------|
| `usage_month` | DATE | First day of the month for which emissions are reported (e.g., 2025-01-01) |
| `scope` | STRING | Emission scope: "Scope 1", "Scope 2", or "Scope 3" |
| `location` | STRING | Azure region/location (e.g., "west europe", "east us") - normalized to lowercase |
| `account_id` | STRING | Azure subscription ID (cloud account/business unit) |
| `resource_type` | STRING | Azure resource type (e.g., "virtualmachines", "storageaccounts") - extracted from full type (e.g., "microsoft.compute/virtualmachines" → "virtualmachines") |
| `emissions_kg_co2e` | FLOAT64 | Carbon emissions for that dimension in kg CO2e (zero emissions are excluded) |
| `record_id` | STRING | Unique identifier for deduplication (account_id + usage_month + scope + location + resource_type) |
| `last_updated` | TIMESTAMP | When the record was last updated (with full timezone precision) |

### BigQuery Features

- **Deduplication**: Automatic deduplication using unique `record_id` prevents duplicate entries
- **MERGE Operations**: Uses UPSERT logic to handle data updates and prevent duplicates
- **Persistent Staging Table**: Uses a reusable staging table (`tablename_staging`) that gets truncated instead of created/deleted each run
- **Partitioning**: Table is partitioned by `usage_month` for efficient querying
- **Automatic Table Creation**: Tables are created automatically if they don't exist (dataset must be pre-created via Terraform)
- **Batch Processing**: Records are inserted in configurable batches for optimal performance
- **Timeout Handling**: All BigQuery operations have timeouts to prevent hanging

### Environment Variables

You can also configure BigQuery using environment variables instead of command-line flags:

- `BIGQUERY_PROJECT_ID` - BigQuery project ID
- `BIGQUERY_DATASET` - BigQuery dataset ID  
- `BIGQUERY_TABLE` - BigQuery table ID
- `GOOGLE_APPLICATION_CREDENTIALS` - Path to GCP service account credentials JSON file

### Data Output

The exporter provides a simplified JSON API that returns carbon emissions data in the following format:

```json
{
  "records": [
    {
      "usageMonth": "2024-08-01",
      "location": "west europe",
      "resourceType": "virtualmachines",
      "carbonFootprint": {
        "scope1": 123.45,
        "scope2": 678.90,
        "scope3": 234.56
      }
    }
  ]
}
```

## Data Structure

### CarbonRecord
Each carbon record contains:
- **usageMonth**: The month of the emissions data (YYYY-MM-DD format)
- **location**: Azure region/location (e.g., "west europe", "east us") - normalized to lowercase
- **resourceType**: Azure resource type (e.g., "virtualmachines", "storageaccounts") - extracted from full type path
- **carbonFootprint**: Breakdown of emissions by scope (aggregated from all resources with the same location/resource type)
  - **scope1**: Direct emissions from owned or controlled sources (kgCO2e)
  - **scope2**: Indirect emissions from purchased energy (kgCO2e)
  - **scope3**: All other indirect emissions in the value chain (kgCO2e)

## Development

### Building
```bash
make build
```

### Testing
```bash
make test
```

### Running Locally
```bash
make local-run
```

### Docker Development
```bash
make build-image
```

## API Reference

The exporter uses the Azure Carbon Optimization API:
- **Base URL**: `https://management.azure.com`
- **API Version**: `2025-04-01`
- **Authentication**: Azure AD OAuth2
- **Report Type**: `ItemDetailsReport` with `CategoryType=Resource` to get individual resources with location and resource type
- **Documentation**: `https://learn.microsoft.com/en-us/rest/api/carbon/carbon-service`

## Querying BigQuery Data

### Example Queries

**Sum all Scope 3 emissions for a specific month:**
```sql
SELECT SUM(emissions_kg_co2e) AS total_scope3_emissions_kg_co2e
FROM `your-project.your_dataset.your_table`
WHERE usage_month = DATE('2025-10-01')
  AND scope = 'Scope 3'
```

**Breakdown by location and resource type:**
```sql
SELECT 
  location,
  resource_type,
  scope,
  SUM(emissions_kg_co2e) AS total_emissions_kg_co2e
FROM `your-project.your_dataset.your_table`
WHERE usage_month = DATE('2025-10-01')
GROUP BY location, resource_type, scope
ORDER BY total_emissions_kg_co2e DESC
```

**Monthly emissions trend by scope:**
```sql
SELECT 
  usage_month,
  scope,
  SUM(emissions_kg_co2e) AS total_emissions_kg_co2e
FROM `your-project.your_dataset.your_table`
WHERE usage_month >= DATE('2025-01-01')
GROUP BY usage_month, scope
ORDER BY usage_month DESC, scope ASC
```
