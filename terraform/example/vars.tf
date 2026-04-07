locals {
  access_cidrs = {
    api = ["10.10.0.0/16"]
  }
  aws_region   = "us-east-1"
  cluster_name = "bonito"
  control_plane = {
    instance_type = "m5.large"
    internal      = true
    key_pair      = "easyto"
    subnet_ids = {
      autoscaling_group = [
        "subnet-d8b171acc1abf632a",
        "subnet-fbf354152798fa5ab",
        "subnet-cbdb15e59755031c4",
      ]
      load_balancer = [
        "subnet-d8b171acc1abf632a",
        "subnet-fbf354152798fa5ab",
        "subnet-cbdb15e59755031c4",
      ]
    }
  }
  kms_key_id = "alias/keights-bonito"
  node_groups = {
    default = {
      instance_type = "m5.large"
      mixed_instances_overrides = [
        {
          instance_requirements = {
            excluded_instance_types = [
              "t2.*", "m4.*", "c4.*", "r4.*", "i3.*", "d2.*", "m3.*", "c3.*", "r3.*",
            ]
            vcpu_count = {
              min = 4
              max = 8
            }
            memory_mib = {
              min = 8192
              max = 16384
            }
          }
        }
      ]
      subnet_ids = [
        "subnet-d8b171acc1abf632a",
        "subnet-fbf354152798fa5ab",
        "subnet-cbdb15e59755031c4",
      ]
    }
  }
  vpc_id = "vpc-3301d49634cccecce"
}
