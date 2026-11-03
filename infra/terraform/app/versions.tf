terraform {
  required_version = ">= 1.10"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }

  # Local state on purpose: one person runs it and `down` leaves nothing to track. It holds
  # the generated secrets, so it is git-ignored: keep it private, and keep it while the stack is up.
}

provider "aws" {
  region = var.aws_region
  default_tags {
    tags = { Project = var.project, Stack = "app", ManagedBy = "terraform" }
  }
}
