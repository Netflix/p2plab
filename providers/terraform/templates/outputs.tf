output "labagents" {
  value = {
    "us-west-2" = module.labagent_us-west-2.labagents
    "us-east-1" = module.labagent_us-east-1.labagents
    "eu-west-1" = module.labagent_eu-west-1.labagents
  }
}
