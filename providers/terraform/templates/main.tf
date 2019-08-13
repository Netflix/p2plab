terraform {
  required_version = ">= 0.12.6"

  backend "s3" {
    bucket = "{{.Bucket}}"
    key    = "{{.Key}}"
    region = "{{.Region}}"
  }
}

provider "aws" {
  alias  = "us-west-2"
  region = "us-west-2"
}

provider "aws" {
  alias  = "us-east-1"
  region = "us-east-1"
}

provider "aws" {
  alias  = "eu-west-1"
  region = "eu-west-1"
}

module "labagent_us-west-2" {
  source = "./modules/labagent"

  providers = {
    aws = aws.us-west-2
  }

  cluster_id                = var.cluster_id
  labagents                 = var.labagents["us-west-2"]
  labagent_instance_profile = var.labagent_instance_profile
  internal_subnets          = var.internal_subnets["us-west-2"]
}

module "labagent_us-east-1" {
  source = "./modules/labagent"

  providers = {
    aws = aws.us-east-1
  }

  cluster_id                = var.cluster_id
  labagents                 = var.labagents["us-east-1"]
  labagent_instance_profile = var.labagent_instance_profile
  internal_subnets          = var.internal_subnets["us-east-1"]
}

module "labagent_eu-west-1" {
  source = "./modules/labagent"

  providers = {
    aws = aws.eu-west-1
  }

  cluster_id                = var.cluster_id
  labagents                 = var.labagents["eu-west-1"]
  labagent_instance_profile = var.labagent_instance_profile
  internal_subnets          = var.internal_subnets["eu-west-1"]
}
