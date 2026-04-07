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
  runtime           = "provided.al2023"
  event_source      = "aws.autoscaling"
  event_detail_type = "EC2 Instance Launch Successful"
}

resource "aws_lambda_function" "it" {
  function_name = "keights-instance-attr-${var.stack_key}"
  handler       = "bootstrap"
  role          = var.iam_role_arn
  runtime       = local.runtime
  s3_bucket     = var.s3.bucket
  s3_key        = var.s3.key
  timeout       = 30

  dynamic "vpc_config" {
    for_each = var.vpc_config == null ? [] : [1]

    content {
      subnet_ids         = var.vpc_config.subnet_ids
      security_group_ids = var.vpc_config.security_group_ids
    }
  }
}

resource "aws_cloudwatch_event_rule" "them" {
  for_each = var.autoscaling_group_names

  name = "keights-instance-attr-${each.value}"

  event_pattern = jsonencode({
    detail-type = [local.event_detail_type]
    detail = {
      AutoScalingGroupName = [each.value]
    }
    source = [local.event_source]
  })
}

resource "aws_cloudwatch_event_target" "them" {
  for_each = var.autoscaling_group_names

  rule      = aws_cloudwatch_event_rule.them[each.value].name
  target_id = aws_cloudwatch_event_rule.them[each.value].name
  arn       = aws_lambda_function.it.arn
}

resource "aws_lambda_permission" "it" {
  for_each = var.autoscaling_group_names

  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.it.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.them[each.value].arn
}
