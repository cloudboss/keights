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
  asg_name          = "keights-control-plane-${var.cluster_name}"
  event_source      = "aws.autoscaling"
  event_detail_type = "EC2 Instance Launch Successful"
  runtime           = "provided.al2023"
}

resource "aws_lambda_function" "it" {
  function_name = "keights-auto-namer-${var.cluster_name}"
  handler       = "bootstrap"
  role          = var.iam_role_arn
  runtime       = local.runtime
  s3_bucket     = var.s3.bucket
  s3_key        = var.s3.key
  timeout       = 30

  environment {
    variables = {
      ASG_NAME         = local.asg_name
      DNS_TTL          = "15"
      HOST_BASE_NAME   = var.hostname_prefix
      HOSTED_ZONE_NAME = var.route53.hosted_zone_name
      HOSTED_ZONE_ID   = var.route53.hosted_zone_id
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

resource "aws_cloudwatch_event_rule" "it" {
  name = "keights-auto-namer-${var.cluster_name}"

  event_pattern = jsonencode({
    detail-type = [local.event_detail_type]
    detail = {
      AutoScalingGroupName = [local.asg_name]
    }
    source = [local.event_source]
  })
}

resource "aws_cloudwatch_event_target" "it" {
  rule      = aws_cloudwatch_event_rule.it.name
  target_id = aws_cloudwatch_event_rule.it.name
  arn       = aws_lambda_function.it.arn
}

resource "aws_lambda_permission" "it" {
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.it.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.it.arn
}
