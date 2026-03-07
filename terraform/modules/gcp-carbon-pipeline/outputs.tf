output "grafana_service_account_email" {
  description = "Email of the created Grafana service account. Null if var.grafana_bigquery_data_source is false."
  value       = var.grafana_bigquery_data_source ? google_service_account.grafana[0].email : null
}
