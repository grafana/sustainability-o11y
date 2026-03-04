resource "aws_glue_catalog_database" "carbon" {
  name = var.glue_database_name
}

resource "aws_glue_crawler" "carbon" {
  name          = var.glue_crawler_name
  role          = aws_iam_role.glue_service_role.arn
  database_name = aws_glue_catalog_database.carbon.name
  schedule      = var.crawler_schedule

  s3_target {
    path = "s3://${aws_s3_bucket.carbon.id}/${var.s3_prefix}/${var.export_name}/data"
  }

  schema_change_policy {
    delete_behavior = "LOG"
    update_behavior = "LOG"
  }

  tags = var.tags
}
