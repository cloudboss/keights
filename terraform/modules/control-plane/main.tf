# Copyright © 2026 Joseph Wright <joseph@cloudboss.co>
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.

locals {
  azs_autoscaling_group = (
    var.storage.etcd == null
    ? {}
    : {
      for sn in var.subnet_ids :
      sn => data.aws_subnet.autoscaling_group[sn].availability_zone
    }
  )

  # CloudFormation is being used for the ASG as it works well for
  # coordinating control plane updates due to cfn-signal.
  cfn_template_body = {
    Resources = {
      AutoScalingGroup = {
        Type = "AWS::AutoScaling::AutoScalingGroup"
        CreationPolicy = {
          ResourceSignal = {
            Count   = local.cluster_size
            Timeout = "PT30M"
          }
        }
        Properties = {
          AutoScalingGroupName   = local.name
          DesiredCapacity        = local.cluster_size
          HealthCheckGracePeriod = 45
          HealthCheckType        = "ELB"
          LaunchTemplate = {
            LaunchTemplateId = aws_launch_template.it.id
            Version          = aws_launch_template.it.latest_version
          }
          MaxSize           = local.cluster_size
          MinSize           = local.cluster_size
          TargetGroupARNs   = [var.load_balancer.target_group_arn]
          VPCZoneIdentifier = var.subnet_ids
        }
        UpdatePolicy = {
          AutoScalingRollingUpdate = {
            MaxBatchSize          = 1
            MinInstancesInService = local.min_instances_in_service
            PauseTime             = "PT15M"
            WaitOnResourceSignals = true
            SuspendProcesses = [
              "HealthCheck",
              "ReplaceUnhealthy",
              "AZRebalance",
              "AlarmNotification",
              "ScheduledActions",
            ]
          }
        }
      }
    }
  }

  cluster_size = length(var.subnet_ids)

  min_instances_in_service = (
    local.cluster_size == 1 ? 0 : ceil(local.cluster_size / 2)
  )

  name = "keights-control-plane-${var.cluster_name}"

  tags_instances = merge(
    { "kubernetes.io/cluster/${var.cluster_name}" = "owned" },
    local.tags_launch_template,
  )
  tags_launch_template = merge(
    {
      "Name"                         = local.name
      "keights.cloudboss.co/cluster" = var.cluster_name
    },
    var.tags,
  )

  volume_tags = merge(var.tags, {
    "keights.cloudboss.co/cluster" = var.cluster_name
    "keights.cloudboss.co/etcd"    = ""
  })

  aws_cloud_controller_manager_values_default = {
    args = [
      "--allocate-node-cidrs=false",
      "--cloud-provider=aws",
      "--controllers=cloud-node-controller,cloud-node-lifecycle-controller,service-lb-controller",
      "--v=2",
    ]
    hostNetworking = true
  }

  aws_cloud_controller_manager_values = yamlencode(merge(
    local.aws_cloud_controller_manager_values_default,
    var.addons.aws_cloud_controller_manager.values
  ))

  identity_mappings_base = [
    {
      name = "kubernetes-admin"
      spec = {
        arn      = var.caller_identity.arn
        username = "kubernetes-admin"
        groups   = ["system:masters"]
      }
    },
    {
      name = "kubernetes-nodes"
      spec = {
        arn      = var.node_role_arn
        username = "system:node:{{EC2PrivateDNSName}}"
        groups   = ["system:bootstrappers", "system:nodes"]
      }
    },
  ]
  identity_mappings = concat(
    local.identity_mappings_base,
    var.identity_mappings,
  )
  aws_iam_authenticator_values_default = {
    awsPartition     = var.aws_partition
    clusterId        = var.cluster_name
    identityMappings = local.identity_mappings
  }
  aws_iam_authenticator_values = yamlencode(merge(
    local.aws_iam_authenticator_values_default,
    var.addons.aws_iam_authenticator.values
  ))

  aws_vpc_cni_values_default = {
    init = {
      image = {
        pullPolicy = "IfNotPresent"
        region     = var.aws_region
      }
    }
    nodeAgent = {
      image = {
        pullPolicy = "IfNotPresent"
        region     = var.aws_region
      }
    }
    image = {
      pullPolicy = "IfNotPresent"
      region     = var.aws_region
    }
  }

  aws_vpc_cni_values = yamlencode(merge(
    local.aws_vpc_cni_values_default,
    var.addons.aws_vpc_cni.values
  ))

  aws_cloud_controller_manager_yaml          = "aws-cloud-controller-manager.yaml"
  aws_iam_authenticator_yaml                 = "aws-iam-authenticator.yaml"
  aws_vpc_cni_yaml                           = "aws-vpc-cni.yaml"
  aws_cloud_controller_manager_values_key_s3 = "${var.s3_bucket_prefix}/${var.cluster_name}/controller/addons/${local.aws_cloud_controller_manager_yaml}"
  aws_iam_authenticator_values_key_s3        = "${var.s3_bucket_prefix}/${var.cluster_name}/controller/addons/${local.aws_iam_authenticator_yaml}"
  aws_vpc_cni_values_key_s3                  = "${var.s3_bucket_prefix}/${var.cluster_name}/controller/addons/${local.aws_vpc_cni_yaml}"
  kubeadm_init_config_key_s3                 = "${var.s3_bucket_prefix}/${var.cluster_name}/controller/kubeadm-init.yaml"
  kubeadm_init_config_path_host              = "/etc/kubernetes/kubeadm-init.yaml"

  keights_inputs_sha = sha256(join("", [
    local.aws_cloud_controller_manager_values,
    local.aws_iam_authenticator_values,
    local.aws_vpc_cni_values,
    local.kubeadm_init_config,
  ]))
}

