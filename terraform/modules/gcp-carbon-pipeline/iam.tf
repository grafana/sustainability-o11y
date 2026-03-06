# Permissions to initiate a carbon data transfer at the project level
resource "google_organization_iam_custom_role" "bigquery_transfer_climate_data_project" {
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
  org_id = var.org_id
  role   = google_organization_iam_custom_role.bigquery_transfer_climate_data_project.id
  member = google_service_account.gcp_climate_data.member
}

resource "google_organization_iam_member" "climate_billing_role" {
  org_id = var.org_id
  role   = "roles/billing.carbonViewer"
  member = google_service_account.gcp_climate_data.member
}
