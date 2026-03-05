terraform {
  required_version = ">= 1.5"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 5.0"
    }
  }

  # Required: configure your remote state backend
  # backend "gcs" {
  #   bucket = "my-org-terraform-state"
  #   prefix = "carbon/gcp"
  # }
}

provider "google" {
  project = var.project_id

  # Uncomment to use a specific credentials file instead of application default credentials
  # credentials = file("path/to/credentials.json")
}
