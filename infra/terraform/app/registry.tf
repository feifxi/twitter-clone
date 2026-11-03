# Images and releases. A release is one commit: both images tagged with its SHA, plus that
# commit's infra/compose and infra/observability files under s3://<deploy bucket>/stack/<sha>/.

locals {
  registry = split("/", aws_ecr_repository.app["api"].repository_url)[0]
}

resource "aws_ecr_repository" "app" {
  for_each = toset(["api", "web"])
  name     = "${local.name}-${each.key}"
  # A tag always means the same image, so a rollback is "deploy an older SHA"
  image_tag_mutability = "IMMUTABLE"
  force_delete         = true # down removes the images too
  image_scanning_configuration {
    scan_on_push = true
  }
}

resource "aws_ecr_lifecycle_policy" "app" {
  for_each   = aws_ecr_repository.app
  repository = each.value.name
  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep the last 10 images"
      selection    = { tagStatus = "any", countType = "imageCountMoreThan", countNumber = 10 }
      action       = { type = "expire" }
    }]
  })
}

resource "aws_s3_bucket" "deploy" {
  bucket_prefix = "${local.name}-deploy-"
  force_destroy = true
}

resource "aws_s3_bucket_public_access_block" "deploy" {
  bucket                  = aws_s3_bucket.deploy.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Where to deploy, for infra/scripts/deploy.sh on your machine and in CI (nothing secret).
# Its absence means the stack is down, so CI skips the deploy instead of failing.
resource "aws_ssm_parameter" "deploy_config" {
  name = "/${var.project}/deploy/config"
  type = "String"
  value = jsonencode({
    registry         = local.registry
    bucket           = aws_s3_bucket.deploy.id
    instance_id      = aws_instance.host.id
    domain           = var.domain
    google_client_id = var.google_client_id # compiled into the web image
  })
}
