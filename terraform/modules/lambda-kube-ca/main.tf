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
  runtime = "provided.al2023"
}

resource "aws_lambda_function" "it" {
  function_name = "keights-kube-ca-${var.stack_key}"
  handler       = "bootstrap"
  role          = var.iam_role_arn
  runtime       = local.runtime
  s3_bucket     = var.s3.bucket
  s3_key        = var.s3.key
  timeout       = 30

  environment {
    variables = {
      CLUSTER_NAME         = var.stack_key
      ENCRYPTION_ALGORITHM = var.encryption_algorithm
      KMS_KEY_ID           = var.kms_key_id
    }
  }

  dynamic "vpc_config" {
    for_each = var.vpc_config == null ? [] : [1]

    content {
      subnet_ids         = var.vpc_config.subnet_ids
      security_group_ids = var.vpc_config.security_group_ids
    }
  }
}
