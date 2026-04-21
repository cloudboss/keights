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

output "iam_instance_profile_control_plane" {
  value = aws_iam_instance_profile.control_plane
}

output "iam_policy_control_plane" {
  value = aws_iam_policy.control_plane
}

output "iam_role_control_plane" {
  value = aws_iam_role.control_plane
}

output "iam_instance_profile_etcd" {
  value = one(aws_iam_instance_profile.etcd)
}

output "iam_policy_etcd" {
  value = one(aws_iam_policy.etcd)
}

output "iam_role_etcd" {
  value = one(aws_iam_role.etcd)
}

output "iam_instance_profile_node" {
  value = aws_iam_instance_profile.node
}

output "iam_policy_node" {
  value = aws_iam_policy.node
}

output "iam_role_node" {
  value = aws_iam_role.node
}

output "iam_policy_ebs_csi" {
  value = aws_iam_policy.ebs_csi
}

output "iam_policy_lambda_logs" {
  value = aws_iam_policy.lambda_logs
}

output "iam_policy_lambda_vpc" {
  value = one(aws_iam_policy.lambda_vpc)
}

output "iam_policy_lambda_auto_namer" {
  value = aws_iam_policy.lambda_auto_namer
}

output "iam_role_lambda_auto_namer" {
  value = aws_iam_role.lambda_auto_namer
}

output "iam_policy_lambda_instance_attr" {
  value = aws_iam_policy.lambda_instance_attr
}

output "iam_role_lambda_instance_attr" {
  value = aws_iam_role.lambda_instance_attr
}

output "iam_policy_lambda_kube_ca" {
  value = aws_iam_policy.lambda_kube_ca
}

output "iam_role_lambda_kube_ca" {
  value = aws_iam_role.lambda_kube_ca
}

output "iam_role_policy_attachment_lambda_kube_ca" {
  value = aws_iam_role_policy_attachment.lambda_kube_ca
}

output "iam_role_policy_attachment_lambda_kube_ca_logs" {
  value = aws_iam_role_policy_attachment.lambda_kube_ca_logs
}
