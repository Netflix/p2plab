data "aws_ami" "labagent" {
  owners      = ["self"]
  most_recent = true

  filter {
    name   = "state"
    values = ["available"]
  }
  filter {
    name   = "tag:Name"
    values = ["labagent"]
  }
}

data "aws_security_group" "labagent" {
  filter {
    name   = "group-name"
    values = ["p2plab-labagent"]
  }
}

resource "aws_autoscaling_group" "labagent" {
  for_each = var.labagents

  name                = each.key
  max_size            = each.value.size
  min_size            = each.value.size
  desired_capacity    = each.value.size
  health_check_type   = "EC2"
  vpc_zone_identifier = var.internal_subnets

  launch_template {
    id = aws_launch_template.labagent[each.key].id
  }
}

resource "aws_launch_template" "labagent" {
  for_each = var.labagents

  image_id               = data.aws_ami.labagent.id
  name                   = each.key
  instance_type          = each.value.instance_type
  vpc_security_group_ids = [data.aws_security_group.labagent.id]

  iam_instance_profile {
    name = var.labagent_instance_profile
  }
}
