# keights-stack

An Ansible role to provision a Kubernetes cluster using CloudFormation.

# Requirements

AWS credentials, via environment variables or an EC2 instance profile.

Python dependencies are listed in `requirements.txt`.

# Role Variables

All role variables go under a top level dictionary `keights_stack`.

`cluster_name`: (Required, type *string*) - Unique name given to cluster.

`state`: (Optional, choice of `present` or `absent`, default `present`) - Whether or not the cluster is present or absent.

`vpc_id`: (Required, type *string*) - Amazon VPC ID.

`kms_key_id`: (Conditional, type *string*, required if `create_iam_resources` is `true`) - ID of KMS key used for encrypting certificates.

`kms_key_alias`: (Required, type *string*) - Alias of KMS key given above.

`api_access_cidr`: (Required, type *string*) - CIDR block given access to the Kubernetes API load balancer.

`ssh_access_cidr`: (Required, type *string*) - CIDR block given ssh access to cluster nodes.

`node_port_access_cidr`: (Optional, type *string*) - CIDR block given access to NodePort services. If not defined, then NodePorts are not exposed.

`resource_bucket`: (Required, type *string*) - S3 bucket used for storing and retrieving artifacts.

`cluster_domain`: (Optional, type *string*, default `cluster.local`) - Domain used by internal Kubernetes network.

`etcd_domain`: (Optional, type *string*, default `{{cluster_name}}.local`) - Domain used by etcd servers, by default this is derived from the cluster name.

`etcd_hosted_zone_id`: (Optional, type *string*, default `''`) - Route53 hosted zone ID used for etcd DNS records. If not provided, a private hosted zone will be created. The name of the zone with this ID must match the value of `etcd_domain`.

`etcd_prefix`: (Optional, type *string*, default `etcd`) - Prefix given to etcd DNS records. This will be combined with the availability zone and the value of the `etcd_domain` parameter.

`etcd_mode` (Optional, choice of `stacked` or `external`, default `stacked`) - If `stacked`, then etcd runs on the masters. If `external`, then etcd runs on its own instances.

`cfn_role_arn`: (Optional, type *string*) - IAM service role ARN to be passed to CloudFormation. See [AWS documentation on using CloudFormation with a service role](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/using-iam-servicerole.html) for more details.

`k8s_version`: (Optional, type *string*) - Version of Kubernetes. This defaults to the version corresponding with the `keights-stack` version, for example if the `keights-stack` version is `1.10.7-3`, then `k8s_version` is `1.10.7`. Versions other than the default will not be tested.

`image`: (Optional, type *string*) - The default AMI to use for both masters and nodes. This should be the name of the image, rather than an AMI ID. This defaults to `debian-buster-k8s-hvm-amd64-v<version>`, where `<version>` is the keights version. A public image with this name is available in `us-east-1`, so if you are not running there, you may copy it into your own region. If more than one image is found with the same name, the first one is used.

`image_owner`: (Optional, type *string*, default `256008164056`) - AWS account owning the AMI. Set this if using your own image.

`lookup_image`: (Optional, type *bool*, default `true`) - This may be set to `false` if not using `image` at all. In this case, `image_id` must be specified for the masters and all node groups.

`create_iam_resources`: (Optional, type *bool*, default `true`) - Whether or not to create IAM roles and policies for the cluster. If `false`, then IAM roles will need to be created another way and passed as parameters to the remaining stacks.

`instattr_lambda_role_arn`: (Conditional, type *string*, required if `create_iam_resources` is `false`) - The ARN of the IAM role for the InstAttr lambda.

`auto_namer_lambda_role_arn`: (Conditional, type *string*, required if `create_iam_resources` is `false`) - The ARN of the IAM role for the AutoNamer lambda.

`kube_ca_lambda_role_arn`: (Conditional, type *string*, required if `create_iam_resources` is `false`) - The ARN of the IAM role for the KubeCA lambda.

`subnet_to_az_lambda_role_arn`: (Conditional, type *string*, required if `create_iam_resources` is `false`) - The ARN of the IAM role for the SubnetToAZ lambda.

`masters`: (Required, type *dict*) - A dictionary of variables for Kubernetes masters, described below.

`etcd`: (Optional, type *dict*) - A dictionary of variables used when `etcd_mode` is `external`, described below.

`node_groups`: (Optional, type *list* of *dict*, default `[]`) - A list of dictionaries, each one defining a group of Kubernetes nodes, described below.

### masters

