terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }

  # Required: configure your remote state backend
  # backend "s3" {
  #   bucket         = "my-org-terraform-state"   # your state bucket
  #   key            = "carbon/aws/default.tfstate"
  #   dynamodb_table = "my-org-terraform-state-lock"
  #   region         = "us-east-2"                # region of your state bucket
  # }
}

provider "aws" {
  # Must match the region set in the module
  region = "us-east-2"

  # Uncomment and set to your AWS CLI profile if not using environment variables or instance roles
  # profile = "my-aws-profile"
}
