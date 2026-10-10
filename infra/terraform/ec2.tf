# The app host: Caddy (TLS) in front of the Next.js web app and the Go API, plus the
# monitoring agents, run with Docker Compose (see ../ec2). Shell access is Session
# Manager (aws ssm start-session), so no SSH port is open.

resource "aws_iam_role" "app" {
  name = "${local.name}-app"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

# SSM agent: Run Command (CI deploys) and Session Manager.
resource "aws_iam_role_policy_attachment" "app_ssm_core" {
  role       = aws_iam_role.app.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_role_policy" "app" {
  name = "${local.name}-app"
  role = aws_iam_role.app.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "ReadRuntimeConfig"
        Effect   = "Allow"
        Action   = ["ssm:GetParametersByPath"]
        Resource = "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter/${var.project_name}/${var.environment}/*"
      },
      {
        Sid      = "MediaObjects"
        Effect   = "Allow"
        Action   = ["s3:PutObject", "s3:GetObject", "s3:DeleteObject"]
        Resource = "${aws_s3_bucket.media.arn}/*"
      },
      {
        Sid      = "PullImages"
        Effect   = "Allow"
        Action   = ["ecr:BatchCheckLayerAvailability", "ecr:BatchGetImage", "ecr:GetDownloadUrlForLayer"]
        Resource = [for repo in aws_ecr_repository.app : repo.arn]
      },
      {
        Sid      = "EcrLogin"
        Effect   = "Allow"
        Action   = ["ecr:GetAuthorizationToken"]
        Resource = "*"
      },
    ]
  })
}

resource "aws_iam_instance_profile" "app" {
  name = "${local.name}-app"
  role = aws_iam_role.app.name
}

resource "aws_instance" "app" {
  ami                    = data.aws_ssm_parameter.al2023_ami.value
  instance_type          = var.ec2_instance_type
  subnet_id              = aws_subnet.public.id
  vpc_security_group_ids = [aws_security_group.app.id]
  iam_instance_profile   = aws_iam_instance_profile.app.name

  root_block_device {
    volume_size = 30
    volume_type = "gp3"
    encrypted   = true
  }

  metadata_options {
    http_tokens                 = "required" # IMDSv2 only
    http_put_response_hop_limit = 2          # containers sit one hop behind the host
  }

  # Changing the stack files rebuilds the instance. Caddy then requests a fresh
  # certificate; Let's Encrypt allows 5 per domain per week, so batch such changes.
  user_data_replace_on_change = true
  user_data = templatefile("${path.module}/../ec2/cloud-init.yaml.tftpl", {
    compose             = filebase64("${path.module}/../ec2/compose.yml")
    caddyfile           = filebase64("${path.module}/../ec2/Caddyfile")
    alloy               = filebase64("${path.module}/../ec2/config.alloy")
    deploy_script       = filebase64("${path.module}/../ec2/deploy.sh")
    compose_version     = var.docker_compose_version
    region              = var.aws_region
    api_ssm_path        = local.api_ssm_path
    monitoring_ssm_path = local.monitoring_ssm_path
    monitoring          = local.monitoring_enabled
    ecr_api_url         = aws_ecr_repository.app["api"].repository_url
    ecr_web_url         = aws_ecr_repository.app["web"].repository_url
    site_domain         = var.domain
    acme_email          = var.acme_email
  })

  # The first deploy reads these at boot.
  depends_on = [aws_ssm_parameter.api, aws_ssm_parameter.monitoring]

  tags = { Name = "${local.name}-app" }

  lifecycle {
    ignore_changes = [ami] # don't rebuild just because a newer AMI was published
  }
}

# Stable public IP for the domain's DNS record.
resource "aws_eip" "app" {
  instance = aws_instance.app.id
  domain   = "vpc"
  tags     = { Name = "${local.name}-app" }
}
