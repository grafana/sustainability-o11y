# sustainability-o11y

This repo houses everything related to sustainability observability at Grafana: data pipelines, dashboards, and documentation for collecting and visualizing cloud carbon emissions across the Cloud Service Providers AWS, GCP, and Azure.

## Data Pipelines

### AWS

Terraform-based pipeline that exports AWS Customer Carbon Footprint Tool (CCFT) data to S3, catalogs it with AWS Glue, and makes it queryable via Athena for Grafana to consume.

See [docs/aws-pipeline.md](docs/aws-pipeline.md).

### GCP

See [docs/gcp-pipeline.md](docs/gcp-pipeline.md).

### Azure

Prometheus-compatible exporter for Azure carbon emissions data.

See [docs/azure-pipeline.md](docs/azure-pipeline.md).
