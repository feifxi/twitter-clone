output "urls" {
  value = {
    site    = local.site_origin
    grafana = "https://grafana.${var.domain}"
  }
}

output "instance_id" {
  description = "Shell on the host: aws ssm start-session --target <id>"
  value       = aws_instance.host.id
}

output "public_ip" {
  value = aws_eip.host.public_ip
}

output "name_servers" {
  description = "Must match the NS records at your DNS provider (they come from bootstrap's delegation set)"
  value       = aws_route53_zone.main.name_servers
}

output "grafana_password" {
  description = "Grafana login: admin / this (terraform output -raw grafana_password)"
  value       = random_password.grafana.result
  sensitive   = true
}
