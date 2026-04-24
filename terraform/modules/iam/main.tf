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
  assume_role_policy_ec2 = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      },
    ]
  })

  assume_role_policy_lambda = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      },
    ]
  })
}

resource "aws_iam_policy" "etcd" {
  count = var.features.etcd_external ? 1 : 0

  name = "keights-etcd-${var.cluster_name}"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ec2:AttachVolume",
          "ec2:DescribeVolumes",
        ]
        Resource = ["*"]
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParameter",
          "ssm:GetParametersByPath",
        ]
        Resource = [
          "arn:${var.aws_partition}:ssm:${var.aws_region}:${var.aws_account_id}:parameter/keights/${var.cluster_name}/controller/etcd-ca.crt",
          "arn:${var.aws_partition}:ssm:${var.aws_region}:${var.aws_account_id}:parameter/keights/${var.cluster_name}/controller/etcd-ca.key",
        ]
      },
      {
        Effect = "Allow"
        Action = ["kms:Decrypt"]
        Resource = [
          "arn:${var.aws_partition}:kms:${var.aws_region}:${var.aws_account_id}:key/${var.kms_key_id}",
        ]
      },
    ]
  })
  tags = var.tags
}

resource "aws_iam_role" "etcd" {
  count = var.features.etcd_external ? 1 : 0

  assume_role_policy = local.assume_role_policy_ec2
  name               = "keights-etcd-${var.cluster_name}"
  tags               = var.tags
}

resource "aws_iam_role_policy_attachment" "etcd" {
  count = var.features.etcd_external ? 1 : 0

  role       = aws_iam_role.etcd[0].name
  policy_arn = aws_iam_policy.etcd[0].arn
}

resource "aws_iam_instance_profile" "etcd" {
  count = var.features.etcd_external ? 1 : 0

  name = "keights-etcd-${var.cluster_name}"
  role = aws_iam_role.etcd[0].name
  tags = var.tags
}

resource "aws_iam_policy" "control_plane" {
  name = "keights-control-plane-${var.cluster_name}"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ec2:AttachVolume",
          "ec2:AuthorizeSecurityGroupIngress",
          "ec2:CreateRoute",
          "ec2:CreateSecurityGroup",
          "ec2:DeleteSecurityGroup",
          "ec2:CreateTags",
          "ec2:CreateVolume",
          "ec2:DeleteRoute",
          "ec2:DeleteVolume",
          "ec2:DescribeAccountAttributes",
          "ec2:DescribeAvailabilityZones",
          "ec2:DescribeInstances",
          "ec2:DescribeInstanceTopology",
          "ec2:DescribeRegions",
          "ec2:DescribeRouteTables",
          "ec2:DescribeSecurityGroups",
          "ec2:DescribeSubnets",
          "ec2:DescribeVolumes",
          "ec2:DescribeVpcs",
          "ec2:DetachVolume",
          "ec2:ModifyInstanceAttribute",
          "ec2:ModifyVolume",
          "ec2:RevokeSecurityGroupIngress",
        ]
        Resource = ["*"]
      },
      {
        Effect = "Allow"
        Action = [
          "elasticloadbalancing:AddTags",
          "elasticloadbalancing:AttachLoadBalancerToSubnets",
          "elasticloadbalancing:ApplySecurityGroupsToLoadBalancer",
          "elasticloadbalancing:CreateListener",
          "elasticloadbalancing:CreateLoadBalancer",
          "elasticloadbalancing:CreateLoadBalancerPolicy",
          "elasticloadbalancing:CreateLoadBalancerListeners",
          "elasticloadbalancing:CreateTargetGroup",
          "elasticloadbalancing:ConfigureHealthCheck",
          "elasticloadbalancing:DeleteListener",
          "elasticloadbalancing:DeleteLoadBalancer",
          "elasticloadbalancing:DeleteLoadBalancerListeners",
          "elasticloadbalancing:DeregisterInstancesFromLoadBalancer",
          "elasticloadbalancing:DescribeListeners",
          "elasticloadbalancing:DescribeLoadBalancerAttributes",
          "elasticloadbalancing:DescribeLoadBalancerPolicies",
          "elasticloadbalancing:DescribeLoadBalancers",
          "elasticloadbalancing:DescribeTargetGroups",
          "elasticloadbalancing:DescribeTargetHealth",
          "elasticloadbalancing:DetachLoadBalancerFromSubnets",
          "elasticloadbalancing:ModifyListener",
          "elasticloadbalancing:ModifyLoadBalancerAttributes",
          "elasticloadbalancing:ModifyTargetGroup",
          "elasticloadbalancing:RegisterInstancesWithLoadBalancer",
          "elasticloadbalancing:RegisterTargets",
          "elasticloadbalancing:SetLoadBalancerPoliciesForBackendServer",
        ]
        Resource = ["*"]
      },
      {
        Effect = "Allow"
        Action = [
          "autoscaling:DescribeAutoScalingGroups",
          "autoscaling:DescribeAutoScalingInstances",
          "autoscaling:DescribeLaunchConfigurations",
          "autoscaling:DescribeTags",
          "autoscaling:GetAsgForInstance",
          "autoscaling:SetDesiredCapacity",
          "autoscaling:TerminateInstanceInAutoScalingGroup",
          "autoscaling:UpdateAutoScalingGroup",
        ]
        Resource = ["*"]
      },
      {
        Effect = "Allow"
        Action = [
          "iam:CreateServiceLinkedRole",
          "iam:ListServerCertificates",
          "iam:GetServerCertificate",
        ]
        Resource = ["*"]
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParameter",
          "ssm:GetParametersByPath",
        ]
        Resource = [
          "arn:${var.aws_partition}:ssm:${var.aws_region}:${var.aws_account_id}:parameter/keights/${var.cluster_name}/cluster/*",
          "arn:${var.aws_partition}:ssm:${var.aws_region}:${var.aws_account_id}:parameter/keights/${var.cluster_name}/controller/*",
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
        ]
        Resource = [
          "arn:${var.aws_partition}:kms:${var.aws_region}:${var.aws_account_id}:key/${var.kms_key_id}",
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "s3:ListBucket",
        ]
        Resource = [
          "arn:${var.aws_partition}:s3:::${var.s3_bucket}",
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:ListObjectsV2",
        ]
        Resource = [
          "arn:${var.aws_partition}:s3:::${var.s3_bucket}/${var.s3_bucket_prefix}/${var.cluster_name}/controller/*",
        ]
      },
    ]
  })
  tags = var.tags
}

