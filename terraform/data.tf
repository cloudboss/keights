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

data "aws_caller_identity" "current" {}

# Resolves an STS assumed-role ARN (e.g. when terraform is applied via OIDC)
# to the underlying IAM role ARN so it's usable as an aws-iam-authenticator
# identity mapping. For IAM users the ARN passes through unchanged.
data "aws_iam_session_context" "current" {
  arn = data.aws_caller_identity.current.arn
}

data "aws_partition" "current" {}

data "aws_region" "current" {}

# Use a lookup to get actual ID if alias is passed in `var.kms_key_id`.
data "aws_kms_key" "it" {
  count = var.kms_key_id != null ? 1 : 0

  key_id = var.kms_key_id
}

data "aws_kms_key" "storage" {
  count = var.storage.kms_key_id != null ? 1 : 0

  key_id = var.storage.kms_key_id
}

data "aws_kms_key" "storage_control_plane" {
  count = try(var.control_plane.storage.kms_key_id, null) != null ? 1 : 0

  key_id = var.control_plane.storage.kms_key_id
}

data "aws_kms_key" "storage_node_groups" {
  for_each = {
    for name, group in var.node_groups : name => group.storage.kms_key_id
    if try(group.storage.kms_key_id, null) != null
  }

  key_id = each.value
}
