# One repository per image; deploys tag both with the same commit SHA.
resource "aws_ecr_repository" "app" {
  for_each = toset(["api", "web"])
  name     = "${local.name}-${each.key}"
  # Tags are commit SHAs: a deployed tag always means the same image (and rollback = redeploy an old SHA).
  image_tag_mutability = "IMMUTABLE"
  force_delete         = true

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
      description  = "Keep the last ${var.ecr_images_to_keep} images"
      selection = {
        tagStatus   = "any"
        countType   = "imageCountMoreThan"
        countNumber = var.ecr_images_to_keep
      }
      action = { type = "expire" }
    }]
  })
}
