output "api_url" {
  value = local.api_base_url
}

output "web_url" {
  value = local.web_origin
}

output "amplify_default_url" {
  value = local.amplify_url
}

output "api_public_ip" {
  description = "Elastic IP of the API host (api_domain's A record)"
  value       = aws_eip.api.public_ip
}

output "ec2_instance_id" {
  description = "GitHub Actions variable EC2_INSTANCE_ID; also used for SSH via EICE"
  value       = aws_instance.api.id
}

output "ecr_repository_url" {
  description = "GitHub Actions variable ECR_REPOSITORY"
  value       = aws_ecr_repository.api.repository_url
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
  description = "Records to create when Route 53 doesn't manage the domain"
  value = local.manage_dns ? "Managed in Route 53." : join("\n", compact(concat(
    ["${var.api_domain} A ${aws_eip.api.public_ip}"],
    [aws_amplify_domain_association.web.certificate_verification_dns_record],
    [for sd in aws_amplify_domain_association.web.sub_domain : sd.dns_record],
  )))
}
