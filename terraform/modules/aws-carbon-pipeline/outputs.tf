output "s3_bucket_arn" {
  description = "ARN of the S3 bucket storing carbon data exports."
  value       = aws_s3_bucket.carbon.arn
}

output "s3_bucket_id" {
  description = "Name of the S3 bucket storing carbon data exports."
  value       = aws_s3_bucket.carbon.id
}

output "glue_database_name" {
  description = "Name of the Glue catalog database."
  value       = aws_glue_catalog_database.carbon.name
}

output "glue_crawler_name" {
  description = "Name of the Glue crawler."
  value       = aws_glue_crawler.carbon.name
}

output "glue_service_role_arn" {
  description = "ARN of the IAM role used by the Glue crawler."
  value       = aws_iam_role.glue_service_role.arn
}

output "s3_bucket_public_access_block_id" {
  description = "ID of the public access block resource for the carbon S3 bucket."
  value       = aws_s3_bucket_public_access_block.carbon.id
}
