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
  security_group_name_control_plane = "keights-control-plane-${var.cluster_name}"
  security_group_name_etcd          = "keights-etcd-${var.cluster_name}"
  security_group_name_lambda        = "keights-lambda-${var.cluster_name}"
  security_group_name_load_balancer = "keights-load-balancer-${var.cluster_name}"
  security_group_name_node          = "keights-node-${var.cluster_name}"

  security_group_tags_control_plane = merge(var.tags, {
    Name = local.security_group_name_control_plane
  })
  security_group_tags_etcd   = merge(var.tags, { Name = local.security_group_name_etcd })
  security_group_tags_lambda = merge(var.tags, { Name = local.security_group_name_lambda })
  security_group_tags_load_balancer = merge(var.tags, {
    Name = local.security_group_name_load_balancer
  })
  security_group_tags_node = merge(var.tags, { Name = local.security_group_name_node })

  security_group_mapping = {
    control_plane = aws_security_group.control_plane.id
    etcd          = one(aws_security_group.etcd[*].id)
    load_balancer = aws_security_group.load_balancer.id
    node          = aws_security_group.node.id
  }

  security_group_rules_ssh_cidrs = [for cidr in var.access_cidrs.ssh : {
    cidr_ipv4   = cidr
    from_port   = 22
    to_port     = 22
    ip_protocol = "tcp"
    type        = "ingress"
  }]

  security_group_rules_control_plane_base = [
    {
      cidr_ipv4   = "0.0.0.0/0"
      ip_protocol = "-1"
      type        = "egress"
    },
    {
      from_port                 = 10250 # Kubelet
      ip_protocol               = "tcp"
      referenced_security_group = "control_plane"
      to_port                   = 10250
      type                      = "ingress"
    },
    {
      from_port                 = 53
      ip_protocol               = "tcp"
      referenced_security_group = "control_plane"
      to_port                   = 53
      type                      = "ingress"
    },
    {
      from_port                 = 53
      ip_protocol               = "udp"
      referenced_security_group = "node"
      to_port                   = 53
      type                      = "ingress"
    },
    {
      from_port                 = 53
      ip_protocol               = "tcp"
      referenced_security_group = "node"
      to_port                   = 53
      type                      = "ingress"
    },
    {
      from_port                 = 53
      ip_protocol               = "udp"
      referenced_security_group = "control_plane"
      to_port                   = 53
      type                      = "ingress"
    },
    {
      from_port                 = 6443 # Kubernetes API
      ip_protocol               = "tcp"
      referenced_security_group = "control_plane"
      to_port                   = 6443
      type                      = "ingress"
    },
    {
      from_port                 = 6443
      ip_protocol               = "tcp"
      referenced_security_group = "load_balancer"
      to_port                   = 6443
      type                      = "ingress"
    },
    {
      from_port                 = 6443
      ip_protocol               = "tcp"
      referenced_security_group = "node"
      to_port                   = 6443
      type                      = "ingress"
    },
  ]
  security_group_rules_control_plane_etcd = (
    var.features.etcd_external
    ? []
    : [{
      from_port                 = 2379
      ip_protocol               = "tcp"
      referenced_security_group = "control_plane"
      to_port                   = 2380
      type                      = "ingress"
    }]
  )
  security_group_rules_control_plane = concat(
    local.security_group_rules_control_plane_base,
    local.security_group_rules_control_plane_etcd,
    local.security_group_rules_ssh_cidrs,
  )

  security_group_rules_etcd_base = [
    {
      from_port                 = 2379
      ip_protocol               = "tcp"
      referenced_security_group = "etcd"
      to_port                   = 2380
      type                      = "ingress"
    },
    {
      from_port                 = 2379
      ip_protocol               = "tcp"
      referenced_security_group = "control_plane"
      to_port                   = 2379
      type                      = "ingress"
    },
  ]
  security_group_rules_etcd = concat(
    local.security_group_rules_etcd_base,
    local.security_group_rules_ssh_cidrs,
  )

  security_group_rules_load_balancer_base = [
    {
      from_port                 = 443
      ip_protocol               = "tcp"
      referenced_security_group = "control_plane"
      to_port                   = 443
      type                      = "ingress"
    },
    {
      from_port                 = 6443
      ip_protocol               = "tcp"
      referenced_security_group = "control_plane"
      to_port                   = 6443
      type                      = "egress"
    },
  ]
  security_group_rules_load_balancer_api_cidrs = [
    for cidr in var.access_cidrs.api : {
      cidr_ipv4   = cidr
      from_port   = 443
      to_port     = 443
      ip_protocol = "tcp"
      type        = "ingress"
    }
  ]
  security_group_rules_load_balancer = concat(
    local.security_group_rules_load_balancer_base,
    local.security_group_rules_load_balancer_api_cidrs,
  )

  security_group_rules_node_base = [
    {
      cidr_ipv4   = "0.0.0.0/0"
      ip_protocol = "-1"
      type        = "egress"
    },
    {
      ip_protocol               = "-1"
      referenced_security_group = "control_plane"
      type                      = "ingress"
    },
    {
      ip_protocol               = "-1"
      referenced_security_group = "node"
      type                      = "ingress"
    },
  ]
  security_group_rules_node_node_ports_cidrs = [for cidr in var.access_cidrs.node_ports : {
    cidr_ipv4   = cidr
    from_port   = 30000
    to_port     = 32767
    ip_protocol = "tcp"
    type        = "ingress"
  }]
  security_group_rules_node = concat(
    local.security_group_rules_node_base,
    local.security_group_rules_node_node_ports_cidrs,
    local.security_group_rules_ssh_cidrs,
  )
}

