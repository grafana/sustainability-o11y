data "google_project" "project" {
  project_id = var.project_id
}

resource "google_bigquery_dataset" "gcp_carbon_footprint" {
  project     = var.project_id
  dataset_id  = var.dataset_id
  description = "Monthly export of GCP Carbon footprint data"
  location    = var.dataset_location

  # Service account responsible for the data transfer
  access {
    role          = "OWNER"
    user_by_email = google_service_account.gcp_climate_data.email
  }

  # Service account that needs write access to deliver data into this dataset
  access {
    role          = "WRITER"
    user_by_email = "service-${data.google_project.project.number}@gcp-sa-bigquerydatatransfer.iam.gserviceaccount.com"
  }

  dynamic "access" {
    for_each = var.grafana_bigquery_data_source ? [google_service_account.grafana_bigquery_data_source[0].email] : []
    content {
      role          = "READER"
      user_by_email = access.value
    }
  }

  # Optional: grant an existing Grafana service account dataViewer access
  dynamic "access" {
    for_each = var.grafana_service_account_email != null ? [var.grafana_service_account_email] : []
    content {
      role          = "READER"
      user_by_email = access.value
    }
  }

  # Optional: additional IAM bindings (e.g. for other readers or tooling)
  dynamic "access" {
    for_each = var.additional_dataset_access
    content {
      role           = access.value.role
      user_by_email  = access.value.user_by_email
      group_by_email = access.value.group_by_email
    }
  }
}

# NOTE: Before the first apply, the Carbon Footprint data source must be enrolled.
# The Terraform Google provider does not yet support this step
# (https://github.com/hashicorp/terraform-provider-google/issues/20217).
# Run the following before applying:
#
#   gcloud services enable bigquerydatatransfer.googleapis.com --project=<project-id>

resource "google_bigquery_data_transfer_config" "gcp_carbon_footprint_transfer" {
  project                = var.project_id
  display_name           = var.dataset_id
  data_source_id         = "61cede5a-0000-2440-ad42-883d24f8f7b8"
  destination_dataset_id = google_bigquery_dataset.gcp_carbon_footprint.dataset_id
  service_account_name   = google_service_account.gcp_climate_data.email

  params = {
    billing_accounts = join(",", var.billing_account_ids)
  }
}
