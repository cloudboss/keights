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

variable "access_cidrs" {
  type = object({
    api        = optional(list(string), ["0.0.0.0/0"])
    node_ports = optional(list(string), [])
    ssh        = optional(list(string), [])
  })
  description = "CIDR blocks that are allowed access to Kubernetes."

  default = {}
}

variable "addons" {
  type = object({
    aws_cloud_controller_manager = optional(object({
      values = optional(map(any), {})
    }), {})
    aws_iam_authenticator = optional(object({
      values = optional(map(any), {})
    }), {})
    aws_ebs_csi_driver = optional(object({
      values = optional(map(any), {})
    }), {})
    aws_vpc_cni = optional(object({
      values = optional(map(any), {})
    }), {})
    cert_manager = optional(object({
      values = optional(map(any), {})
    }), {})
    pod_identity_webhook = optional(object({
      values = optional(map(any), {})
    }), {})
  })
  description = "Configuration for addon Helm charts."

  default = {}
}

variable "ami" {
  type = object({
    filters = optional(list(object({
      name   = string
      values = list(string)
    })), [])
    most_recent = optional(bool, true)
    name        = optional(string, "")
    owner       = optional(string, "")
  })
  description = "An object to configure the AMI to use. One of filters or name must be set."

  default = {}
}

variable "cluster_name" {
  type        = string
  description = "The name of the Kubernetes cluster."
}

variable "control_plane" {
  type = object({
    debug_logging      = optional(bool)
    instance_type      = string
    internal           = optional(bool, true)
    key_pair           = optional(string)
    monitoring_enabled = optional(bool, true)
    storage            = optional(any)
    subnet_ids = object({
      autoscaling_group = set(string)
      load_balancer     = set(string)
    })
  })
  description = "Configuration for the control plane."

  validation {
    condition     = length(var.control_plane.subnet_ids.autoscaling_group) % 2 != 0
    error_message = "There must be an odd number of subnet_ids in `var.control_plane.subnet_ids.autoscaling_group`."
  }
}

variable "encryption_algorithm" {
  type        = string
  description = "The encryption algorithm used for cluster encryption."

  default = "ECDSA-P256"

  validation {
    condition     = contains(["ECDSA-P256", "ECDSA-P384", "RSA-3072", "RSA-4096"], var.encryption_algorithm)
    error_message = "The value of `var.encryption_algorithm` must be one of `ECDSA-P256`, or `ECDSA-P384`, `RSA-3072`, or `RSA-4096`."
  }
}

variable "etcd_mode" {
  type        = string
  description = "The etcd cluster deployment mode. Must be one of `external` or `stacked`. The default is `stacked`, which configures etcd to run on the control plane nodes. In `external` mode, etcd runs on separate instances."

  default = "stacked"

  validation {
    condition     = contains(["external", "stacked"], var.etcd_mode)
    error_message = "The value of `var.etcd_mode` must be either `external` or `stacked`."
  }
}

variable "identity_mappings" {
  type = list(object({
    name = string
    spec = object({
      arn      = string
      username = string
      groups   = list(string)
    })
  }))
  description = "Configuration for aws-iam-authenticator identity mappings. By default the IAM identity of the cluster creator will have administrative access and nodes will have access to join the cluster."

  default = []
}

variable "irsa_enabled" {
  type        = bool
  description = "Enable IAM Roles for Service Accounts (IRSA). When enabled, an S3 bucket for OIDC discovery documents and an IAM OIDC provider will be created."

  default = true
}

variable "kms_key_id" {
  type        = string
  description = "The KMS Key ID used for cluster encryption. Can be an ID or alias. If not provided, a new key will be created with the alias `keights-cluster-$${var.cluster_name}`."

  default = null
}

variable "kubernetes_configuration" {
  type = object({
    cluster_domain = optional(string, "cluster.local")
    image_registry = optional(string, "registry.k8s.io")
    service_subnet = optional(string, "10.96.0.0/12")
    version        = optional(string)
  })
  description = "Kubernetes configuration options. If `version` is null, the Kubernetes version is parsed from the AMI name, provided it follows the keights convention `keights-vX.Y.Z-k8s-[v]A.B.C-<timestamp>`."

  default = {}
}

variable "lambda" {
  type = object({
    s3 = optional(object({
      bucket = optional(string, "cloudboss-keights")
      prefix = optional(string, "lambda")
    }), {})
    subnet_ids = optional(list(string))
    version    = optional(string, "0.1.0")
  })
  description = "Configuration for Lambda functions. If `subnet_ids` are not provided, they will be deployed without VPC configuration."

  default = {}
}

variable "node_groups" {
  type = map(object({
    ami = optional(any)
    autoscaling_group = optional(object({
      desired_capacity      = optional(number, 2)
      desired_capacity_type = optional(string, "units")
      instances_max         = optional(number, 5)
      instances_min         = optional(number, 0)
    }), {})
    debug_logging            = optional(bool)
    extra_security_group_ids = optional(list(string))
    iam_instance_profile     = optional(string)
    instance_refresh = optional(any, {
      strategy = "Rolling"
    })
    instance_type             = optional(string)
    key_pair                  = optional(string)
    mixed_instances_overrides = optional(list(any), [])
    monitoring_enabled        = optional(bool, true)
    storage                   = optional(any)
    subnet_ids                = optional(set(string))
  }))
  description = "Configuration for node groups. Each key is the name of a node group, and the value is an object with the configuration for that node group."

  default = {}
}

variable "route53" {
  type = object({
    hosted_zone_id   = optional(string)
    hosted_zone_name = optional(string)
  })
  description = "Configuration for Route53 private zone for etcd instances. If `hosted_zone_id` is not provided, a new private hosted zone will be created with the name `$${var.cluster_name}.local` or the value of `var.route53.hosted_zone_name` if provided."

  default = {}
}

variable "s3" {
  type = object({
    bucket_name   = optional(string)
    bucket_prefix = optional(string, "keights")
  })
  description = "Configuration for S3 bucket used for storing cluster assets. If `bucket_name` is not provided, a new bucket will be created."

  default = {}
}

variable "storage" {
  type = object({
    kms_key_id = optional(string)
    containerd = optional(object({
      device = optional(string, "/dev/sdf")
      iops   = optional(number)
      size   = optional(number, 10)
      type   = optional(string, "gp3")
    }), {})
    etcd = optional(object({
      device = optional(string, "/dev/sdg")
      iops   = optional(number)
      size   = optional(number, 10)
      type   = optional(string, "gp3")
    }), {})
  })
  description = "Configuration for storage. The containerd block configures the storage defaults for all cluster machines, but can be overridden individually on the control plane and node groups. `kms_key_id` is the KMS key used to encrypt EBS volumes (can be an ID, ARN, or alias). If null and `var.kms_key_id` is also null, the auto-created cluster key will be used; otherwise volumes are encrypted with the account's default EBS key."

  default = {}
}

variable "tags" {
  type        = map(string)
  description = "A map of tags to assign to resources."

  default = null
}

variable "vpc_id" {
  type        = string
  description = "The VPC ID where resources will be deployed."
}
