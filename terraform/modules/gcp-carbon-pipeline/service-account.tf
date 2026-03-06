resource "google_service_account" "gcp_climate_data" {
  project      = var.project_id
  account_id   = var.service_account_id
  display_name = var.service_account_id
  description  = "Service account for GCP climate data exporting to BigQuery"
}
