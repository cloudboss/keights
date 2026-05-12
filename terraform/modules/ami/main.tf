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

variable "ami" {
  type = any
}

variable "caller_identity" {
  type = any
}

locals {
  ami_filters_computed = [
    {
      name   = "name"
      values = [var.ami.name]
    }
  ]
  ami_filters = (
    length(var.ami.filters) > 0
    ? var.ami.filters
    : local.ami_filters_computed
  )
  ami_owner_default = var.caller_identity.account_id
  ami_owner = (
    length(var.ami.owner) > 0 ? var.ami.owner : local.ami_owner_default
  )

  # Parse the Kubernetes version from a keights AMI name of the form
  # `keights-vX.Y.Z-k8s-vA.B.C-<timestamp>`. The "v" on the kubernetes
  # version is optional so legacy AMI names still return a value.
  kubernetes_version_matches = regexall(
    "^keights-v[0-9]+\\.[0-9]+\\.[0-9]+-k8s-v?([0-9]+\\.[0-9]+\\.[0-9]+)-",
    data.aws_ami.it.name,
  )
  kubernetes_version = one(local.kubernetes_version_matches[*][0])
}

data "aws_ami" "it" {
  most_recent = var.ami.most_recent
  owners      = [local.ami_owner]

  dynamic "filter" {
    for_each = local.ami_filters
    content {
      name   = filter.value.name
      values = filter.value.values
    }
  }
}

output "id" {
  value = data.aws_ami.it.id
}

output "kubernetes_version" {
  value = local.kubernetes_version
}
