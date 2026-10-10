# One public subnet for the EC2 host and two private subnets for RDS (a DB subnet
# group needs two AZs). Private subnets have no internet route: nothing in them
# needs one, so there's no NAT to pay for.

resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
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
  cidr_block        = cidrsubnet(var.vpc_cidr, 8, 0)
  availability_zone = data.aws_availability_zones.available.names[0]
  tags              = { Name = "${local.name}-public" }
}

resource "aws_subnet" "private" {
  count             = 2
  vpc_id            = aws_vpc.main.id
  cidr_block        = cidrsubnet(var.vpc_cidr, 8, 10 + count.index)
  availability_zone = data.aws_availability_zones.available.names[count.index]
  tags              = { Name = "${local.name}-private-${count.index}" }
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

# EC2 Instance Connect Endpoint: SSH (and RDS tunnels) without public SSH or key pairs.
resource "aws_ec2_instance_connect_endpoint" "main" {
  subnet_id          = aws_subnet.public.id
  security_group_ids = [aws_security_group.eice.id]
  preserve_client_ip = false
  tags               = { Name = "${local.name}-eice" }
}

# ── Security groups ─────────────────────────────────

resource "aws_security_group" "api" {
  name        = "${local.name}-api"
  description = "API host: HTTPS from anywhere, SSH only via EICE"
  vpc_id      = aws_vpc.main.id
}

resource "aws_vpc_security_group_ingress_rule" "api_http" {
  for_each          = toset(["80", "443"])
  security_group_id = aws_security_group.api.id
  description       = "Caddy (HTTP redirects + ACME, HTTPS)"
  ip_protocol       = "tcp"
  from_port         = tonumber(each.value)
  to_port           = tonumber(each.value)
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "api_http3" {
  security_group_id = aws_security_group.api.id
  description       = "Caddy HTTP/3"
  ip_protocol       = "udp"
  from_port         = 443
  to_port           = 443
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "api_ssh_from_eice" {
  security_group_id            = aws_security_group.api.id
  description                  = "SSH via EC2 Instance Connect Endpoint"
  ip_protocol                  = "tcp"
  from_port                    = 22
  to_port                      = 22
  referenced_security_group_id = aws_security_group.eice.id
}

resource "aws_vpc_security_group_egress_rule" "api_all" {
  security_group_id = aws_security_group.api.id
  description       = "Outbound: ECR, SSM, Gemini, Google, Let's Encrypt, RDS"
  ip_protocol       = "-1"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_security_group" "eice" {
  name        = "${local.name}-eice"
  description = "EC2 Instance Connect Endpoint"
  vpc_id      = aws_vpc.main.id
}

resource "aws_vpc_security_group_egress_rule" "eice_ssh" {
  security_group_id            = aws_security_group.eice.id
  description                  = "SSH to the API host"
  ip_protocol                  = "tcp"
  from_port                    = 22
  to_port                      = 22
  referenced_security_group_id = aws_security_group.api.id
}

resource "aws_security_group" "rds" {
  name        = "${local.name}-rds"
  description = "Postgres from the API host only (admin tunnels go through it over SSH)"
  vpc_id      = aws_vpc.main.id
}

resource "aws_vpc_security_group_ingress_rule" "rds_from_api" {
  security_group_id            = aws_security_group.rds.id
  description                  = "Postgres from the API host"
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
  referenced_security_group_id = aws_security_group.api.id
}
