resource "aws_s3_bucket" "carbon" {
  bucket        = var.bucket_name
  force_destroy = false

  tags = var.tags
}

data "aws_iam_policy_document" "carbon_bucket_policy" {
  statement {
    sid    = "EnableAWSDataExportsToWriteToS3AndCheckPolicy"
    effect = "Allow"

    principals {
      type = "Service"
      identifiers = [
        "billingreports.amazonaws.com",
        "bcm-data-exports.amazonaws.com"
      ]
    }

    actions = [
      "s3:PutObject",
      "s3:GetBucketPolicy"
    ]

    resources = [
      "${aws_s3_bucket.carbon.arn}/*",
      aws_s3_bucket.carbon.arn
    ]

    condition {
      test     = "StringLike"
      variable = "aws:SourceAccount"
      values   = [var.account_id]
    }

    condition {
      test     = "StringLike"
      variable = "aws:SourceArn"
      values = [
        # CUR and BCM Data Exports are only available in us-east-1 — this is not configurable
        "arn:aws:cur:us-east-1:${var.account_id}:definition/*",
        "arn:aws:bcm-data-exports:us-east-1:${var.account_id}:export/*"
      ]
    }
  }
}

resource "aws_s3_bucket_policy" "carbon" {
  bucket = aws_s3_bucket.carbon.id
  policy = data.aws_iam_policy_document.carbon_bucket_policy.json
}
