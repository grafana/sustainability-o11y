module "gcp_carbon_pipeline" {
  source = "../../modules/gcp-carbon-pipeline"

  # Required
  project_id = "my-gcp-project"

  org_id              = "123456789012"
  billing_account_ids = ["ABCDEF-123456-ABCDEF"]

  # Optional — override module defaults
  dataset_id         = "gcp_carbon_footprint"
  dataset_location   = "us"
  service_account_id = "gcp-climate-data"
}
