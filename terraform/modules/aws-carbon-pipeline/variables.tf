variable "account_id" {
  description = "AWS account ID used to scope bucket policy conditions."
  type        = string
}

variable "region" {
  description = "AWS region where the S3 bucket and exports are configured."
  type        = string
  default     = "us-east-2"
}

variable "bucket_name" {
  description = "Name of the S3 bucket to store carbon data exports."
  type        = string
  default     = "carbon-data-exports"
}

variable "export_name" {
  description = "Name of the BCM Data Export job."
  type        = string
  default     = "carbon-data-exports"
}

variable "s3_prefix" {
  description = "S3 key prefix for exported data."
  type        = string
  default     = "exports"
}

variable "glue_role_name" {
  description = "Name of the IAM role assumed by the Glue crawler."
  type        = string
  default     = "glue-carbon-crawler-service-role"
}

variable "glue_database_name" {
  description = "Name of the Glue catalog database."
  type        = string
  default     = "carbon"
}

variable "glue_crawler_name" {
  description = "Name of the Glue crawler."
  type        = string
  default     = "carbon-crawler"
}

variable "crawler_schedule" {
  description = "Cron schedule for the Glue crawler. Defaults to monthly, aligned with AWS emissions data refresh."
  type        = string
  default     = "cron(00 08 15-26 * ? *)"
}

variable "tags" {
  description = "Tags to apply to all taggable resources."
  type        = map(string)
  default     = {}
}
