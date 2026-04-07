terraform {
  required_version = "=1.14.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "=6.38.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "=3.8.1"
    }
  }
}

provider "aws" {
  region = local.aws_region
}

module "keights" {
  source = "https://github.com/cloudboss/keights/releases/download/v2.0.0/keights-terraform-v2.0.0.tar.gz"

  access_cidrs  = local.access_cidrs
  cluster_name  = local.cluster_name
  control_plane = local.control_plane
  kms_key_id    = local.kms_key_id
  node_groups   = local.node_groups
  vpc_id        = local.vpc_id
}
