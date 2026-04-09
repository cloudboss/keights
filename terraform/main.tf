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

module "s3" {
  source = "./modules/s3"
  count  = local.s3_bucket_count

  cluster_name = var.cluster_name
  tags         = var.tags
}

module "route53" {
  source = "./modules/route53"

  hosted_zone_id   = var.route53.hosted_zone_id
  hosted_zone_name = local.hosted_zone_name
  tags             = var.tags
  vpc_id           = var.vpc_id
}

module "irsa" {
  source = "./modules/irsa"
  count  = var.irsa_enabled ? 1 : 0

  audience     = "sts.amazonaws.com"
  cluster_name = var.cluster_name
  jwks         = jsondecode(aws_lambda_invocation.kube_ca.result).jwks
  tags         = var.tags
}

module "iam" {
  source = "./modules/iam"

  aws_account_id = local.aws_account_id
  aws_partition  = local.aws_partition
  aws_region     = local.aws_region
  cluster_name   = var.cluster_name
  features = {
    etcd_external = local.is_etcd_external
    lambda_vpc    = local.is_lambda_vpc
  }
  kms_key_id             = data.aws_kms_key.it.id
  route53_hosted_zone_id = module.route53.hosted_zone_id
  s3_bucket              = local.s3_bucket
  s3_bucket_prefix       = "keights"
  tags                   = var.tags
}

module "security_groups" {
  source = "./modules/security-groups"

  access_cidrs = var.access_cidrs
  cluster_name = var.cluster_name
  features = {
    etcd_external = local.is_etcd_external
    lambda_vpc    = local.is_lambda_vpc
  }
  vpc_id = var.vpc_id
  tags   = var.tags
}

module "lambda_auto_namer" {
  source = "./modules/lambda-auto-namer"

  cluster_name    = var.cluster_name
  hostname_prefix = "keights-etcd-${var.cluster_name}"
  iam_role_arn    = module.iam.iam_role_lambda_auto_namer.arn
  route53 = {
    hosted_zone_id   = module.route53.hosted_zone_id
    hosted_zone_name = module.route53.hosted_zone_name
  }
  s3 = {
    bucket = var.lambda.s3.bucket
    key    = local.lambda_s3_keys.auto_namer
  }
  vpc_config = local.vpc_config_lambda
}

module "lambda_instance_attr" {
  source = "./modules/lambda-instance-attr"

  autoscaling_group_names = [
    "keights-control-plane-${var.cluster_name}",
  ]
  iam_role_arn = module.iam.iam_role_lambda_instance_attr.arn
  s3 = {
    bucket = var.lambda.s3.bucket
    key    = local.lambda_s3_keys.instance_attr
  }
  stack_key  = var.cluster_name
  vpc_config = local.vpc_config_lambda
}

module "lambda_kube_ca" {
  source = "./modules/lambda-kube-ca"

  encryption_algorithm = var.encryption_algorithm
  iam_role_arn         = module.iam.iam_role_lambda_kube_ca.arn
  kms_key_id           = data.aws_kms_key.it.id
  s3 = {
    bucket = var.lambda.s3.bucket
    key    = local.lambda_s3_keys.kube_ca
  }
  stack_key  = var.cluster_name
  vpc_config = local.vpc_config_lambda
}

resource "aws_lambda_invocation" "kube_ca" {
  function_name = module.lambda_kube_ca.lambda.function_name
  input         = jsonencode({})
}

module "load_balancer" {
  source = "./modules/load-balancer"

  cluster_name       = var.cluster_name
  internal           = var.control_plane.internal
  security_group_ids = [module.security_groups.security_group_load_balancer.id]
  subnet_ids         = var.control_plane.subnet_ids.load_balancer
  tags               = var.tags
  vpc_id             = var.vpc_id
}

module "control_plane" {
  source = "./modules/control-plane"

