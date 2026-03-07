resource "google_service_account" "gcp_climate_data" {
  project      = var.project_id
  account_id   = var.service_account_id
  display_name = var.service_account_id
  description  = "Service account for GCP climate data exporting to BigQuery"
}

resource "google_service_account" "grafana_bigquery_data_source" {
  count        = var.grafana ? 1 : 0
  project      = var.project_id
  account_id   = "grafana-bigquery-datasource"
  display_name = "grafana-bigquery-datasource"
  description  = "Service account for Grafana to query GCP carbon footprint data in BigQuery"
}
