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
  aws_account_id = data.aws_caller_identity.current.account_id

  aws_partition = data.aws_partition.current.partition

  aws_region = data.aws_region.current.region

  irsa = var.irsa_enabled ? {
    oidc_issuer             = module.irsa[0].oidc_issuer
    ebs_csi_driver_role_arn = module.irsa_roles[0].iam_role_ebs_csi.arn
  } : null

  is_etcd_external = var.etcd_mode == "external"

  is_lambda_vpc = var.lambda.subnet_ids != null && length(var.lambda.subnet_ids) > 0

  kms_key_id = (
    var.kms_key_id != null
    ? data.aws_kms_key.it[0].id
    : module.kms_key[0].key_id
  )

  kms_key_id_storage = (
    var.storage.kms_key_id != null
    ? data.aws_kms_key.storage[0].arn
    : (var.kms_key_id == null ? module.kms_key[0].key_arn : null)
  )

  lambda_s3_keys = {
    auto_namer    = "${var.lambda.s3.prefix}/${local.lambda_version}/auto-namer-${local.lambda_version}.zip"
    instance_attr = "${var.lambda.s3.prefix}/${local.lambda_version}/instance-attr-${local.lambda_version}.zip"
    kube_ca       = "${var.lambda.s3.prefix}/${local.lambda_version}/kube-ca-${local.lambda_version}.zip"
  }

  lambda_version_prefix = (
    var.lambda.version != null && var.lambda.version != ""
    ? substr(var.lambda.version, 0, 1)
    : ""
  )

  lambda_version = (
    local.lambda_version_prefix == "v"
    ? var.lambda.version
    : "v${var.lambda.version}"
  )

  hosted_zone_name = (
    !(var.route53.hosted_zone_name == null || var.route53.hosted_zone_name == "")
    ? var.route53.hosted_zone_name
    : "${var.cluster_name}.local"
  )

  s3_bucket = (
    !(var.s3.bucket_name == null || var.s3.bucket_name == "")
    ? var.s3.bucket_name
    : one(module.s3[*].bucket.bucket)
  )

  s3_bucket_count = (
    !(var.s3.bucket_name == null || var.s3.bucket_name == "")
    ? 0
    : 1
  )

  vpc_config_lambda = (
    var.lambda.subnet_ids == null || length(var.lambda.subnet_ids) == 0
    ? null
    : {
      security_group_ids = [module.security_groups.security_group_lambda.id]
      subnet_ids         = var.lambda.subnet_ids
    }
  )

  node_groups = { for name, group in var.node_groups :
    name => {
      ami                       = var.ami
      autoscaling_group         = group.autoscaling_group
      debug_logging             = group.debug_logging
      iam_instance_profile      = coalesce(group.iam_instance_profile, module.iam.iam_instance_profile_node.arn)
      instance_refresh          = group.instance_refresh
      instance_type             = group.instance_type
      key_pair                  = group.key_pair
      mixed_instances_overrides = group.mixed_instances_overrides
      monitoring_enabled        = group.monitoring_enabled
      security_group_ids = toset(concat(
        [module.security_groups.security_group_node.id],
        coalesce(group.extra_security_group_ids, []),
      ))
      storage = {
        kms_key_id = try(data.aws_kms_key.storage_node_groups[name].arn, local.kms_key_id_storage)
        containerd = {
          device = try(group.storage.containerd.device, var.storage.containerd.device)
          iops   = try(group.storage.containerd.iops, var.storage.containerd.iops)
          size   = try(group.storage.containerd.size, var.storage.containerd.size)
          type   = try(group.storage.containerd.type, var.storage.containerd.type)
        }
      }
      subnet_ids = try(group.subnet_ids, var.control_plane.subnet_ids.autoscaling_group)
    }
  }

  modules = [
    "br_netfilter",
    "ip6t_REJECT",
    "ipt_REJECT",
    "nf_conntrack_netlink",
    "nf_nat",
    "nf_tables",
    "nft_chain_nat",
    "nft_compat",
    "overlay",
    "veth",
    "xt_MASQUERADE",
    "xt_REDIRECT",
    "xt_addrtype",
    "xt_comment",
    "xt_connmark",
    "xt_conntrack",
    "xt_mark",
    "xt_nat",
    "xt_nfacct",
    "xt_statistic",
  ]

  sysctls = [
    {
      name  = "net.bridge.bridge-nf-call-iptables"
      value = "1"
    },
    {
      name  = "net.bridge.bridge-nf-call-ip6tables"
      value = "1"
    },
    {
      name  = "net.ipv4.ip_forward"
      value = "1"
    },
  ]
}
