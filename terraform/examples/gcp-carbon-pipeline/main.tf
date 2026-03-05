module "gcp_carbon_pipeline" {
  source = "../../modules/gcp-carbon-pipeline"

  # Required
  org_id             = "123456789012"
  project_id         = "my-gcp-project"
  billing_account_id = "ABCDEF-123456-ABCDEF"

  # Optional — override module defaults
  dataset_id         = "gcp_carbon_footprint"
  dataset_location   = "US"
  service_account_id = "gcp-climate-data"
}
