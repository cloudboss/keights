# Terraform

This directory defines a Terraform module to deploy [keights](https://github.com/cloudboss/keights) clusters.

This module is distributed as a GitHub release tarball and can be referenced by URL:

```hcl
module "keights" {
  source = "https://github.com/cloudboss/keights/releases/download/v2.0.0/keights-terraform-v2.0.0.tar.gz"
```

See the [example](./example) directory for a sample that follows the [DELAR](https://www.cloudboss.co/articles/delar-model/) model.

# Inputs

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| access\_cidrs | CIDR blocks that are allowed access to Kubernetes. | [object](#access_cidrs-object) | `{}` | no |
| addons | Configuration for addon Helm charts. | [object](#addons-object) | `{}` | no |
| ami | Configuration of the AMI to use for instances. One of `filters` or `name` must be set. | [object](#ami-object) | `{}` | no |
| cluster\_name | The name of the Kubernetes cluster. | string | N/A | yes |
| control\_plane | Configuration for the control plane. | [object](#control_plane-object) | N/A | yes |
| encryption\_algorithm | The encryption algorithm used for cluster encryption. Must be one of `ECDSA-P256`, `ECDSA-P384`, `RSA-3072`, or `RSA-4096`. | string | `ECDSA-P256` | no |
| etcd\_mode | The etcd cluster deployment mode. Must be one of `external` or `stacked`. In `stacked` mode, etcd runs on the control plane nodes. In `external` mode, etcd runs on separate instances. | string | `stacked` | no |
| identity\_mappings | Configuration for aws-iam-authenticator identity mappings. By default the IAM identity of the cluster creator will have administrative access and nodes will have access to join the cluster. | list([object](#identity_mappings-object)) | `[]` | no |
| kms\_key\_id | The KMS Key ID used for cluster encryption. | string | N/A | yes |
| kubernetes\_configuration | Kubernetes configuration options. | [object](#kubernetes_configuration-object) | `{}` | no |
| lambda | Configuration for Lambda functions. If `subnet_ids` are not provided, they will be deployed without VPC configuration. | [object](#lambda-object) | `{}` | no |
| node\_groups | Configuration for node groups. Each key is the name of a node group and the value is a configuration object. | map([object](#node_group-object)) | `{}` | no |
| route53 | Configuration for Route53 private zone for etcd instances. If `hosted_zone_id` is not provided, a new private hosted zone will be created. | [object](#route53-object) | `{}` | no |
| s3 | Configuration for S3 bucket used for storing cluster assets. If `bucket_name` is not provided, a new bucket will be created. | [object](#s3-object) | `{}` | no |
| storage | Configuration for storage defaults. The containerd block configures defaults for all cluster machines, but can be overridden individually on the control plane and node groups. | [object](#storage-object) | `{}` | no |
| tags | A map of tags to assign to resources. | map(string) | `null` | no |
| vpc\_id | The VPC ID where resources will be deployed. | string | N/A | yes |

## access\_cidrs object

An object to configure the CIDR blocks that will be given access to the cluster.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| api | CIDR blocks allowed to access the Kubernetes API. | list(string) | `["0.0.0.0/0"]` | no |
| node\_ports | CIDR blocks allowed to access node ports. | list(string) | `[]` | no |
| ssh | CIDR blocks allowed SSH access. | list(string) | `[]` | no |

## addons object

An object to configure cluster addons. Addons are deployed with Helm charts, and each addon is configured the same way, taking a values object to override the defaults.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| aws\_cloud\_controller\_manager | Configuration for the AWS cloud controller manager addon. | [object](#addon-object) | `{}` | no |
| aws\_iam\_authenticator | Configuration for the AWS IAM authenticator addon. | [object](#addon-object) | `{}` | no |
| aws\_vpc\_cni | Configuration for the AWS VPC CNI addon. | [object](#addon-object) | `{}` | no |
| cert\_manager | Configuration for the cert-manager addon. | [object](#addon-object) | `{}` | no |

## addon object

The addon object with values passed to Helm.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| values | Additional Helm values to merge into the addon chart. | map(any) | `{}` | no |

## ami object

An object to configure the default cluster AMI. This can be overridden at the control plane or node group levels.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| filters | Filters to search for an AMI. Required if `name` is not defined. | list(object) | `[]` | conditional |
| most\_recent | Whether or not to return the most recent image found. | bool | `true` | no |
| name | Name of the AMI. Required if `filters` is not defined. | string | `""` | conditional |
| owner | AWS account where the image is located. | string | `""` | no |

## control\_plane object

An object to configure the control plane.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| debug\_logging | Enable debug logging on control plane nodes. | bool | `null` | no |
| instance\_type | Type of the EC2 instances. | string | N/A | yes |
| internal | Whether or not the load balancer is internal. | bool | `true` | no |
| key\_pair | Name of an SSH key pair to assign to instances. If none is assigned, sshd will not run. | string | `null` | no |
| monitoring\_enabled | Enable detailed monitoring on instances. | bool | `true` | no |
| storage | Override storage configuration for control plane nodes. | any | `null` | no |
| subnet\_ids | Subnet IDs for the autoscaling group and load balancer. | [object](#control_plane-subnet_ids-object) | N/A | yes |

### control\_plane subnet\_ids object

An object to configure subnet IDs for the control plane.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| autoscaling\_group | Subnets for control plane instances. Must be an odd number. | set(string) | N/A | yes |
| load\_balancer | Subnets for the API server load balancer. | set(string) | N/A | yes |

## identity\_mappings object

The identity mappings object is used to configure [aws-iam-authenticator](https://github.com/kubernetes-sigs/aws-iam-authenticator). By default, the identity that created the cluster will have administrative privileges, and nodes will have permission to bootstrap and join the cluster.

```
[
  {
    name = "kubernetes-admin"
    spec = {
      arn      = var.caller_identity.arn
      username = "kubernetes-admin"
      groups   = ["system:masters"]
    }
  },
  {
    name = "kubernetes-nodes"
    spec = {
      arn      = var.node_role_arn
      username = "system:node:{{EC2PrivateDNSName}}"
      groups   = ["system:bootstrappers", "system:nodes"]
    }
  },
]
```

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| name | Name of the identity mapping. | string | N/A | yes |
| spec | Specification of the identity mapping. | [object](#identity_mappings-spec-object) | N/A | yes |

### identity\_mappings spec object

Each identity mappings `spec` object has the following structure.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| arn | IAM ARN to map. | string | N/A | yes |
| username | Kubernetes username for the mapping. | string | N/A | yes |
| groups | Kubernetes groups for the mapping. | list(string) | N/A | yes |

## kubernetes\_configuration object

An object to configure internal Kubernetes settings.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| cluster\_domain | The cluster DNS domain. | string | `cluster.local` | no |
| image\_registry | The container image registry for Kubernetes components. | string | `registry.k8s.io` | no |
| service\_subnet | The CIDR for Kubernetes services. | string | `10.96.0.0/12` | no |
| version | The Kubernetes version. | string | `1.34.1` | no |

## lambda object

Configuration for Lambdas that are deployed with the cluster.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| s3 | S3 configuration for Lambda deployment packages. | [object](#lambda-s3-object) | `{}` | no |
| subnet\_ids | VPC subnet IDs for Lambda functions. If not provided, Lambdas are deployed without VPC configuration. | list(string) | `null` | no |
| version | Version of the Lambda deployment packages. | string | `0.1.0` | no |

## lambda s3 object

Configuration of S3 settings for Lambdas.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| bucket | S3 bucket containing Lambda deployment packages. | string | `cloudboss-keights` | no |
| prefix | S3 key prefix for Lambda deployment packages. | string | `lambda` | no |

## node\_group object

A configuration object for a node group.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| ami | Override the AMI for this node group. Same structure as the top-level [ami](#ami-object). | any | `null` | no |
| autoscaling\_group | Autoscaling group configuration. | [object](#node_group-autoscaling_group-object) | `{}` | no |
| debug\_logging | Enable debug logging on nodes. | bool | `null` | no |
| extra\_security\_group\_ids | Additional security group IDs to attach to instances. | list(string) | `null` | no |
| iam\_instance\_profile | An existing IAM instance profile to use instead of creating one. | string | `null` | no |
| instance\_refresh | Instance refresh configuration. See the upstream [asg module](https://github.com/cloudboss/terraform-aws-asg) for the structure. | any | `{ strategy = "Rolling" }` | no |
| instance\_type | Type of the EC2 instances. Required if `mixed_instances_overrides` is not defined. | string | `null` | conditional |
| key\_pair | Name of an SSH key pair to assign to instances. | string | `null` | no |
| mixed\_instances\_overrides | A list of override objects for mixed instances. See the upstream [asg module](https://github.com/cloudboss/terraform-aws-asg) for the structure. Required if `instance_type` is not defined. | list(any) | `[]` | conditional |
| monitoring\_enabled | Enable detailed monitoring on instances. | bool | `true` | no |
| storage | Override storage configuration for this node group. | any | `null` | no |
| subnet\_ids | Subnets for instances in this node group. | set(string) | `null` | no |

## node\_group autoscaling\_group object

The autoscaling group settings for a node group.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| instances\_desired | The initial number of instances desired. | number | `2` | no |
| instances\_max | The maximum number of instances. | number | `5` | no |
| instances\_min | The minimum number of instances. | number | `0` | no |

## route53 object

An object to configure `route53` settings, used for etcd hostnames.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| hosted\_zone\_id | ID of an existing Route53 private hosted zone. If not provided, a new zone is created. | string | `null` | no |
| hosted\_zone\_name | Name of the hosted zone to create. Defaults to `${cluster_name}.local`. | string | `null` | no |

## s3 object

An object to configure settings for the S3 bucket used to host cluster resources such as config files and templates.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| bucket\_name | Name of an existing S3 bucket. If not provided, a new bucket is created. | string | `null` | no |
| bucket\_prefix | Prefix for the auto-generated bucket name. | string | `keights` | no |

## storage object

An object to configure storage. The `contained` object defines defaults for the whole cluster, but can be overridden at the control plane or node group levels.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| containerd | Storage configuration for the containerd data volume. | [object](#storage-volume-object) | `{}` | no |
| etcd | Storage configuration for the etcd data volume. | [object](#storage-volume-object) | `{}` | no |

## storage volume object

An object to configure storage volumes.

| Name | Description | Type | Default | Required |
|------|-------------|:----:|:-------:|:--------:|
| device | The block device name. | string | `/dev/sdf` (containerd) or `/dev/sdg` (etcd) | no |
| iops | Number of IOPS for the volume. | number | `null` | no |
| size | Size of the volume in GB. | number | `10` | no |
| type | Type of the EBS volume. | string | `gp3` | no |

# Outputs

| Name | Description |
|------|-------------|
| load\_balancer\_dns\_name | The DNS name of the API server load balancer. |
