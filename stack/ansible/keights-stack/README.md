# keights-stack

An Ansible role to provision a Kubernetes cluster using CloudFormation.

# Requirements

AWS credentials, via environment variables or an EC2 instance profile.

Python dependencies are listed in `requirements.txt`.

# Role Variables

All role variables go under a top level dictionary `keights_stack`.

`cluster_name`: (Required, type *string*) - Unique name given to cluster.

`vpc_id`: (Required, type *string*) - Amazon VPC ID.

`kms_key_id`: (Required, type *string*) - ID of KMS key used for encrypting certificates.

`kms_key_alias`: (Required, type *string*) - Alias of KMS key given above.

`api_access_cidr`: (Required, type *string*) - CIDR block given access to the Kubernetes API load balancer.

`ssh_access_cidr`: (Required, type *string*) - CIDR block given ssh access to cluster nodes.

`resource_bucket`: (Required, type *string*) - S3 bucket used for storing and retrieving artifacts.

`k8s_version`: (Optional, type *string*) - Version of Kubernetes. This defaults to the version corresponding with the `keights-stack` version, for example if the `keights-stack` version is `1.10.7-3`, then `k8s_version` is `1.10.7`. Versions other than the default will not be tested.

`image`: (Optional, type *string*) - The default AMI to use for both masters and nodes. This should be the name of the image, rather than an AMI ID. This defaults to `debian-stretch-k8s-hvm-amd64-v<version>`, where `<version>` is the keights version. A public image with this name is available in `us-east-1`, so if you are not running there, you may copy it into your own region. If more than one image is found with the same name, the first one is used.

`image_owner`: (Optional, type *string*, default `256008164056`) - AWS account owning the AMI. Set this if using your own image.

`lookup_image`: (Optional, type *bool*, default `true`) - This may be set to `false` if not using `image` at all. In this case, `image_id` must be specified for the masters and all node groups.

`masters`: (Required, type *dict*) - A dictionary of variables for Kubernetes masters, described below.

`node_groups`: (Optional, type *list* of *dict*, default `[]`) - A list of dictionaries, each one defining a group of Kubernetes nodes, described below.

### masters

`service_cidr`: (Required, type *string*) - In-cluster CIDR block used for Kubernetes services.

`pod_cidr`: (Required, type *string*) - In-cluster CIDR block used for Kubernetes pods.

`subnet_ids`: (Required, type *list* of *string*) - Subnet IDs in which masters will live. Each subnet *must* be in a separate availability zone, and the number of subnet IDs will determine the number of masters.

`instance_type`: (Required, type *string*) - Type of EC2 instance, e.g. `m4.large`.

`keypair`: (Required, type *string*) - SSH keypair assigned to EC2 instances.

`load_balancer_scheme`: (Required, choice of `internal` or `internet-facing`) - Scheme assigned to Kubernetes API load balancer.

`load_balancer_idle_timeout`: (Optional, type int, default `600`) - Idle timeout on Kubernetes API load balancer.

`image_id`: (Optional, type *string*) - EC2 AMI ID, will override `keights_stack.image`.

`extra_security_groups`: (Optional, type *list* of *string*, default `[]`) - Additional security groups that may be assigned to masters.

`etcd_volume_size`: (Optional, type *int*, default `10`) - Size of etcd volume in gigabytes.

`etcd_device`: (Optional, type *string*, default `xvdg`) - Name of etcd EBS volume device.

`etcd_internal_device`: (Optional, type *string*, default `/dev/xvdg`) - Name of etcd volume device within machine, which may differ between EC2 instance types.

`etcd_internal_device`: (Optional, type *string*, default `/dev/xvdg`) - Name of etcd volume device within machine, which may differ between EC2 instance types.

`image_repository`: (Optional, type *string*, default `k8s.gcr.io`) - Repository from which Kubernetes component docker images are pulled.

### node_groups

`name`: (Required, type *string*) - Unique name given to node group.

`min_instances`: (Required, type *int*) - Minimum number of EC2 instances in group.

`max_instances`: (Required, type *int*) - Maximum number of EC2 instances in group.

`update_max_batch_size`: (Optional, type *int*, default `1`) - On a stack update, the maximum number of EC2 instances to update at a time.

`subnet_ids`: (Required, type *list* of *string*) - Subnet IDs in which nodes will live.

`instance_type`: (Required, type *string*) - Type of EC2 instance, e.g. `m4.large`.

`keypair`: (Required, type *string*) - SSH keypair assigned to EC2 instances.

`image_id`: (Optional, type *string*) - EC2 AMI ID, will override `keights_stack.image`.

`extra_security_groups`: (Optional, type *list* of *string*, default `[]`) - Additional security groups that may be assigned to nodes.

`node_labels`: (Optional, type *dict*, default `{}`) - A dictionary of key/value pairs used to assign Kubernetes labels to nodes.

# Dependencies

A list of other roles hosted on Galaxy should go here, plus any details in regards to parameters that may need to be set for other roles, or variables that are used from other roles.

# Example Playbook

```
- hosts: localhost
  connection: local
  vars:
    cluster_name: cb
    vpc_id: vpc-ba92ad08
    resource_bucket: cloudboss-public
    subnet_ids:
	- subnet-6c31888d
	- subnet-f3cdd152
	- subnet-e245724f
    keypair: keights
    kms_key_id: 714e0cab-0d59-4885-b45b-31d44467fe5c
    kms_key_alias: alias/cloudboss
    ssh_access_cidr: 0.0.0.0/0

- hosts: localhost
  connection: local
  roles:
  - role: keights-stack
    keights_stack:
      cluster_name: '{{ cluster_name }}'
      vpc_id: '{{ vpc_id }}'
      kms_key_id: '{{ kms_key_id }}'
      kms_key_alias: '{{ kms_key_alias }}'
      api_access_cidr: 0.0.0.0/0
      ssh_access_cidr: '{{ ssh_access_cidr }}'
      keights_version: '{{ keights_version }}'
      resource_bucket: '{{ resource_bucket }}'
      masters:
        service_cidr: 10.1.0.0/16
        pod_cidr: 10.0.0.0/16
        subnet_ids: '{{ subnet_ids }}'
        instance_type: t2.large
        keypair: '{{ keypair }}'
        load_balancer_scheme: internet-facing
        etcd_volume_size: 10
      node_groups:
      - name: app
        min_instances: 1
        max_instances: 3
        vpc_id: '{{ vpc_id }}'
        subnet_ids: '{{ subnet_ids }}'
        instance_type: t2.large
        keypair: '{{ keypair }}'
        ssh_access_cidr: '{{ ssh_access_cidr }}'
        node_labels:
          class: app
          env: dev
```

# License

MIT

# Author Information

Joseph Wright <joseph@cloudboss.co>