resource "aws_iam_role" "control_plane" {
  assume_role_policy = local.assume_role_policy_ec2
  name               = "keights-control-plane-${var.cluster_name}"
  tags               = var.tags
}

resource "aws_iam_role_policy_attachment" "control_plane" {
  role       = aws_iam_role.control_plane.name
  policy_arn = aws_iam_policy.control_plane.arn
}

resource "aws_iam_role_policy_attachment" "control_plane_cni" {
  role       = aws_iam_role.control_plane.name
  policy_arn = "arn:${var.aws_partition}:iam::aws:policy/AmazonEKS_CNI_Policy"
}

resource "aws_iam_role_policy_attachment" "control_plane_ecr" {
  role       = aws_iam_role.control_plane.name
  policy_arn = "arn:${var.aws_partition}:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

resource "aws_iam_instance_profile" "control_plane" {
  name = aws_iam_role.control_plane.name
  role = aws_iam_role.control_plane.name
  tags = var.tags
}

resource "aws_iam_policy" "ebs_csi" {
  name = "keights-ebs-csi-${var.cluster_name}"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ec2:AttachVolume",
          "ec2:CreateSnapshot",
          "ec2:CreateTags",
          "ec2:CreateVolume",
          "ec2:DeleteSnapshot",
          "ec2:DeleteVolume",
          "ec2:DescribeAvailabilityZones",
          "ec2:DescribeInstances",
          "ec2:DescribeSnapshots",
          "ec2:DescribeTags",
          "ec2:DescribeVolumes",
          "ec2:DetachVolume",
          "ec2:ModifyVolume",
        ]
        Resource = ["*"]
      },
    ]
  })
  tags = var.tags
}

resource "aws_iam_role_policy_attachment" "control_plane_ebs_csi" {
  count = var.features.irsa ? 0 : 1

  role       = aws_iam_role.control_plane.name
  policy_arn = aws_iam_policy.ebs_csi.arn
}

resource "aws_iam_policy" "node" {
  name = "keights-node-${var.cluster_name}"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ec2:DescribeInstances",
          "ec2:DescribeRegions",
        ]
        Resource = ["*"]
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParameter",
          "ssm:GetParametersByPath",
        ]
        Resource = [
          "arn:${var.aws_partition}:ssm:${var.aws_region}:${var.aws_account_id}:parameter/keights/${var.cluster_name}/cluster/*",
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
        ]
        Resource = [
          "arn:${var.aws_partition}:kms:${var.aws_region}:${var.aws_account_id}:key/${var.kms_key_id}",
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "s3:ListBucket",
        ]
        Resource = [
          "arn:${var.aws_partition}:s3:::${var.s3_bucket}",
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:ListObjectsV2",
        ]
        Resource = [
          "arn:${var.aws_partition}:s3:::${var.s3_bucket}/${var.s3_bucket_prefix}/${var.cluster_name}/node/*",
        ]
      },
    ]
  })
  tags = var.tags
}

