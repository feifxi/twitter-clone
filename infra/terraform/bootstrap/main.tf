# Created ONCE per AWS account and kept: everything that must outlive the app stack's
# up/down cycles. Costs nothing.
#   - a Route 53 reusable delegation set: four fixed name servers, so the NS records at your
#     DNS provider keep working while the app stack's hosted zone is destroyed and recreated
#   - the GitHub Actions deploy role (github.tf)
#
#   cd infra/terraform/bootstrap && terraform init && terraform apply
#
# State is local (git-ignored): small, secret-free, and only needed to change these resources.

terraform {
  required_version = ">= 1.10"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

variable "project" {
  description = "Prefix of every resource name; must match the app stack's"
  type        = string
  default     = "chmtwt"
}

variable "aws_region" {
  type    = string
  default = "ap-southeast-1"
}

provider "aws" {
  region = var.aws_region
  default_tags {
    tags = { Project = var.project, Stack = "bootstrap", ManagedBy = "terraform" }
  }
}

data "aws_caller_identity" "current" {}

resource "aws_route53_delegation_set" "main" {
  reference_name = var.project
}

output "delegation_set_id" {
  description = "Put in infra/terraform/app/terraform.tfvars as delegation_set_id"
  value       = aws_route53_delegation_set.main.id
}

output "name_servers" {
  description = "Add each as an NS record for your subdomain at your DNS provider"
  value       = aws_route53_delegation_set.main.name_servers
}
