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
  name = "keights-node-${var.cluster_name}-${var.node_group_name}"

  bootstrap_kubelet_conf              = "bootstrap-kubelet.conf"
  kubelet_bootstrap_kubeconfig_key_s3 = "keights/${var.cluster_name}/node/${local.bootstrap_kubelet_conf}"
  kubelet_config_key_s3               = "keights/${var.cluster_name}/node/kubelet-config.yaml"

  keights_inputs_sha = sha256(join("", [
    local.kubelet_bootstrap_kubeconfig,
    local.kubelet_config,
  ]))

  tags = merge(
    {
      "Name"                                      = local.name
      "kubernetes.io/cluster/${var.cluster_name}" = "owned"
      "keights.cloudboss.co/cluster"              = var.cluster_name
    },
    var.tags,
  )
}

resource "aws_s3_object" "kubelet_bootstrap_kubeconfig" {
  bucket  = var.s3_bucket
  key     = local.kubelet_bootstrap_kubeconfig_key_s3
  content = local.kubelet_bootstrap_kubeconfig
}

resource "aws_s3_object" "kubelet_configuration" {
  bucket  = var.s3_bucket
  key     = local.kubelet_config_key_s3
  content = local.kubelet_config
}

module "user_data" {
  source  = "cloudboss/easyto-user-data/aws"
  version = "0.4.0"

  command = ["/usr/bin/runsvdir", "/etc/service"]
  debug   = var.debug_logging
  env = [
    {
      # Since some configurations are stored in S3 and SSM and do not affect
      # the launch template directly, we include a hash of those configurations
      # to force a change to the launch configuration and trigger an update.
      name  = "KEIGHTS_INPUTS_SHA"
      value = local.keights_inputs_sha
    },
  ]
  env-from = [
    {
      imds = {
        name = "HOSTNAME"
        path = "hostname"
      }
    },
  ]
  init-scripts = [
    <<-EOS
      #!/bin/sh -e
      echo $${HOSTNAME} > /etc/hostname

      kernel_version=$(uname -r)
      mkdir -p /lib/modules/$${kernel_version}
      mount --bind /.easyto/lib/modules/$${kernel_version} /lib/modules/$${kernel_version}

      mount --bind /.easyto/run /run
      mkdir -p /run/runit/supervise
      update-service --add /etc/sv/containerd
      update-service --add /etc/sv/kubelet-node
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
      s3 = {
        bucket     = var.s3_bucket
        key-prefix = local.kubelet_bootstrap_kubeconfig_key_s3
        mount = {
          destination = "/etc/kubernetes/${local.bootstrap_kubelet_conf}"
        }
      }
    },
    {
      s3 = {
        bucket     = var.s3_bucket
        key-prefix = local.kubelet_config_key_s3
        mount = {
          destination = "/var/lib/kubelet/config.yaml"
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
  ]
}

module "asg" {
  source  = "cloudboss/asg/aws"
  version = "0.1.1"

  ami = var.ami
  block_device_mappings = [
    {
      device_name = var.storage.containerd.device
      ebs = {
        delete_on_termination = true
        encrypted             = true
        iops                  = var.storage.containerd.iops
        volume_size           = var.storage.containerd.size
        volume_type           = var.storage.containerd.type
      }
    }
  ]
  iam_instance_profile      = var.iam_instance_profile
  instance_type             = var.instance_type
  instance_refresh          = var.instance_refresh
  instances_desired         = var.instances_desired
  instances_max             = var.instances_max
  instances_min             = var.instances_min
  mixed_instances_overrides = var.mixed_instances_overrides
  name                      = local.name
  security_group_ids        = var.security_group_ids
  ssh_key                   = var.key_pair
  subnet_ids                = var.subnet_ids
  tags = {
    instance = local.tags
  }
  user_data = {
    value = module.user_data.value
  }
  vpc_id = var.vpc_id
}
