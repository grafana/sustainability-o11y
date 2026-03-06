output "service_account_email" {
  description = "Email of the service account used by the Data Transfer."
  value       = google_service_account.gcp_climate_data.email
}

output "bigquery_dataset_id" {
  description = "ID of the BigQuery dataset receiving the carbon footprint export."
  value       = google_bigquery_dataset.gcp_carbon_footprint.dataset_id
}

output "data_transfer_name" {
  description = "Resource name of the BigQuery Data Transfer config."
  value       = google_bigquery_data_transfer_config.gcp_carbon_footprint_transfer.name
}