resource "aws_security_group" "control_plane" {
  name   = local.security_group_name_control_plane
  tags   = local.security_group_tags_control_plane
  vpc_id = var.vpc_id
}

module "security_group_rules_control_plane" {
  source  = "cloudboss/security-group-rules/aws"
  version = "0.1.1"

  mapping           = local.security_group_mapping
  rules             = local.security_group_rules_control_plane
  security_group_id = aws_security_group.control_plane.id
}

resource "aws_security_group" "etcd" {
  count = var.features.etcd_external ? 1 : 0

  name   = local.security_group_name_etcd
  tags   = local.security_group_tags_etcd
  vpc_id = var.vpc_id
}

module "security_group_rules_etcd" {
  count   = var.features.etcd_external ? 1 : 0
  source  = "cloudboss/security-group-rules/aws"
  version = "0.1.1"

  mapping           = local.security_group_mapping
  rules             = local.security_group_rules_etcd
  security_group_id = aws_security_group.etcd[0].id
}

resource "aws_security_group" "lambda" {
  count = var.features.lambda_vpc ? 1 : 0

  name   = local.security_group_name_lambda
  tags   = local.security_group_tags_lambda
  vpc_id = var.vpc_id
}

module "security_group_rules_lambda" {
  count   = var.features.lambda_vpc ? 1 : 0
  source  = "cloudboss/security-group-rules/aws"
  version = "0.1.1"

  rules = [
    {
      cidr_ipv4   = "0.0.0.0/0"
      from_port   = 443
      ip_protocol = "tcp"
      to_port     = 443
      type        = "egress"
    },
  ]
  security_group_id = aws_security_group.lambda[0].id
}

resource "aws_security_group" "load_balancer" {
  name   = local.security_group_name_load_balancer
  tags   = local.security_group_tags_load_balancer
  vpc_id = var.vpc_id
}

module "security_group_rules_load_balancer" {
  source  = "cloudboss/security-group-rules/aws"
  version = "0.1.1"

  mapping           = local.security_group_mapping
  rules             = local.security_group_rules_load_balancer
  security_group_id = aws_security_group.load_balancer.id
}

resource "aws_security_group" "node" {
  name   = local.security_group_name_node
  tags   = local.security_group_tags_node
  vpc_id = var.vpc_id
}

module "security_group_rules_node" {
  source  = "cloudboss/security-group-rules/aws"
  version = "0.1.1"

  mapping           = local.security_group_mapping
  rules             = local.security_group_rules_node
  security_group_id = aws_security_group.node.id
}