data "aws_subnet" "autoscaling_group" {
  for_each = var.subnet_ids

  id = each.value
}

resource "aws_ebs_volume" "etcd" {
  for_each = var.subnet_ids

  availability_zone = local.azs_autoscaling_group[each.value]
  encrypted         = true
  size              = var.storage.etcd.size
  tags              = local.volume_tags
  type              = var.storage.etcd.type
}

resource "aws_s3_object" "aws_cloud_controller_manager_values" {
  bucket  = var.s3_bucket
  key     = local.aws_cloud_controller_manager_values_key_s3
  content = local.aws_cloud_controller_manager_values
}

resource "aws_s3_object" "aws_iam_authenticator_values" {
  bucket  = var.s3_bucket
  key     = local.aws_iam_authenticator_values_key_s3
  content = local.aws_iam_authenticator_values
}

resource "aws_s3_object" "aws_vpc_cni_values" {
  bucket  = var.s3_bucket
  key     = local.aws_vpc_cni_values_key_s3
  content = local.aws_vpc_cni_values
}

resource "aws_s3_object" "kubeadm_init_config" {
  bucket  = var.s3_bucket
  key     = local.kubeadm_init_config_key_s3
  content = local.kubeadm_init_config
}

resource "aws_launch_template" "it" {
  image_id                             = module.ami.id
  instance_initiated_shutdown_behavior = "stop"
  instance_type                        = var.instance_type
  key_name                             = var.key_pair
  name                                 = local.name
  tags                                 = local.tags_launch_template
  user_data                            = base64encode(module.user_data.value)
  vpc_security_group_ids               = var.security_group_ids

  metadata_options {
    http_endpoint      = "enabled"
    http_tokens        = "required"
    http_protocol_ipv6 = "enabled"
  }

  monitoring {
    enabled = var.monitoring_enabled
  }

  block_device_mappings {
    device_name = var.storage.containerd.device

    ebs {
      delete_on_termination = true
      encrypted             = true
      iops                  = var.storage.containerd.iops
      volume_size           = var.storage.containerd.size
      volume_type           = var.storage.containerd.type
    }
  }

  iam_instance_profile {
    arn = var.iam_instance_profile
  }

  tag_specifications {
    resource_type = "instance"
    tags          = local.tags_instances
  }
}

resource "aws_cloudformation_stack" "it" {
  name          = local.name
  template_body = yamlencode(local.cfn_template_body)
}

module "ami" {
  source = "../ami"

  ami             = var.ami
  caller_identity = var.caller_identity
}

module "user_data" {
  source  = "cloudboss/easyto-user-data/aws"
  version = "0.4.0"

