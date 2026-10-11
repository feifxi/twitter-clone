# Optional: manage the domain's record in Route 53. Without a hosted zone, create
# the record in the dns_records output at your DNS provider instead.

resource "aws_route53_record" "app" {
  count   = local.manage_dns ? 1 : 0
  zone_id = var.route53_zone_id
  name    = var.domain
  type    = "A"
  ttl     = 300
  records = [aws_eip.app.public_ip]
}
