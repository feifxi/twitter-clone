# Next.js (SSR) on Amplify Hosting, built from GitHub on every push to gh_branch.

resource "aws_iam_role" "amplify" {
  name = "${local.name}-amplify"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "amplify.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "amplify" {
  role       = aws_iam_role.amplify.name
  policy_arn = "arn:aws:iam::aws:policy/AdministratorAccess-Amplify"
}

resource "aws_amplify_app" "web" {
  name                 = local.name
  repository           = var.gh_repo_url
  access_token         = var.gh_token
  iam_service_role_arn = aws_iam_role.amplify.arn
  platform             = "WEB_COMPUTE"

  # NEXT_PUBLIC_* values are compiled into the browser bundle, so they are public by design.
  environment_variables = {
    AMPLIFY_MONOREPO_APP_ROOT    = "twitter-next-web"
    NEXT_TELEMETRY_DISABLED      = "1"
    NEXT_PUBLIC_API_URL          = local.api_base_url
    NEXT_PUBLIC_GOOGLE_CLIENT_ID = var.google_client_id
  }

  build_spec = <<-EOT
    version: 1
    applications:
      - appRoot: twitter-next-web
        frontend:
          phases:
            preBuild:
              commands:
                - npm ci
            build:
              commands:
                - npm run build
          artifacts:
            baseDirectory: .next
            files:
              - '**/*'
          cache:
            paths:
              - node_modules/**/*
  EOT
}

resource "aws_amplify_branch" "main" {
  app_id            = aws_amplify_app.web.id
  branch_name       = var.gh_branch
  framework         = "Next.js - SSR"
  stage             = "PRODUCTION"
  enable_auto_build = true
}

# twitter.<domain>. The web app and the API must share a registrable domain so
# the API's session cookie is first-party for the browser.
resource "aws_amplify_domain_association" "web" {
  app_id                = aws_amplify_app.web.id
  domain_name           = var.root_domain
  wait_for_verification = false

  sub_domain {
    branch_name = aws_amplify_branch.main.branch_name
    prefix      = trimsuffix(var.web_domain, ".${var.root_domain}")
  }
}