  command = ["/usr/bin/runsvdir", "/etc/service"]
  debug   = var.debug_logging
  env = [
    {
      # Since some configurations are stored in S3 and SSM and do not affect the
      # launch template directly, include a hash of those configurations to force a
      # change to the launch template and trigger an update when they change.
      name  = "KEIGHTS_INPUTS_SHA"
      value = local.keights_inputs_sha
    },
  ]
  env-from = [
    {
      imds = {
        name = "AVAILABILITY_ZONE"
        path = "placement/availability-zone"
      }
    },
    {
      imds = {
        name = "HOSTNAME"
        path = "hostname"
      }
    },
    {
      imds = {
        name = "IPV4_ADDRESS"
        path = "local-ipv4"
      }
    },
  ]
  init-scripts = [
    <<-EOS
      #!/bin/sh -e
      echo $${HOSTNAME} > /etc/hostname

      sed -i "s|__AVAILABILITY_ZONE__|$${AVAILABILITY_ZONE}|" ${local.kubeadm_init_config_path_host}
      sed -i "s|__HOSTNAME__|$${HOSTNAME}|" ${local.kubeadm_init_config_path_host}
      sed -i "s|__IPV4_ADDRESS__|$${IPV4_ADDRESS}|" ${local.kubeadm_init_config_path_host}

      echo CFN_STACK_NAME=${local.name} > /etc/sv/cfn-signal-control-plane/environment
      echo IPV4_ADDRESS=$${IPV4_ADDRESS} >> /etc/sv/cfn-signal-control-plane/environment

      kernel_version=$(uname -r)
      mkdir -p /lib/modules/$${kernel_version}
      mount --bind /.easyto/lib/modules/$${kernel_version} /lib/modules/$${kernel_version}

      mount --bind /.easyto/run /run
      mkdir -p /run/runit/supervise
      update-service --add /etc/sv/aws-iam-authenticator-init
      update-service --add /etc/sv/cfn-signal-control-plane
      update-service --add /etc/sv/containerd
      update-service --add /etc/sv/keights-addons
      update-service --add /etc/sv/kubeadm-init
      update-service --add /etc/sv/kubelet-control-plane
    EOS
  ]
  modules = var.modules
  sysctls = var.sysctls
  volumes = [
    {
      ebs = {
        device = var.storage.containerd.device
        mount = {
          destination = "/var/lib/containerd"
          fs-type     = "ext4"
          mode        = "0700"
        }
      }
    },
    {
      ebs = {
        attachment = {
          tags = [
            {
              key   = "keights.cloudboss.co/cluster"
              value = var.cluster_name
            },
            {
              key   = "keights.cloudboss.co/etcd"
              value = ""
            },
          ]
        }
        device = var.storage.etcd.device
        mount = {
          destination = "/var/lib/etcd"
          fs-type     = "ext4"
          mode        = "0700"
        }
      }
    },
    {
      s3 = {
        bucket     = var.s3_bucket
        key-prefix = local.aws_cloud_controller_manager_values_key_s3
        mount = {
          destination = "/etc/kubernetes/charts/${local.aws_cloud_controller_manager_yaml}"
        }
      }
    },
    {
      s3 = {
        bucket     = var.s3_bucket
        key-prefix = local.aws_iam_authenticator_values_key_s3
        mount = {
          destination = "/etc/kubernetes/charts/${local.aws_iam_authenticator_yaml}"
        }
      }
    },
    {
      s3 = {
        bucket     = var.s3_bucket
        key-prefix = local.aws_vpc_cni_values_key_s3
        mount = {
          destination = "/etc/kubernetes/charts/${local.aws_vpc_cni_yaml}"
        }
      }
    },
    {
      s3 = {
        bucket     = var.s3_bucket
        key-prefix = local.kubeadm_init_config_key_s3
        mount = {
          destination = local.kubeadm_init_config_path_host
        }
      }
    },
    {
      ssm = {
        path = "/keights/${var.cluster_name}/cluster/ca.crt"
        mount = {
          destination = "/etc/kubernetes/pki/ca.crt"
        }
      }
    },
    {
      ssm = {
        path = "/keights/${var.cluster_name}/controller/ca.key"
        mount = {
          destination = "/etc/kubernetes/pki/ca.key"
        }
      }
    },
    {
      ssm = {
        path = "/keights/${var.cluster_name}/controller/etcd-ca.crt"
        mount = {
          destination = "/etc/kubernetes/pki/etcd/ca.crt"
        }
      }
    },
    {
      ssm = {
        path = "/keights/${var.cluster_name}/controller/etcd-ca.key"
        mount = {
          destination = "/etc/kubernetes/pki/etcd/ca.key"
        }
      }
    },
    {
      ssm = {
        path = "/keights/${var.cluster_name}/controller/front-proxy-ca.crt"
        mount = {
          destination = "/etc/kubernetes/pki/front-proxy-ca.crt"
        }
      }
    },
    {
      ssm = {
        path = "/keights/${var.cluster_name}/controller/front-proxy-ca.key"
        mount = {
          destination = "/etc/kubernetes/pki/front-proxy-ca.key"
        }
      }
    },
    {
      ssm = {
        path = "/keights/${var.cluster_name}/controller/sa.key"
        mount = {
          destination = "/etc/kubernetes/pki/sa.key"
        }
      }
    },
    {
      ssm = {
        path = "/keights/${var.cluster_name}/controller/sa.pub"
        mount = {
          destination = "/etc/kubernetes/pki/sa.pub"
        }
      }
    },
  ]
}