`service_cidr`: (Required, type *string*) - In-cluster CIDR block used for Kubernetes services.

`pod_cidr`: (Required, type *string*) - In-cluster CIDR block used for Kubernetes pods.

`subnet_ids`: (Required, type *list* of *string*) - Subnet IDs in which masters will live. When `etcd_mode` is `stacked`, each subnet *must* be in a separate availability zone, and the number of subnet IDs will determine the number of masters. Either one or three subnets may be used for the masters when `etcd_mode` is `stacked`. When `etcd_mode` is `external`, there can be any number of subnet IDs in any availability zones.

`instance_type`: (Required, type *string*) - Type of EC2 instance, e.g. `m4.large`.

`keypair`: (Required, type *string*) - SSH keypair assigned to EC2 instances.

`instance_profile`: (Conditional, type *string*, required if `create_iam_resources` is `false`) - The name of the IAM instance profile assigned to EC2 instances.

`load_balancer_scheme`: (Required, choice of `internal` or `internet-facing`) - Scheme assigned to Kubernetes API load balancer.

`load_balancer_idle_timeout`: (Optional, type int, default `600`) - Idle timeout on Kubernetes API load balancer.

`num_instances`: (Optional, type *int*, default `1`) - Number of master instances when `etcd_mode` is `external`.

`image_id`: (Optional, type *string*) - EC2 AMI ID, will override `keights_stack.image`.

`extra_security_groups`: (Optional, type *list* of *string*, default `[]`) - Additional security groups that may be assigned to masters.

`etcd_volume_size`: (Optional, type *int*, default `10`) - Size of etcd volume in gigabytes.

`etcd_device`: (Optional, type *string*, default `/dev/xvdg`) - Name of etcd EBS volume device.

`image_repository`: (Optional, type *string*, default `k8s.gcr.io`) - Repository from which Kubernetes component docker images are pulled.

