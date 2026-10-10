output "site_url" {
  value = local.site_origin
}

output "public_ip" {
  description = "Elastic IP of the app host (the domain's A record)"
  value       = aws_eip.app.public_ip
}

output "ec2_instance_id" {
  description = "GitHub Actions variable EC2_INSTANCE_ID; shell: aws ssm start-session --target <id>"
  value       = aws_instance.app.id
}

output "ecr_api_repository" {
  description = "GitHub Actions variable ECR_API_REPOSITORY"
  value       = aws_ecr_repository.app["api"].repository_url
}

output "ecr_web_repository" {
  description = "GitHub Actions variable ECR_WEB_REPOSITORY"
  value       = aws_ecr_repository.app["web"].repository_url
}

output "github_deploy_role_arn" {
  description = "GitHub Actions variable AWS_DEPLOY_ROLE_ARN"
  value       = aws_iam_role.github_deploy.arn
}

output "rds_endpoint" {
  value = aws_db_instance.main.endpoint
}

output "media_bucket" {
  value = aws_s3_bucket.media.id
}

output "cloudfront_domain" {
  value = aws_cloudfront_distribution.media.domain_name
}

output "dns_records" {
  description = "Record to create when Route 53 doesn't manage the domain"
  value       = local.manage_dns ? "Managed in Route 53." : "${var.domain} A ${aws_eip.app.public_ip}"
}
