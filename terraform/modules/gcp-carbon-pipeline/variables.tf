variable "org_id" {
  description = "GCP organization ID. Used to create org-level custom roles and IAM bindings."
  type        = string
}

variable "project_id" {
  description = "GCP project ID where the BigQuery dataset and Data Transfer config will be created."
  type        = string
}

variable "billing_account_ids" {
  description = "One or more GCP billing account IDs to scope the carbon footprint export."
  type        = list(string)
}

variable "dataset_id" {
  description = "BigQuery dataset ID to create for the carbon footprint export."
  type        = string
  default     = "gcp_carbon_footprint"
}

variable "dataset_location" {
  description = "Location for the BigQuery dataset."
  type        = string
  default     = "US"
}

variable "service_account_id" {
  description = "Account ID for the service account that runs the Data Transfer."
  type        = string
  default     = "gcp-climate-data"
}
