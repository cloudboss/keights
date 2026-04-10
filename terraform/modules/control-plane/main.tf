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

  aws_ebs_csi_driver_node_selector = {
    "node-role.kubernetes.io/control-plane" = ""
  }

  aws_ebs_csi_driver_pod_anti_affinity = {
    preferredDuringSchedulingIgnoredDuringExecution = [
      {
        weight = 100
        podAffinityTerm = {
          topologyKey = "kubernetes.io/hostname"
          labelSelector = {
            matchExpressions = [
              {
                key      = "app.kubernetes.io/name"
                operator = "In"
                values   = ["aws-ebs-csi-driver"]
              }
            ]
          }
        }
      }
    ]
  }

  aws_ebs_csi_driver_controller_base = {
    affinity = {
      podAntiAffinity = local.aws_ebs_csi_driver_pod_anti_affinity
    }
    nodeSelector = local.aws_ebs_csi_driver_node_selector
    replicaCount = local.cluster_size
    resources = {
      requests = {
        cpu    = "10m"
        memory = "40Mi"
      }
    }
    tolerations = [
      {
        key    = "node-role.kubernetes.io/control-plane"
        effect = "NoSchedule"
      }
    ]
    serviceAccount = {
      annotations = {}
    }
  }

  aws_ebs_csi_driver_controller = (
    var.irsa != null
    ? merge(local.aws_ebs_csi_driver_controller_base, {
      serviceAccount = {
        annotations = {
          "eks.amazonaws.com/role-arn" = var.irsa.ebs_csi_driver_role_arn
        }
      }
    })
    : local.aws_ebs_csi_driver_controller_base
  )

  aws_ebs_csi_driver_values = yamlencode(merge(
    {
      controller = local.aws_ebs_csi_driver_controller
      node       = { hostNetwork = true }
      helmTester = { enabled = false }
    },
    var.addons.aws_ebs_csi_driver.values
  ))

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

  cert_manager_node_selector = {
    "node-role.kubernetes.io/control-plane" = ""
  }

  cert_manager_pod_anti_affinity = {
    preferredDuringSchedulingIgnoredDuringExecution = [
      {
        weight = 100
        podAffinityTerm = {
          topologyKey = "kubernetes.io/hostname"
          labelSelector = {
            matchExpressions = [
              {
                key      = "app.kubernetes.io/instance"
                operator = "In"
                values   = ["cert-manager"]
              }
            ]
          }
        }
      }
    ]
  }

  cert_manager_pod_labels = {
    "pod-identity-webhook/exclude" = ""
  }

  cert_manager_values_default = {
    affinity = {
      podAntiAffinity = local.cert_manager_pod_anti_affinity
    }
    cainjector = {
      affinity = {
        podAntiAffinity = local.cert_manager_pod_anti_affinity
      }
      nodeSelector = local.cert_manager_node_selector
      podDisruptionBudget = {
        enabled = local.cluster_size > 1
      }
      podLabels    = local.cert_manager_pod_labels
      replicaCount = local.cluster_size
      tolerations = [
        {
          key    = "node-role.kubernetes.io/control-plane"
          effect = "NoSchedule"
        }
      ]
    }
    installCRDs  = true
    nodeSelector = local.cert_manager_node_selector
    podDisruptionBudget = {
      enabled = local.cluster_size > 1
    }
    podLabels    = local.cert_manager_pod_labels
    replicaCount = local.cluster_size
    startupapicheck = {
      enabled = false
    }
    tolerations = [
      {
        key    = "node-role.kubernetes.io/control-plane"
        effect = "NoSchedule"
      }
    ]
    webhook = {
      affinity = {
        podAntiAffinity = local.cert_manager_pod_anti_affinity
      }
      nodeSelector = local.cert_manager_node_selector
      podDisruptionBudget = {
        enabled = local.cluster_size > 1
      }
      podLabels    = local.cert_manager_pod_labels
      replicaCount = local.cluster_size
      tolerations = [
        {
          key    = "node-role.kubernetes.io/control-plane"
          effect = "NoSchedule"
        }
      ]
    }
  }

  cert_manager_values = yamlencode(merge(
    local.cert_manager_values_default,
    var.addons.cert_manager.values
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

  pod_identity_webhook_values_default = {
    args = [
      "--in-cluster=false",
      "--tls-cert=/etc/webhook/certs/tls.crt",
      "--tls-key=/etc/webhook/certs/tls.key",
      "--annotation-prefix=eks.amazonaws.com",
      "--token-audience=sts.amazonaws.com",
      "--aws-default-region=${var.aws_region}",
      "--sts-regional-endpoint=true",
      "--logtostderr",
    ]
    failurePolicy = "Fail"
    podLabels = {
      "pod-identity-webhook/exclude" = ""
    }
    mutatingWebhookConfiguration = {
      objectSelector = {
        matchExpressions = [
          {
            key      = "tier"
            operator = "NotIn"
            values   = ["control-plane"]
          },
          {
            key      = "k8s-app"
            operator = "NotIn"
            values = [
              "aws-cloud-controller-manager",
              "aws-node",
              "aws-iam-authenticator",
              "kube-dns",
              "kube-proxy",
            ]
          },
          {
            key      = "pod-identity-webhook/exclude"
            operator = "DoesNotExist"
          },
        ]
      }
    }
    replicas = local.cluster_size
    tolerations = [
      {
        key    = "node-role.kubernetes.io/control-plane"
        effect = "NoSchedule"
      }
    ]
  }

  pod_identity_webhook_values = yamlencode(merge(
    local.pod_identity_webhook_values_default,
    var.addons.pod_identity_webhook.values
  ))

  addon_s3_prefix = "${var.s3_bucket_prefix}/${var.cluster_name}/controller/addons"

  aws_cloud_controller_manager_yaml          = "aws-cloud-controller-manager.yaml"
  aws_ebs_csi_driver_yaml                    = "aws-ebs-csi-driver.yaml"
  aws_iam_authenticator_yaml                 = "aws-iam-authenticator.yaml"
  aws_vpc_cni_yaml                           = "aws-vpc-cni.yaml"
  cert_manager_yaml                          = "cert-manager.yaml"
  pod_identity_webhook_yaml                  = "pod-identity-webhook.yaml"
  aws_cloud_controller_manager_values_key_s3 = "${local.addon_s3_prefix}/${local.aws_cloud_controller_manager_yaml}"
  aws_ebs_csi_driver_values_key_s3           = "${local.addon_s3_prefix}/${local.aws_ebs_csi_driver_yaml}"
  aws_iam_authenticator_values_key_s3        = "${local.addon_s3_prefix}/${local.aws_iam_authenticator_yaml}"
  aws_vpc_cni_values_key_s3                  = "${local.addon_s3_prefix}/${local.aws_vpc_cni_yaml}"
  cert_manager_values_key_s3                 = "${local.addon_s3_prefix}/${local.cert_manager_yaml}"
  pod_identity_webhook_values_key_s3         = "${local.addon_s3_prefix}/${local.pod_identity_webhook_yaml}"
  kubeadm_init_config_key_s3                 = "${var.s3_bucket_prefix}/${var.cluster_name}/controller/kubeadm-init.yaml"
  kubeadm_init_config_path_host              = "/etc/kubernetes/kubeadm-init.yaml"

  keights_inputs_sha = sha256(join("", [
    local.aws_cloud_controller_manager_values,
    local.aws_ebs_csi_driver_values,
    local.aws_iam_authenticator_values,
    local.aws_vpc_cni_values,
    local.cert_manager_values,
    local.pod_identity_webhook_values,
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

resource "aws_s3_object" "aws_ebs_csi_driver_values" {
  bucket  = var.s3_bucket
  key     = local.aws_ebs_csi_driver_values_key_s3
  content = local.aws_ebs_csi_driver_values
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

resource "aws_s3_object" "cert_manager_values" {
  bucket  = var.s3_bucket
  key     = local.cert_manager_values_key_s3
  content = local.cert_manager_values
}

resource "aws_s3_object" "pod_identity_webhook_values" {
  count = var.irsa != null ? 1 : 0

  bucket  = var.s3_bucket
  key     = local.pod_identity_webhook_values_key_s3
  content = local.pod_identity_webhook_values
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
    http_endpoint               = "enabled"
    http_put_response_hop_limit = var.irsa != null ? 1 : 2
    http_tokens                 = "required"
    http_protocol_ipv6          = "enabled"
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
  volumes = local.volumes
}
