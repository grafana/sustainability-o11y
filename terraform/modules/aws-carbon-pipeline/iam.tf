data "aws_iam_policy_document" "glue_assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["glue.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

data "aws_iam_policy_document" "glue_s3_access" {
  statement {
    effect = "Allow"
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:ListBucket"
    ]
    resources = [
      aws_s3_bucket.carbon.arn,
      "${aws_s3_bucket.carbon.arn}/*"
    ]
  }
}

resource "aws_iam_role" "glue_service_role" {
  name               = var.glue_role_name
  assume_role_policy = data.aws_iam_policy_document.glue_assume_role.json

  tags = var.tags
}

resource "aws_iam_policy" "glue_s3_access" {
  name   = "${var.glue_role_name}-s3-policy"
  policy = data.aws_iam_policy_document.glue_s3_access.json

  tags = var.tags
}

# Attach the AWS-managed Glue service role (grants CloudWatch logs, Glue catalog access, etc.)
resource "aws_iam_role_policy_attachment" "glue_service_role" {
  role       = aws_iam_role.glue_service_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSGlueServiceRole"
}

# Attach the S3 access policy scoped to the carbon bucket
resource "aws_iam_role_policy_attachment" "glue_s3_access" {
  role       = aws_iam_role.glue_service_role.name
  policy_arn = aws_iam_policy.glue_s3_access.arn
}
