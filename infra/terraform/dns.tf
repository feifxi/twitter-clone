# Optional: manage api.<domain> in Route 53. Without a hosted zone, create the
# records listed in the dns_records output at your DNS provider instead.
# (For twitter.<domain>, Amplify adds its own records when the zone is in Route 53.)

resource "aws_route53_record" "api" {
  count   = local.manage_dns ? 1 : 0
  zone_id = var.route53_zone_id
  name    = var.api_domain
  type    = "A"
  ttl     = 300
  records = [aws_eip.api.public_ip]
}
