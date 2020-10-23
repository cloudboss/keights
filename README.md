# keights

Keights (rhymes with "heights") is a [Kubernetes](https://kubernetes.io/) installer for [AWS](https://aws.amazon.com/), using [CloudFormation](https://aws.amazon.com/cloudformation/) and [Ansible](https://docs.ansible.com/ansible/latest/index.html).

CloudFormation templates define all of the AWS resources, such as the API load balancer, autoscaling groups, security groups, and IAM roles.

Ansible roles provide end to end automation for the installation, doing setup tasks, building the CloudFormation stacks, and finally adding the [network plugin](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/) and dashboard to the cluster.

You can also use the CloudFormation templates without Ansible if you want to use them as a starting point for your own custom installer, or to integrate with existing CloudFormation-based automation.

# CI status

All builds and tests are run on [Concourse CI](https://ci.cloudboss.xyz/teams/keights). Every supported Kubernetes minor version has a corresponding git branch in the Keights repository. Each of these branches has its own pipeline with build statuses shown below. All `build-cluster-*` and `upgrade-cluster-*` jobs run conformance tests against the cluster using [Sonobuoy](https://github.com/heptio/sonobuoy).

| Job | Version | Status |
| ----- | ------- | ------ |
| build-pull-request | 1.19 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.19/jobs/build-pull-request/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.19/jobs/build-pull-request/builds/latest) |
| build-cluster-external-etcd | 1.19 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.19/jobs/build-cluster-external-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.19/jobs/build-cluster-external-etcd/builds/latest) |
| build-cluster-stacked-etcd | 1.19 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.19/jobs/build-cluster-stacked-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.19/jobs/build-cluster-stacked-etcd/builds/latest) |
| upgrade-cluster-external-etcd | 1.18 -> 1.19 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.19/jobs/upgrade-cluster-external-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.19/jobs/upgrade-cluster-external-etcd/builds/latest) |
| upgrade-cluster-stacked-etcd | 1.18 -> 1.19 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.19/jobs/upgrade-cluster-stacked-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.19/jobs/upgrade-cluster-stacked-etcd/builds/latest) |
| build-release | 1.19 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.19/jobs/build-release/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.19/jobs/build-release/builds/latest) |
||||
| build-pull-request | 1.18 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.18/jobs/build-pull-request/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.18/jobs/build-pull-request/builds/latest) |
| build-cluster-external-etcd | 1.18 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.18/jobs/build-cluster-external-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.18/jobs/build-cluster-external-etcd/builds/latest) |
| build-cluster-stacked-etcd | 1.18 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.18/jobs/build-cluster-stacked-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.18/jobs/build-cluster-stacked-etcd/builds/latest) |
| upgrade-cluster-external-etcd | 1.17 -> 1.18 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.18/jobs/upgrade-cluster-external-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.18/jobs/upgrade-cluster-external-etcd/builds/latest) |
| upgrade-cluster-stacked-etcd | 1.17 -> 1.18 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.18/jobs/upgrade-cluster-stacked-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.18/jobs/upgrade-cluster-stacked-etcd/builds/latest) |
| build-release | 1.18 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.18/jobs/build-release/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.18/jobs/build-release/builds/latest) |
||||
| build-pull-request | 1.17 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.17/jobs/build-pull-request/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.17/jobs/build-pull-request/builds/latest) |
| build-cluster-external-etcd | 1.17 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.17/jobs/build-cluster-external-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.17/jobs/build-cluster-external-etcd/builds/latest) |
| build-cluster-stacked-etcd | 1.17 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.17/jobs/build-cluster-stacked-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.17/jobs/build-cluster-stacked-etcd/builds/latest) |
| upgrade-cluster-external-etcd | 1.16 -> 1.17 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.17/jobs/upgrade-cluster-external-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.17/jobs/upgrade-cluster-external-etcd/builds/latest) |
| upgrade-cluster-stacked-etcd | 1.16 -> 1.17 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.17/jobs/upgrade-cluster-stacked-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.17/jobs/upgrade-cluster-stacked-etcd/builds/latest) |
| build-release | 1.17 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.17/jobs/build-release/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.17/jobs/build-release/builds/latest) |
||||
| build-pull-request | 1.16 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.16/jobs/build-pull-request/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.16/jobs/build-pull-request/builds/latest) |
| build-cluster-external-etcd | 1.16 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.16/jobs/build-cluster-external-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.16/jobs/build-cluster-external-etcd/builds/latest) |
| build-cluster-stacked-etcd | 1.16 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.16/jobs/build-cluster-stacked-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.16/jobs/build-cluster-stacked-etcd/builds/latest) |
| upgrade-cluster-external-etcd | 1.15 -> 1.16 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.16/jobs/upgrade-cluster-external-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.16/jobs/upgrade-cluster-external-etcd/builds/latest) |
| upgrade-cluster-stacked-etcd | 1.15 -> 1.16 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.16/jobs/upgrade-cluster-stacked-etcd/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.16/jobs/upgrade-cluster-stacked-etcd/builds/latest) |
| build-release | 1.16 | [![Build Status](https://ci.cloudboss.xyz/api/v1/teams/keights/pipelines/keights-v1.16/jobs/build-release/badge)](https://ci.cloudboss.xyz/teams/keights/pipelines/keights-v1.16/jobs/build-release/builds/latest) |

# Rationale

Now that AWS offers [EKS](https://aws.amazon.com/eks/), why bother managing your own cluster?

When this project was started, EKS hadn't been announced. It was some time after the announcement before it was even minimally ready. By that time, keights was already working and offering an AWS native approach to managing Kubernetes.

Here are some reasons you might still consider using it:

* Keights manages the whole cluster, not just the control plane like EKS.

* Keights helps manage the cluster life cycle, with well-tested rolling upgrades from previous versions.

* Keights is vanilla Kubernetes configured by [kubeadm](https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm/), stitched together by Ansible and CloudFormation.

* Keights supports more current versions of Kubernetes than EKS.

* Keights allows a single master control plane or an HA control plane, whereas EKS only supports an HA control plane. A single master can save costs for small development clusters.

* Keights lets you choose the EC2 instance size for your control plane.

* Compared to some other installers, Keights stays out of the way of managing your VPC and subnets. It assumes you have already designed the network the way you want, or that you may be constrained by corporate policies that disallow modifying the VPC.

* Keights was designed to work in air gapped environments and does not require an internet gateway.

* You want to experiment or develop on Kubernetes and need more control.

* Compared to some installers, Keights is fully AWS native. Because it is plain CloudFormation at its core, you can integrate it into your AWS accounts your way, for example with [stack sets](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/stacksets-getting-started-create.html) or [AWS Service Catalog](https://docs.aws.amazon.com/servicecatalog/latest/adminguide/introduction.html).

# Releases

Keights releases follow the Kubernetes version with an incrementing suffix appended. For example, Kubernetes `1.10.0` had Keights releases `v1.10.0-1`, `v1.10.0-2`, `v1.10.0-3`, and `v1.10.0-4`.

Releases are downloaded from the [GitHub release page](https://github.com/cloudboss/keights/releases). When using Ansible to deploy, it downloads the release for you automatically.

# CloudFormation with Ansible

Keights builds clusters from several CloudFormation templates. All clusters use the `common.yml` and `node.yml` templates. Clusters with a [stacked etcd](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/ha-topology/) use `master-stacked.yml` to build the control plane, and clusters with an external etcd use `etcd.yml` and `master-external.yml` to build the control plane. You can create more than one instance of the `node.yml` stack for a cluster, using different parameters to define instance sizes and node labels.

Ansible does not run on machines in the cluster to configure them, as you might expect. For one thing, there is not a lot of configuration to be done; Keights follows an immutable infrastructure approach and uses a custom AMI with dependencies preinstalled. Per-cluster configuration variables are set in [user data](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html) in the CloudFormation templates. The remaining per-instance configuration is handled by the [`keights` binary](#the-keights-binary) and kubeadm via systemd services built into the AMI.

Instead of running on cluster machines, Ansible runs against your local machine or CI/CD tool of choice, and builds or updates the cluster via CloudFormation, which then configures itself. The first Ansible role, `keights-stack`, builds the CloudFormation stacks, passing outputs from one stack as inputs to the next as required. Another Ansible role, `keights-system`, adds the CNI network plugin and Kubernetes dashboard.

Using Ansible together with CloudFormation enables idempotent stack creation and updates. Ansible roles also act as a packaging format for the CloudFormation templates which can be versioned. Cluster upgrades are done by changing the role versions and rerunning Ansible.

## Building a cluster

### Requirements

Before you start, you will need:

* AWS credentials - Keights does not take AWS credentials as parameters. It expects them to be in scope in your environment. This allows a lot of flexibility in how and where you run Keights, and the underlying AWS SDK will take care of retrieving credentials and passing them to AWS. The standard way to do this is through a [credentials file](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html), [environment variables](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html), or an [instance profile](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use_switch-role-ec2_instance-profiles.html) if running Keights from an EC2 instance.

* AWS region - Set the environment variable `AWS_DEFAULT_REGION` to your desired region.

* An S3 bucket - Keights does not create S3 buckets, but you will need one to store the artifacts used to deploy Lambdas.

* A KMS key with an alias - Keights does not touch your KMS keys. If you don't have one, you can create a new key in the IAM AWS console under "Encryption Keys" and give it an alias.

* An AMI for the Keights version in your region - For each release, there is a public AMI published in the `us-east-1` region. If you are in another region, you will need to [copy it into your account and region](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/CopyingAMIs.html#ami-copy-steps). The published AMI can be found in the AWS console as a public image with owner `256008164056`, and is called `debian-buster-k8s-hvm-amd64-v${KeightsVersion}`, where `${KeightsVersion}` is replaced with the actual version. If you do copy the image to your account, keep the same name to make things easier.

* Python - Ansible will be installed into a [virtualenv](https://virtualenv.pypa.io/en/latest/), so you need either Python 3, or Python 2 with the `virtualenv` command installed.

### Deployment script

A script called `deploy` is included in the `stack/ansible` directory of the Keights repository. It is a wrapper around `ansible-playbook` which handles environment setup. It creates a Python virtualenv, installs Ansible into it, uses `ansible-galaxy` to download the Ansible roles, and then runs `ansible-playbook`.

The `deploy` script expects a particular directory layout and must be run from such a directory. The `stack/ansible/example` directory is structured according to the expectations of the `deploy` script.

### Running the build

Choose a name for the cluster. For our purpose, it is `legbegbe`.

Copy the `stack/ansible/example` directory somewhere.

```
> cp -R stack/ansible/example ~/legbegbe
```

You can modify any file in the directory, but you should be able to build a cluster by only modifying `vars.yml` and setting the Ansible role versions in `requirements.yml`.

All of the identifiers in `vars.yml` except `image_owner` are fake, so at a minimum you must change `vpc_id`, `master_subnet_ids`, `node_subnet_ids`, `resource_bucket`, `kms_key_id`, `kms_key_alias`, and `keypair`. If you have copied the AMI to your own AWS account, you must change `image_owner` to your account number. Of course, you may modify any other values such as instance types. The example `vars.yml` is well commented to explain each variable.

In `requirements.yml`, the versions of the Ansible roles should match the Keights version you want to use. You can get the URLs for the roles for each version from the [GitHub release page](https://github.com/cloudboss/keights/releases).

```
- src: https://github.com/cloudboss/keights/releases/download/v1.13.2-1/keights-stack-v1.13.2-1.tar.gz
  name: keights-stack
- src: https://github.com/cloudboss/keights/releases/download/v1.13.2-1/keights-system-v1.13.2-1.tar.gz
  name: keights-system
```

Make sure your AWS credentials and region are in scope. For example, if you were using a credentials file with a profile called `cloudboss`, you'd set the following variables:

```
> export AWS_DEFAULT_REGION=us-east-1
> export AWS_PROFILE=cloudboss
```

The cluster name must be set as an environment variable called `CLUSTER`.

```
> export CLUSTER=legbegbe
```

Now run `deploy` from the directory you created:

```
> cd ~/legbegbe
> /path/to/deploy
```

The `deploy` scripts passes any additional arguments you give it to `ansible-playbook`, so for example you can increase Ansible's verbosity by passing `-vvv` to `deploy`.

In a few minutes, your cluster should be ready.

### Further customization

Check the [`keights-stack` README](https://github.com/cloudboss/keights/tree/master/stack/ansible/keights-stack) and the [`keights-system` README](https://github.com/cloudboss/keights/tree/master/stack/ansible/keights-system) for all of the available parameters that may be passed to the roles. Changing some of them may require modifying `playbook.yml`, whereas those that are already defined in `vars.yml` may be changed there.

## Accessing the cluster with kubectl

In the `stack` directory is a script called `kubesetup`. It sets up the file defined by the `KUBECONFIG` environment variable, defaulting to `~/.kube/config`. To use it, you need both kubectl and the AWS CLI to be installed and on your `PATH`, and the AWS region and credentials must be in scope.

```
> export AWS_DEFAULT_REGION=us-east-1
> export AWS_PROFILE=cloudboss
> export KUBECONFIG=~/.kube/legbegbe.kubeconfig
```

Run `kubesetup` with the cluster name as its only argument:

```
> ./kubesetup legbegbe
```

Now you can use kubectl to connect to the cluster.

```
> kubectl get no
NAME                            STATUS   ROLES    AGE   VERSION
ip-172-31-23-200.ec2.internal   Ready    node     8m    v1.13.2
ip-172-31-81-85.ec2.internal    Ready    node     9m    v1.13.2
ip-172-31-90-46.ec2.internal    Ready    master   17m   v1.13.2
```

## Accessing the cluster with ssh

You can ssh to machines in the cluster with the keypair assigned during installation, and username `boss`. The `boss` user has admin access with sudo.

## Keeping up to date

Keep your cluster up to date by modifying any of the files in your directory and rerunning `deploy`. For example, you may update the Kubernetes version by modifying the versions of the roles in `requirements.yml` when there is a new Keights release. Keights strives to be able to make this transition smoothly, and tests each build to be upgradeable from the previous Kubernetes minor release.

Note that some changes to parameters could result in changes which are not backwards compatible. You should *always* run such updates on an identically configured test cluster before running on a live system.

Upgrades currently happen as rolling updates to the autoscaling groups in the cluster. For the masters running in HA mode, this is usually seamless as they have high redundancy built in and only one master is replaced at a time. For application nodes, however, you may see some downtime during the upgrade as nodes are terminated. You can set the number of nodes to be replaced at a time by setting `update_max_batch_size` on each node group. The higher the number, the quicker the cluster will be upgraded, however you will lose more nodes at a time. Schedule your upgrades during a maintenance window where some downtime is acceptable, and experiment with `update_max_batch_size` to find the optimal setting.

In the future, Keights will support an A/B upgrade mode, where new node groups are built and running before old nodes are drained and then terminated. This should result in less or no downtime during upgrades.

It is highly recommended to check your cluster's Ansible directory into source control, and let a CI/CD tool run `deploy` when the source changes. The Ansible role is idempotent, and there should be no changes unless the source in the directory changes. It is important to keep all versions pinned in `requirements.txt` and `requirements.yml`, to avoid unforeseen side effects from changing versions of Ansible or its dependencies.

# CloudFormation without Ansible

At its core, Keights uses just plain CloudFormation. In the Keights git repository, the CloudFormation templates are kept under the `stack/cloudformation` directory, rather than within the Ansible `keights-stack` role, and they are packaged into the role during a release. Keeping them separate means you can integrate the CloudFormation templates into your own system, if you have already created something that is not compatible with the Ansible method. For example, you could import the templates into [AWS Service Catalog](https://docs.aws.amazon.com/servicecatalog/latest/adminguide/introduction.html) and use them from there.

## Lambdas

There are some Lambdas deployed as part of the CloudFormation stacks, and the zip files must exist in your S3 bucket under the paths expected by the templates. Ansible normally uploads them for you, but if you are "going it alone" without Ansible, you need to upload them yourself.

S3 paths within your bucket for each Lambda must be as follows:

* auto_namer - `stackbot/auto_namer/${KeightsVersion}/go1.x/auto_namer-${KeightsVersion}.zip`

* instattr - `stackbot/instattr/${KeightsVersion}/go1.x/instattr-${KeightsVersion}.zip`

* kube_ca - `stackbot/kube_ca/${KeightsVersion}/go1.x/kube_ca-${KeightsVersion}.zip`

* subnet_to_az - `stackbot/subnet_to_az/${KeightsVersion}/go1.x/subnet_to_az-${KeightsVersion}.zip`

The zip files for the Lambdas can be downloaded from the [GitHub releases page](https://github.com/cloudboss/keights/releases).

The bucket name will be passed to the CloudFormation stacks as the `ResourceBucket` parameter.

## CloudFormation stacks

There are three CloudFormation templates: `common.yml`, `master.yml`, and `node.yml`. The stacks must be deployed in that order.

### common.yml

This contains mostly security groups and IAM policies and roles for the cluster.

Some of the outputs of the stack will be needed to pass to `master.yml` and `node.yml`.

These are: `LoadBalancerSecurityGroup`, `MasterSecurityGroup`, `NodeSecurityGroup`, `MasterInstanceProfile`, `NodeInstanceProfile`, `LambdaRoleArn`, and `InstanceAttributeFunctionArn`.

### master.yml

This contains the master stack. It will require as inputs the `LoadBalancerSecurityGroup`, `MasterSecurityGroup`, `MasterInstanceProfile`, `LambdaRoleArn`, and `InstanceAttributeFunctionArn` outputs from the `common.yml` stack.

The `MasterSecurityGroup` output from `common.yml` should go into the `MasterSecurityGroups` parameter, though you may add additional groups to the list.

### node.yml

This contains the node stack. You can deploy as many of these as you like, giving each one its own parameters for the node labels. It will require as inputs the `NodeSecurityGroup`, `NodeInstanceProfile`, and `InstanceAttributeFunctionArn` outputs from the `common.yml` stack, and the `LoadBalancerDnsName` output from the `master.yml` stack.

The `NodeSecurityGroup` output from `common.yml` should go into the `NodeSecurityGroups` parameter, though you may add additional groups to the list.

# AMI

The `stack/ami/debian` directory of the git repository contains a [Packer](https://packer.io/) configuration for building the AMI. This is used to produce a minimal Debian image which has all dependencies preinstalled. If you want to build a custom AMI, you can start with this and modify it to change the disk partitioning or add your own packages.

If you want to use a Linux distribution other than Debian, keep in mind `keights` and `kubeadm` are started by systemd units that are configured in the CloudFormation templates, so it needs to be a systemd based distribution. As of `v1.13.2-2`, both `.dpkg` and `.rpm` packages are built for Keights releases, but only `.dpkg` is being tested on Debian.

Ubuntu versions of the AMI do exist, but are not presently published.

Features of the published Debian AMI are:

* It uses [debootstrap](https://wiki.debian.org/Debootstrap) to create a minimal base system.

* It is built with [merged usr](https://wiki.debian.org/UsrMerge), with `/usr` mounted read-only.

* It is partitioned from multiple EBS volumes, so your logs and other data don't fill your root filesystem. You can also [resize them dynamically](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-modify-volume.html) in a pinch, though you might want to build your own AMI if you need different partition sizes or a different layout.

* [Enhanced networking](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/enhanced-networking.html) is enabled for all instance types.

* It has automatic [NVME device name linking](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/nvme-ebs-volumes.html#identify-nvme-ebs-device) similar to Amazon Linux, so you don't get surprises from NVME devices changing names; you can keep using e.g. `/dev/xvdf`.

* It includes support for common filesystems for your [storage classes](https://kubernetes.io/docs/concepts/storage/storage-classes/): ext4, nfs, xfs, and btrfs.

# The keights binary

The `keights` binary is built into the AMI and is written in Go. The `keights` subdirectory of the git repository contains the code for building the binary, as well as the systemd unit files and Go templates that it uses.

The primary tool for per-instance configuration is kubeadm, but there are a few things it can't do, or that need to be done before it can run. The `keights` binary fills this need. It is a very minimal configuration management tool that does not require any dependencies. Nothing it does is specific to Kubernetes, and in fact it only does four things:

## signal

`keights signal` sends a signal to CloudFormation to let it know that the instance has successfully initialized. This is used by all machines when they first launch. It does the same thing as the `cfn-signal` command created by Amazon. However, `cfn-signal` is very old, unmaintained, and written in Python 2. The Keights AMI does not have or want Python 2, so this command was created instead.

## templatize

`keights templatize` expands [Go templates](https://golang.org/pkg/text/template/) and writes them to files. The `kubeadm init` and `kubeadm join` commands use config files for inputs, and these config files begin as Go templates which are expanded by the variables passed in user data via CloudFormation.

## volumize

`keights volumize` attaches an EBS volume to an EC2 instance and creates a filesystem on it if not present. This is used for etcd volumes on the masters. When masters are terminated, as happens during a rolling update, the etcd volume is detached. The replacement node will find the volume within its availability zone, and attach it to itself. For this reason, each master *must* run in a different availability zone.

## whisper

`keights whisper` retrieves encrypted [SSM parameters](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-paramstore.html) and writes them to files. This is used for retrieving the cluster CA certificates and kubelet bootstrap token, which are generated by a CloudFormation custom resource Lambda.