  addons                = var.addons
  ami                   = var.ami
  irsa                  = local.irsa
  aws_partition         = local.aws_partition
  aws_region            = local.aws_region
  caller_identity       = data.aws_caller_identity.current
  cluster_domain        = var.kubernetes_configuration.cluster_domain
  cluster_name          = var.cluster_name
  debug_logging         = var.control_plane.debug_logging
  encryption_algorithm  = var.encryption_algorithm
  etcd_domain           = local.hosted_zone_name
  iam_instance_profile  = module.iam.iam_instance_profile_control_plane.arn
  identity_mappings     = var.identity_mappings
  image_registry        = var.kubernetes_configuration.image_registry
  instance_type         = var.control_plane.instance_type
  key_pair              = var.control_plane.key_pair
  kube_ca_function_name = module.lambda_kube_ca.lambda.function_name
  kubernetes_version    = var.kubernetes_configuration.version
  load_balancer = {
    dns_name         = module.load_balancer.it.load_balancer.dns_name
    target_group_arn = module.load_balancer.it.target_group.arn
  }
  modules            = local.modules
  monitoring_enabled = var.control_plane.monitoring_enabled
  node_role_arn      = module.iam.iam_role_node.arn
  s3_bucket          = local.s3_bucket
  s3_bucket_prefix   = var.s3.bucket_prefix
  security_group_ids = [module.security_groups.security_group_control_plane.id]
  service_subnet     = var.kubernetes_configuration.service_subnet
  storage = {
    containerd = {
      device = try(var.control_plane.storage.containerd.device, var.storage.containerd.device)
      iops   = try(var.control_plane.storage.containerd.iops, var.storage.containerd.iops)
      size   = try(var.control_plane.storage.containerd.size, var.storage.containerd.size)
      type   = try(var.control_plane.storage.containerd.type, var.storage.containerd.type)
    }
    etcd = {
      device = try(var.control_plane.storage.etcd.device, var.storage.etcd.device)
      iops   = try(var.control_plane.storage.etcd.iops, var.storage.etcd.iops)
      size   = try(var.control_plane.storage.etcd.size, var.storage.etcd.size)
      type   = try(var.control_plane.storage.etcd.type, var.storage.etcd.type)
    }
  }
  subnet_ids = var.control_plane.subnet_ids.autoscaling_group
  sysctls    = local.sysctls
  tags       = var.tags
  vpc_id     = var.vpc_id

  depends_on = [
    resource.aws_lambda_invocation.kube_ca,
    module.lambda_auto_namer,
    module.lambda_instance_attr,
    module.lambda_kube_ca,
  ]
}

module "node_groups" {
  source   = "./modules/node-group"
  for_each = local.node_groups

  ami                       = each.value.ami
  aws_region                = local.aws_region
  caller_identity           = data.aws_caller_identity.current
  cluster_domain            = var.kubernetes_configuration.cluster_domain
  cluster_name              = var.cluster_name
  control_plane_endpoint    = module.load_balancer.it.load_balancer.dns_name
  debug_logging             = each.value.debug_logging
  desired_capacity          = each.value.autoscaling_group.desired_capacity
  desired_capacity_type     = each.value.autoscaling_group.desired_capacity_type
  iam_instance_profile      = each.value.iam_instance_profile
  instance_refresh          = each.value.instance_refresh
  instance_type             = each.value.instance_type
  instances_max             = each.value.autoscaling_group.instances_max
  instances_min             = each.value.autoscaling_group.instances_min
  key_pair                  = each.value.key_pair
  mixed_instances_overrides = each.value.mixed_instances_overrides
  modules                   = local.modules
  monitoring_enabled        = each.value.monitoring_enabled
  node_group_name           = each.key
  s3_bucket                 = local.s3_bucket
  security_group_ids        = each.value.security_group_ids
  subnet_ids                = each.value.subnet_ids
  service_subnet            = var.kubernetes_configuration.service_subnet
  storage                   = each.value.storage
  sysctls                   = local.sysctls
  tags                      = var.tags
  vpc_id                    = var.vpc_id

  depends_on = [
    resource.aws_lambda_invocation.kube_ca,
  ]
}
