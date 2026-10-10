# Releases: CI uploads infra/ec2 for each commit to s3://<deploy bucket>/stack/<sha>/
# and pushes both images to ECR with the same tag; twitter-deploy on the instance
# fetches that commit's stack files and runs its deploy.sh.

resource "aws_s3_bucket" "deploy" {
  bucket        = "${local.name}-deploy-${data.aws_caller_identity.current.account_id}"
  force_destroy = true
}

resource "aws_s3_bucket_public_access_block" "deploy" {
  bucket                  = aws_s3_bucket.deploy.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# The stack files are a few KB per release; keep a year of them for rollbacks.
resource "aws_s3_bucket_lifecycle_configuration" "deploy" {
  bucket = aws_s3_bucket.deploy.id
  rule {
    id     = "expire-old-releases"
    status = "Enabled"
    filter {
      prefix = "stack/"
    }
    expiration {
      days = 365
    }
  }
}

# The last release that came up healthy, written by deploy.sh. A replacement instance
# deploys it at first boot; "none" means nothing has been deployed yet.
resource "aws_ssm_parameter" "release" {
  name  = local.release_param
  type  = "String"
  value = "none"

  lifecycle {
    ignore_changes = [value] # owned by deploys from here on
  }
}
