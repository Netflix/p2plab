variable "cluster_id" {
	type = string
}

variable "labagent_instance_profile" {
	type = string
}

variable "labagents" {
	type = map(object({
		size = number
		instance_type = string
	}))
}

variable "internal_subnets" {
	type = list(string)
}