resource "aws_iam_role" "node" {
  assume_role_policy = local.assume_role_policy_ec2
  name               = "keights-node-${var.cluster_name}"
  tags               = var.tags
}

resource "aws_iam_role_policy_attachment" "node" {
  role       = aws_iam_role.node.name
  policy_arn = aws_iam_policy.node.arn
}

resource "aws_iam_role_policy_attachment" "node_cni" {
  role       = aws_iam_role.node.name
  policy_arn = "arn:${var.aws_partition}:iam::aws:policy/AmazonEKS_CNI_Policy"
}

resource "aws_iam_role_policy_attachment" "node_ecr" {
  role       = aws_iam_role.node.name
  policy_arn = "arn:${var.aws_partition}:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

resource "aws_iam_instance_profile" "node" {
  name = "keights-node-${var.cluster_name}"
  role = aws_iam_role.node.name
  tags = var.tags
}

resource "aws_iam_policy" "lambda_logs" {
  name = "keights-lambda-logs-${var.cluster_name}"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
        ]
        Resource = [
          "arn:${var.aws_partition}:logs:*:*:*",
        ]
      },
    ]
  })
  tags = var.tags
}

resource "aws_iam_policy" "lambda_vpc" {
  count = var.features.lambda_vpc ? 1 : 0

  name = "keights-lambda-vpc-${var.cluster_name}"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ec2:CreateNetworkInterface",
          "ec2:DescribeNetworkInterfaces",
          "ec2:DeleteNetworkInterface",
          "ec2:AssignPrivateIpAddresses",
          "ec2:UnassignPrivateIpAddresses",
        ]
        Resource = ["*"]
      },
    ]
  })
  tags = var.tags
}

resource "aws_iam_policy" "lambda_auto_namer" {
  name = "keights-lambda-auto-namer-${var.cluster_name}"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ec2:DescribeInstances",
        ]
        Resource = ["*"]
      },
      {
        Effect = "Allow"
        Action = [
          "route53:ChangeResourceRecordSets",
        ]
        Resource = [
          "arn:${var.aws_partition}:route53:::hostedzone/${var.route53_hosted_zone_id}"
        ]
      },
    ]
  })
  tags = var.tags
}

resource "aws_iam_role" "lambda_auto_namer" {
  assume_role_policy = local.assume_role_policy_lambda
  name               = "keights-auto-namer-${var.cluster_name}"
  tags               = var.tags
}

resource "aws_iam_role_policy_attachment" "lambda_auto_namer_logs" {
  role       = aws_iam_role.lambda_auto_namer.name
  policy_arn = aws_iam_policy.lambda_logs.arn
}

resource "aws_iam_role_policy_attachment" "lambda_auto_namer_vpc" {
  count = var.features.lambda_vpc ? 1 : 0

  role       = aws_iam_role.lambda_auto_namer.name
  policy_arn = aws_iam_policy.lambda_vpc[0].arn
}

resource "aws_iam_role_policy_attachment" "lambda_auto_namer" {
  role       = aws_iam_role.lambda_auto_namer.name
  policy_arn = aws_iam_policy.lambda_auto_namer.arn
}

resource "aws_iam_policy" "lambda_kube_ca" {
  name = "keights-lambda-kube-ca-${var.cluster_name}"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ssm:DescribeParameters",
        ]
        Resource = ["*"]
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:DeleteParameter",
          "ssm:DeleteParameters",
          "ssm:GetParameters",
          "ssm:GetParametersByPath",
          "ssm:PutParameter",
        ]
        Resource = [
          "arn:${var.aws_partition}:ssm:${var.aws_region}:${var.aws_account_id}:parameter/keights/${var.cluster_name}/*",
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
          "kms:Encrypt",
        ]
        Resource = [
          "arn:${var.aws_partition}:kms:${var.aws_region}:${var.aws_account_id}:key/${var.kms_key_id}",
        ]
      },
    ]
  })
  tags = var.tags
}

resource "aws_iam_role" "lambda_kube_ca" {
  assume_role_policy = local.assume_role_policy_lambda
  name               = "keights-kube-ca-${var.cluster_name}"
  tags               = var.tags
}

resource "aws_iam_role_policy_attachment" "lambda_kube_ca_logs" {
  role       = aws_iam_role.lambda_kube_ca.name
  policy_arn = aws_iam_policy.lambda_logs.arn
}

resource "aws_iam_role_policy_attachment" "lambda_kube_ca_vpc" {
  count = var.features.lambda_vpc ? 1 : 0

  role       = aws_iam_role.lambda_kube_ca.name
  policy_arn = aws_iam_policy.lambda_vpc[0].arn
}

resource "aws_iam_role_policy_attachment" "lambda_kube_ca" {
  role       = aws_iam_role.lambda_kube_ca.name
  policy_arn = aws_iam_policy.lambda_kube_ca.arn
}
