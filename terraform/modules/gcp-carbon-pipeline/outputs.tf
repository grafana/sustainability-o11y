output "bigquery_dataset_id" {
  description = "ID of the BigQuery dataset receiving the carbon footprint export."
  value       = google_bigquery_dataset.gcp_carbon_footprint.dataset_id
}

output "data_transfer_name" {
  description = "Resource name of the BigQuery data transfer config."
  value       = google_bigquery_data_transfer_config.gcp_carbon_footprint_transfer.name
}

output "grafana_service_account_email" {
  description = "Email of the created Grafana service account. Null if grafana_bigquery_data_source is false."
  value       = var.grafana_bigquery_data_source ? google_service_account.grafana_bigquery_data_source[0].email : null
}
