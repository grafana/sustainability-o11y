variable "org_id" {
  description = "GCP organization ID. When set, creates org-level custom roles and IAM bindings required for the monthly data transfer. Leave null if roles are managed externally."
  type        = string
  default     = null
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
  default     = "us"
}

variable "service_account_id" {
  description = "Account ID for the service account that runs the Data Transfer."
  type        = string
  default     = "gcp-climate-data"
}

variable "grafana_bigquery_data_source" {
  description = "When true, creates a dedicated Grafana service account and grants it BigQuery dataViewer and jobUser roles."
  type        = bool
  default     = false
}

variable "grafana_service_account_email" {
  description = "Email of an existing Grafana service account to grant BigQuery dataViewer access. Leave null to skip."
  type        = string
  default     = null
}

variable "additional_dataset_access" {
  description = "Additional IAM bindings to add to the BigQuery dataset. Each object requires role and user_by_email."
  type = list(object({
    role          = string
    user_by_email = string
  }))
  default = []
}
