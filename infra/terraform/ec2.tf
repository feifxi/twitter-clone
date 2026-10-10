# The API host: Caddy (TLS) → Go API, plus Redis and the monitoring agents,
# run with Docker Compose (see ../ec2).

resource "aws_iam_role" "api" {
  name = "${local.name}-api"
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
resource "aws_iam_role_policy_attachment" "api_ssm_core" {
  role       = aws_iam_role.api.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_role_policy" "api" {
  name = "${local.name}-api"
  role = aws_iam_role.api.id
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
        Sid      = "PullApiImage"
        Effect   = "Allow"
        Action   = ["ecr:BatchCheckLayerAvailability", "ecr:BatchGetImage", "ecr:GetDownloadUrlForLayer"]
        Resource = aws_ecr_repository.api.arn
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

resource "aws_iam_instance_profile" "api" {
  name = "${local.name}-api"
  role = aws_iam_role.api.name
}

resource "aws_instance" "api" {
  ami                    = data.aws_ssm_parameter.al2023_ami.value
  instance_type          = var.ec2_instance_type
  subnet_id              = aws_subnet.public.id
  vpc_security_group_ids = [aws_security_group.api.id]
  iam_instance_profile   = aws_iam_instance_profile.api.name

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
    ecr_repository_url  = aws_ecr_repository.api.repository_url
    api_domain          = var.api_domain
    acme_email          = var.acme_email
  })

  # The first deploy reads these at boot.
  depends_on = [aws_ssm_parameter.api, aws_ssm_parameter.monitoring]

  tags = { Name = "${local.name}-api" }

  lifecycle {
    ignore_changes = [ami] # don't rebuild just because a newer AMI was published
  }
}

# Stable public IP for the api.<domain> DNS record.
resource "aws_eip" "api" {
  instance = aws_instance.api.id
  domain   = "vpc"
  tags     = { Name = "${local.name}-api" }
}
