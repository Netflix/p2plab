variable "cluster_id" {
	type = string
}

variable "labagents" {
  type = map(map(object({
    size          = number
    instance_type = string
  })))
}

variable "labagent_instance_profile" {
  default = "labagentInstanceProfile"
}

variable "internal_subnets" {
  default = {
    "us-west-2" = [
      "subnet-090b3fe94bcc53cdb",
      "subnet-0148e2dfc8800fd39",
      "subnet-021adec36bd4bf0e2",
    ]
    "us-east-1" = [
      "subnet-0a9b5ca758b008e3c",
      "subnet-052e37f4809f49e7d",
      "subnet-0bb444d1979bc6ce4",
    ]
    "eu-west-1" = [
      "subnet-0ac98cc8ecb386361",
      "subnet-03339a637d558ac7f",
      "subnet-062c09248bdb6e857",
    ]
  }
}
