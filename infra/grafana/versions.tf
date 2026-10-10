terraform {
  required_version = ">= 1.10"

  required_providers {
    grafana = {
      source  = "grafana/grafana"
      version = "~> 4.40"
    }
  }

  # Same state bucket as ../terraform, its own key:
  #   terraform init -backend-config=../terraform/backend.hcl -backend-config="key=grafana.tfstate"
  backend "s3" {}
}

provider "grafana" {
  url  = var.grafana_url
  auth = var.grafana_auth
}
