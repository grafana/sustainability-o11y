module "gcp_carbon_pipeline" {
  source = "../../modules/gcp-carbon-pipeline"

  # Required
  project_id          = "my-gcp-project"
  billing_account_ids = ["ABCDEF-123456-ABCDEF"]
  org_id              = "123456789012"

  # Optional - Override module defaults
  # dataset_id             = "gcp_carbon_footprint"
  # dataset_location       = "US"
  # data_transfer_location = "US"
  # service_account_id     = "gcp-climate-data"

  # Optional - Enable permissions for service account for Grafana access
  # Create new service account
  # grafana_bigquery_data_source  = true
  # Use existing service account
  # grafana_service_account_email = "grafana@my-gcp-project.iam.gserviceaccount.com"

  # Optional - Additional dataset access
  # additional_dataset_access = [
  #   { role = "READER", user_by_email = "other@my-gcp-project.iam.gserviceaccount.com" }
  # ]
}
