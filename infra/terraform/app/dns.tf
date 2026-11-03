# The zone uses bootstrap's delegation set, so its name servers never change and the NS records
# at your DNS provider keep working across down/up, even though the IP changes every time.
resource "aws_route53_zone" "main" {
  name              = var.domain
  delegation_set_id = var.delegation_set_id
  force_destroy     = true
}

resource "aws_route53_record" "host" {
  for_each = toset(["", "grafana."])
  zone_id  = aws_route53_zone.main.zone_id
  name     = "${each.key}${var.domain}"
  type     = "A"
  ttl      = 60
  records  = [aws_eip.host.public_ip]
}