`docker_options`: (Optional, type *dict*, default `{"ip-masq": false, "iptables": false, "log-driver": "journald", "storage-driver": "overlay2"}`) - Options to write to `/etc/docker/daemon.json`, which should follow [documentation for docker](https://docs.docker.com/engine/reference/commandline/dockerd/#daemon-configuration-file).

`kubeadm_init_config_template`: (Optional, type *string*, default `''`) - A kubeadm init [configuration file](https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-init/#config-file) as a Go template string. If not defined, a default one will be used which is built into the AMI. See [Kubeadm init](#kubeadm-init) below for a description of the variables that will be available within the template. Due to CloudFormation parameter limitations, this string must not be over 4kb.

### etcd

`subnet_ids`: (Required, type *list* of *string*) - Subnet IDs in which etcd instances will live. Each subnet *must* be in a separate availability zone, and the number of subnet IDs will determine the number of instances. Either one or three subnets may be used.

`image_id`: (Optional, type *string*) - EC2 AMI ID, will override `keights_stack.image`.

`instance_type`: (Required, type *string*) - Type of EC2 instance, e.g. `m4.large`.

`keypair`: (Required, type *string*) - SSH keypair assigned to EC2 instances.

`instance_profile`: (Conditional, type *string*, required if `create_iam_resources` is `false`) - The name of the IAM instance profile assigned to EC2 instances.

`extra_security_groups`: (Optional, type *list* of *string*, default `[]`) - Additional security groups that may be assigned to etcd instances.

`volume_size`: (Optional, type *int*, default `10`) - Size of etcd volume in gigabytes.

`device`: (Optional, type *string*, default `/dev/xvdg`) - Name of etcd EBS volume device.

`image_repository`: (Optional, type *string*, default `k8s.gcr.io`) - Repository from which docker image is pulled.

`docker_options`: (Optional, type *dict*, default `{"ip-masq": false, "iptables": false, "log-driver": "journald", "storage-driver": "overlay2"}`) - Options to write to `/etc/docker/daemon.json`, which should follow [documentation for docker](https://docs.docker.com/engine/reference/commandline/dockerd/#daemon-configuration-file).

`instance_profile`: (Conditional, type *string*, required if `create_iam_resources` is `false`) - The name of the IAM instance profile assigned to EC2 instances.

### node_groups

`name`: (Required, type *string*) - Unique name given to node group.

`min_instances`: (Required, type *int*) - Minimum number of EC2 instances in group.

`max_instances`: (Required, type *int*) - Maximum number of EC2 instances in group.

`update_max_batch_size`: (Optional, type *int*, default `1`) - On a stack update, the maximum number of EC2 instances to update at a time.

`subnet_ids`: (Required, type *list* of *string*) - Subnet IDs in which nodes will live.

`instance_type`: (Required, type *string*) - Type of EC2 instance, e.g. `m4.large`.

`keypair`: (Required, type *string*) - SSH keypair assigned to EC2 instances.

`instance_profile`: (Conditional, type *string*, required if `create_iam_resources` is `false`) - The name of the IAM instance profile assigned to EC2 instances.

`image_id`: (Optional, type *string*) - EC2 AMI ID, will override `keights_stack.image`.

`extra_security_groups`: (Optional, type *list* of *string*, default `[]`) - Additional security groups that may be assigned to nodes.

`node_labels`: (Optional, type *dict*, default `{}`) - A dictionary of key/value pairs used to assign Kubernetes labels to nodes.

`image_repository`: (Optional, type *string*, default `k8s.gcr.io`) - Repository from which Kubernetes component docker images are pulled.

`docker_options`: (Optional, type *dict*, default `{"ip-masq": false, "iptables": false, "log-driver": "journald", "storage-driver": "overlay2"}`) - Options to write to `/etc/docker/daemon.json`, which should follow [documentation for docker](https://docs.docker.com/engine/reference/commandline/dockerd/#daemon-configuration-file).

`kubeadm_join_config_template`: (Optional, type *string*, default `''`) - A kubeadm join [configuration file](https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-join/#config-file) as a Go template string. If not defined, a default one will be used which is built into the AMI. See [Kubeadm join](#kubeadm-join) below for a description of the variables that will be available within the template. Due to CloudFormation parameter limitations, this string must not be over 4kb.

`subnet_tags`: (Optional, type *dict*, default `{}`) - A dictionary of tags to add to node subnets. For example `{'kubernetes.io/cluster/cb': 'shared', 'kubernetes.io/role/internal-elb': '1'}`, where `cb` is the name of the cluster; this would allow the `cb` cluster to create internal ELBs in the node subnets. This is documented fully in the [EKS documentation](https://docs.aws.amazon.com/eks/latest/userguide/network_reqs.html#vpc-subnet-tagging), though it is not specific to EKS.

# Kubeadm Configuration Templates

The configuration files defined in `keights_stack.masters.kubeadm_init_config_template` and `keights_stack.node_groups[].kubeadm_join_config_template` should be Go templates, which have a number of variables passed to them before expansion.

## Kubeadm init

The file defined in `kubeadm_init_config_template` will have the following variables available:

`ClusterDomain` - The internal cluster domain, e.g. `cluster.local`.

`EtcdDomain` - The DNS domain where etcd hostnames are created.

`EtcdMode` - The mode in which etcd runs, either `stacked` or `external`.

`Prefix` - The prefix for etcd hostnames. It is combined with the availability zone and `EtcdDomain` to define the FQDN of the host. For example, if `Prefix` is `etcd`, `MyAZ` is `us-east-1a`, and `EtcdDomain` is `cloudboss.local`, the etcd hostname for that availability zone would be `etcd-us-east-1a.cloudboss.local`.

`APIServer` - The DNS name of the Kubernetes API server.

`APIPort` - The port of the Kubernetes API server.

`PodSubnet` - The subnet from which pod IPs are assigned.

`ServiceSubnet` - The subnet from which service IPs are assigned.

`ClusterDNS` - The IP address of the internal cluster DNS server.

`NodeName` - The hostname of the current machine.

`Token` - The token for bootstrapping the kubelet.

`ImageRepository` - The image repository from which control plane images are pulled.

`KubernetesVersion` - The version of Kubernetes.

`AZs` - The list of availability zones in which etcd is running.

`MyAZ` - The availability zone of the current machine.

`MyIP` - The IP address of the current machine.

## Kubeadm join

The file defined in `kubeadm_join_config_template` will have the following variables available:

`APIServer` - The DNS name of the Kubernetes API server.

`APIServerPort` - The port of the Kubernetes API server.

`Token` - The token for bootstrapping the kubelet.

`CACertHash` - The sha256 hash of the cluster CA certificate.

`ImageRepository` - The image repository from which control plane images are pulled.

`NodeLabels` - A list of node labels in `key=value` form.

`NodeName` - The hostname of the current machine.

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
        subnet_ids: '{{ subnet_ids }}'
        instance_type: t2.large
        keypair: '{{ keypair }}'
        node_labels:
          class: app
          env: dev
```

# License

MIT

# Author Information

Joseph Wright <joseph@cloudboss.co>
