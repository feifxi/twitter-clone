# Continuous deployment from GitHub Actions without stored AWS keys: the workflow trades its
# OIDC token for short-lived credentials of this role. It lives here, not in the app stack, so
# it survives up/down cycles. Its permissions name resources by prefix for the same reason:
# the app stack's resources don't exist while it is down. Only jobs in the repository's
# "production" environment can assume it.

variable "github_repo" {
  description = "owner/name of the repository allowed to deploy (its current name: tokens carry it)"
  type        = string
  default     = "feifxi/twitter-clone"
}

# GitHub can sign tokens with an immutable subject that adds the owner and repo IDs
# (repo:owner@ownerId/name@repoId:...), so a deleted-and-recreated repo with the same name can't
# deploy. Look them up with: curl -sL https://api.github.com/repos/<owner>/<name>
variable "github_repo_ids" {
  description = "The repository's owner ID and repo ID, as in its OIDC subject"
  type        = object({ owner = number, repo = number })
  default     = { owner = 137422903, repo = 1152995898 }
}

variable "github_environment" {
  description = "The GitHub environment the deploy job runs in"
  type        = string
  default     = "production"
}

variable "create_github_oidc_provider" {
  description = "false if this AWS account already has GitHub's OIDC provider (there can be only one)"
  type        = bool
  default     = true
}

resource "aws_iam_openid_connect_provider" "github" {
  count          = var.create_github_oidc_provider ? 1 : 0
  url            = "https://token.actions.githubusercontent.com"
  client_id_list = ["sts.amazonaws.com"]
}

data "aws_iam_openid_connect_provider" "github" {
  count = var.create_github_oidc_provider ? 0 : 1
  url   = "https://token.actions.githubusercontent.com"
}

locals {
  github_oidc_provider_arn = var.create_github_oidc_provider ? aws_iam_openid_connect_provider.github[0].arn : data.aws_iam_openid_connect_provider.github[0].arn
  region_account           = "${var.aws_region}:${data.aws_caller_identity.current.account_id}"
  repo                     = split("/", var.github_repo)
}

data "aws_iam_policy_document" "github_trust" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = [local.github_oidc_provider_arn]
    }
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }
    # Only jobs in the deploy environment of this repository, in either subject format
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:sub"
      values = [
        "repo:${var.github_repo}:environment:${var.github_environment}",
        "repo:${local.repo[0]}@${var.github_repo_ids.owner}/${local.repo[1]}@${var.github_repo_ids.repo}:environment:${var.github_environment}",
      ]
    }
  }
}

resource "aws_iam_role" "github_deploy" {
  name                 = "${var.project}-github-deploy"
  assume_role_policy   = data.aws_iam_policy_document.github_trust.json
  max_session_duration = 3600
}

# Exactly what infra/scripts/deploy.sh does: read the deploy config, push images, upload the
# release's stack files, and run the deploy on this project's instance.
data "aws_iam_policy_document" "github_deploy" {
  statement {
    sid       = "ReadDeployConfig"
    actions   = ["ssm:GetParameter"]
    resources = ["arn:aws:ssm:${local.region_account}:parameter/${var.project}/deploy/config"]
  }
  statement {
    sid       = "EcrLogin"
    actions   = ["ecr:GetAuthorizationToken"]
    resources = ["*"]
  }
  statement {
    sid = "PushImages"
    actions = [
      "ecr:BatchCheckLayerAvailability", "ecr:InitiateLayerUpload", "ecr:UploadLayerPart",
      "ecr:CompleteLayerUpload", "ecr:PutImage", "ecr:BatchGetImage", "ecr:DescribeImages",
    ]
    resources = ["arn:aws:ecr:${local.region_account}:repository/${var.project}-*"]
  }
  statement {
    sid       = "ListReleases"
    actions   = ["s3:ListBucket"]
    resources = ["arn:aws:s3:::${var.project}-deploy-*"]
  }
  statement {
    sid       = "UploadReleases"
    actions   = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
    resources = ["arn:aws:s3:::${var.project}-deploy-*/stack/*"]
  }
  statement {
    sid       = "RunDeployDocument"
    actions   = ["ssm:SendCommand"]
    resources = ["arn:aws:ssm:${var.aws_region}::document/AWS-RunShellScript"]
  }
  statement {
    sid       = "OnThisProjectsHostOnly"
    actions   = ["ssm:SendCommand"]
    resources = ["arn:aws:ec2:${local.region_account}:instance/*"]
    condition {
      test     = "StringEquals"
      variable = "ssm:resourceTag/Project"
      values   = [var.project]
    }
  }
  statement {
    sid       = "FollowDeploy"
    actions   = ["ssm:GetCommandInvocation", "ssm:ListCommandInvocations", "ssm:DescribeInstanceInformation"]
    resources = ["*"]
  }
}

resource "aws_iam_role_policy" "github_deploy" {
  name   = "deploy"
  role   = aws_iam_role.github_deploy.id
  policy = data.aws_iam_policy_document.github_deploy.json
}

output "github_deploy_role_arn" {
  description = "Set as the GitHub Actions repository variable AWS_DEPLOY_ROLE_ARN"
  value       = aws_iam_role.github_deploy.arn
}
