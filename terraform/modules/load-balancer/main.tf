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
  name = "keights-${var.cluster_name}"

  port_apiserver = 6443
}

module "lb" {
  source  = "cloudboss/elbv2/aws"
  version = "0.1.2"

  internal = var.internal
  listener = {
    port     = 443
    protocol = "TCP"
    rules = {
      default = {
        type = "forward"
      }
    }
  }
  name               = local.name
  security_group_ids = var.security_group_ids
  subnet_mapping = [for subnet_id in var.subnet_ids : {
    subnet_id = subnet_id
  }]
  tags = {
    default = merge(var.tags, {
      "keights.cloudboss.co/cluster" = var.cluster_name
    })
  }
  target_group = {
    connection_termination = true
    deregistration_delay   = 15
    health_check = {
      interval            = 5
      healthy_threshold   = 2
      matcher             = "200"
      path                = "/readyz"
      port                = local.port_apiserver
      protocol            = "HTTPS"
      timeout             = 3
      unhealthy_threshold = 2
    }
    load_balancing_cross_zone_enabled = true
    port                              = local.port_apiserver
    protocol                          = "TCP"
    # Enable hairpinning for control plane instances.
    preserve_client_ip = false
  }
  type   = "network"
  vpc_id = var.vpc_id
}

output "it" {
  value = module.lb
}
