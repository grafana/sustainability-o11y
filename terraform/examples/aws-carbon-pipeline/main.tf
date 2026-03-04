module "aws_carbon_pipeline" {
  source = "../../modules/aws-carbon-pipeline"

  # Required: your 12-digit AWS account ID
  account_id = "123456789012"
  region     = "us-east-2"

  # Must be globally unique across all AWS accounts
  bucket_name = "my-org-carbon-data-exports"
  export_name = "my-org-carbon-data-exports"

  glue_role_name     = "glue-carbon-crawler-service-role"
  glue_database_name = "carbon"
  glue_crawler_name  = "carbon-crawler"

  tags = {
    team        = "my-team"
    environment = "prod"
  }
}
