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

variable "addons" {
  type = any
}

variable "aws_partition" {
  type = string
}

variable "aws_region" {
  type = string
}

variable "caller_identity" {
  type = any
}

variable "cluster_domain" {
  type = string
}

variable "cluster_name" {
  type = string
}

variable "debug_logging" {
  type = bool
}

variable "encryption_algorithm" {
  type = string
}

variable "etcd_domain" {
  type = string
}

variable "iam_instance_profile" {
  type = string
}

variable "identity_mappings" {
  type = list(any)
}

variable "image_id" {
  type = string
}

variable "irsa" {
  type = any
}

variable "image_registry" {
  type = string
}

variable "instance_type" {
  type = string
}

variable "key_pair" {
  type = string
}

variable "kube_ca_function_name" {
  type = string
}

variable "kubernetes_version" {
  type = string
}

variable "load_balancer" {
  type = any
}

variable "modules" {
  type = list(string)
}

variable "monitoring_enabled" {
  type = bool
}

variable "node_role_arn" {
  type = string
}

variable "s3_bucket" {
  type = string
}

variable "s3_bucket_prefix" {
  type = string
}

variable "security_group_ids" {
  type = list(string)
}

variable "service_subnet" {
  type = string
}

variable "storage" {
  type = any
}

variable "subnet_ids" {
  type = set(string)
}

variable "sysctls" {
  type = list(map(string))
}

variable "tags" {
  type = map(string)
}

variable "vpc_id" {
  type = string
}
