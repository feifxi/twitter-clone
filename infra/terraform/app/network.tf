# One public subnet for the host. Postgres runs on the host too, so there are no private
# subnets and no NAT gateway to pay for.

resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true
  tags                 = { Name = "${local.name}-vpc" }
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id
  tags   = { Name = "${local.name}-igw" }
}

resource "aws_subnet" "public" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.0.0/24"
  availability_zone = data.aws_availability_zones.available.names[0]
  tags              = { Name = "${local.name}-public" }
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }
  tags = { Name = "${local.name}-public" }
}

resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.public.id
  route_table_id = aws_route_table.public.id
}

# Only Caddy is reachable. No SSH: shell access and deploys go through SSM.
resource "aws_security_group" "host" {
  name        = "${local.name}-host"
  description = "HTTP, HTTPS and HTTP/3 to Caddy"
  vpc_id      = aws_vpc.main.id
}

resource "aws_vpc_security_group_ingress_rule" "host" {
  for_each = {
    http  = { protocol = "tcp", port = 80 } # redirect to HTTPS + ACME challenges
    https = { protocol = "tcp", port = 443 }
    http3 = { protocol = "udp", port = 443 }
  }
  security_group_id = aws_security_group.host.id
  description       = each.key
  ip_protocol       = each.value.protocol
  from_port         = each.value.port
  to_port           = each.value.port
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_egress_rule" "host" {
  security_group_id = aws_security_group.host.id
  description       = "ECR, SSM, S3, Gemini, Google, Let's Encrypt"
  ip_protocol       = "-1"
  cidr_ipv4         = "0.0.0.0/0"
}
