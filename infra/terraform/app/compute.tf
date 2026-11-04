# The host: Caddy, the web app, the Go API, Postgres and the monitoring stack, all in Docker
# Compose (infra/compose). Shell access is Session Manager, so no SSH port is open.

locals {
  # Docker Compose plugin, verified against this checksum at first boot
  compose_version = "v2.39.1"
  compose_sha256  = "7b2627ed76f7dcb0d93f649f185af912372229b4c09762a3cd1db5be5255632b" # docker-compose-linux-aarch64
}

data "aws_ssm_parameter" "al2023_arm64" {
  name = "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-arm64"
}

data "aws_iam_policy_document" "assume_ec2" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "host" {
  name               = "${local.name}-host"
  assume_role_policy = data.aws_iam_policy_document.assume_ec2.json
}

# SSM agent (Run Command for deploys, Session Manager for a shell) and pulling images
resource "aws_iam_role_policy_attachment" "host" {
  for_each = toset([
    "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
    "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
  ])
  role       = aws_iam_role.host.name
  policy_arn = each.value
}

data "aws_iam_policy_document" "host" {
  statement {
    sid       = "ReadRuntimeConfig"
    actions   = ["ssm:GetParametersByPath", "ssm:GetParameter"]
    resources = ["arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter/${var.project}/*"]
  }
  statement {
    sid       = "FetchReleases"
    actions   = ["s3:GetObject", "s3:ListBucket"]
    resources = [aws_s3_bucket.deploy.arn, "${aws_s3_bucket.deploy.arn}/stack/*"]
  }
  statement {
    sid       = "MediaObjects" # the API presigns uploads and deletes replaced media
    actions   = ["s3:PutObject", "s3:GetObject", "s3:DeleteObject"]
    resources = ["${aws_s3_bucket.media.arn}/*"]
  }
}

resource "aws_iam_role_policy" "host" {
  name   = "${local.name}-host"
  role   = aws_iam_role.host.id
  policy = data.aws_iam_policy_document.host.json
}

resource "aws_iam_instance_profile" "host" {
  name = "${local.name}-host"
  role = aws_iam_role.host.name
}

resource "aws_instance" "host" {
  ami                    = nonsensitive(data.aws_ssm_parameter.al2023_arm64.value)
  instance_type          = var.instance_type
  subnet_id              = aws_subnet.public.id
  vpc_security_group_ids = [aws_security_group.host.id]
  iam_instance_profile   = aws_iam_instance_profile.host.name

  # First boot only: Docker, Compose and the deploy launcher. The stack itself ships with
  # every deploy, so changing it never replaces the instance.
  user_data_replace_on_change = true
  user_data = templatefile("${path.module}/cloud-init.yaml.tftpl", {
    twitter_deploy  = filebase64("${path.module}/twitter-deploy")
    compose_version = local.compose_version
    compose_sha256  = local.compose_sha256
    region          = var.aws_region
    deploy_bucket   = aws_s3_bucket.deploy.id
    registry        = local.registry
    domain          = var.domain
    api_ssm_path    = local.api_ssm_path
    stack_ssm_path  = local.stack_ssm_path
    project         = var.project
  })

  root_block_device {
    volume_type = "gp3"
    volume_size = 30 # images, Postgres, 15 days of metrics and logs
    encrypted   = true
  }

  metadata_options {
    http_tokens                 = "required" # IMDSv2 only
    http_put_response_hop_limit = 2          # containers sit one hop behind the host
  }

  tags = { Name = "${local.name}-host" }

  # First boot downloads packages and registers with SSM, so the outbound rule must exist first
  depends_on = [aws_vpc_security_group_egress_rule.host]

  lifecycle {
    ignore_changes = [ami] # a newer AMI shouldn't replace a running host; the next up gets it
  }
}

# Stable public IP while the stack is up (the DNS records point at it)
resource "aws_eip" "host" {
  domain   = "vpc"
  instance = aws_instance.host.id
  tags     = { Name = "${local.name}-host" }
}
