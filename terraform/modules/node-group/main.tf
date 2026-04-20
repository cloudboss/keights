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

  bootstrap_kubelet_conf = "bootstrap-kubelet.conf"

  tags = merge(
    {
      "Name"                                      = local.name
      "kubernetes.io/cluster/${var.cluster_name}" = "owned"
      "keights.cloudboss.co/cluster"              = var.cluster_name
    },
    var.tags,
  )
}

module "user_data" {
  source  = "cloudboss/easyto-user-data/aws"
  version = "0.5.0"

  command = ["/usr/bin/runsvdir", "/etc/service"]
  debug   = var.debug_logging
  env-from = [
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
      template = {
        content = <<-EOS
          {{hostname}}
        EOS
        mount = {
          destination = "/etc/hostname"
        }
        variables = {
          hostname = "$(HOSTNAME)"
        }
      }
    },
    {
      template = {
        content = local.kubelet_bootstrap_kubeconfig
        mount = {
          destination = "/etc/kubernetes/${local.bootstrap_kubelet_conf}"
        }
      }
    },
    {
      template = {
        content = <<-EOS
          KUBELET_KEIGHTS_ARGS="--node-ip={{ipv4_address}}"
        EOS
        mount = {
          destination = "/var/lib/kubelet/keights-flags.env"
        }
        variables = {
          ipv4_address = "$(IPV4_ADDRESS)"
        }
      }
    },
    {
      template = {
        content = local.kubelet_config
        mount = {
          destination = "/var/lib/kubelet/config.yaml"
        }
      }
    },
    {
      template = {
        content = data.aws_ssm_parameter.ca_crt.value
        mount = {
          destination = "/etc/kubernetes/pki/ca.crt"
        }
      }
    },
  ]
}

data "aws_ssm_parameter" "ca_crt" {
  name = "/keights/${var.cluster_name}/cluster/ca.crt"
}

module "asg" {
  source  = "cloudboss/asg/aws"
  version = "0.2.0"

  ami = var.ami
  block_device_mappings = [
    {
      device_name = var.storage.containerd.device
      ebs = {
        delete_on_termination = true
        encrypted             = true
        iops                  = var.storage.containerd.iops
        kms_key_id            = var.storage.kms_key_id
        volume_size           = var.storage.containerd.size
        volume_type           = var.storage.containerd.type
      }
    }
  ]
  desired_capacity          = var.desired_capacity
  desired_capacity_type     = var.desired_capacity_type
  iam_instance_profile      = var.iam_instance_profile
  instance_type             = var.instance_type
  instance_refresh          = var.instance_refresh
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
    value         = base64gzip(module.user_data.value)
    base64encoded = true
  }
  vpc_id = var.vpc_id
}
