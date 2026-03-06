resource "google_organization_iam_custom_role" "bigquery_transfer_climate_data_project" {
  count       = var.org_id != null ? 1 : 0
  role_id     = "BigQueryTransferClimateProject"
  org_id      = var.org_id
  title       = "BigQuery Transfer for Climate Data (project permissions)"
  description = "Least privilege for transferring GCP climate data into BigQuery (project permissions)."
  permissions = [
    "bigquery.transfers.update",
    "resourcemanager.projects.update",
    "serviceusage.services.enable",
  ]
}

resource "google_organization_iam_member" "climate_project_role" {
  count  = var.org_id != null ? 1 : 0
  org_id = var.org_id
  role   = google_organization_iam_custom_role.bigquery_transfer_climate_data_project[0].id
  member = google_service_account.gcp_climate_data.member
}

resource "google_organization_iam_member" "climate_billing_role" {
  count  = var.org_id != null ? 1 : 0
  org_id = var.org_id
  role   = "roles/billing.carbonViewer"
  member = google_service_account.gcp_climate_data.member
}

resource "google_project_iam_member" "grafana_bigquery_job_user" {
  count   = var.grafana_bigquery_data_source ? 1 : 0
  project = var.project_id
  role    = "roles/bigquery.jobUser"
  member  = google_service_account.grafana_bigquery_data_source[0].member
}
