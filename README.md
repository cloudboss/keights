# keights

[![Build Status](https://travis-ci.org/cloudboss/keights.svg?branch=master)](https://travis-ci.org/cloudboss/keights)

CloudFormation based automation for Kubernetes.

# About

Keights has the following features:

#### Deploys in AWS using only CloudFormation and a purpose built AMI.

You don't need anything except CloudFormation to get started. The only things to download are three CloudFormation templates.

Though not required, an Ansible playbook is offered to deploy everything in one shot. An additional script is provided which automatically configures your kubeconfig.

One caveat is that if you are not deploying in `us-east-1`, you will have to copy some CloudFormation custom Lambdas from the `cloudboss-public` S3 bucket to a bucket in your region, as Lambdas won't deploy from a bucket outside of their region.

#### HA out of the box.

Keights deploys with either one or three masters. A master can be terminated and the etcd storage volume will be reattached to its replacement, allowing for rolling updates of the masters using a CloudFormation stack update.

#### Stays as vanilla as possible

[`kubeadm`](https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/) is used to handle most of the configuration. A command line utility handles a few extra tasks which are started as systemd services.

The addon manager is also run as a static pod on the masters, which is used to deploy the dashboard and the network, currently [kube-router](https://github.com/cloudnativelabs/kube-router).

#### Does not manage your VPC

If you are like many companies, you don't want Kubernetes automation to build your VPC. You already have standards and policies to enforce how those are built. Keights leaves it up to you to manage your own network.

If you want Kubernetes to manage load balancers, you will need to tag subnets to let it know which ones to use, but again that is left up to you.

#### Does not require an internet gateway in your VPC

Keights assumes you set up your VPC the way you want it, and will not throw errors if it does not find an internet gateway. You know how to get the traffic where you want it to go, whether that means using DirectConnect, a VPN, a proxy, or some other means, and keights isn't going to stand in your way.

#### Is modifiable

If keights doesn't quite work for you, you can modify the CloudFormation templates or build your own AMI. It doesn't try to be all things to all people, but it does try to accomodate by not being a black box.

# Usage

> Note: in this guide, all commands to be run from the shell are shown preceded by `> ` to indicate the shell prompt.

Keights has a git branch for each supported version of Kubernetes. For example, for Kubernetes `1.10.x`, the branch is `v1.10`. Releases are numbered according to the version of Kubernetes with an incrementing suffix appended for each release supporting that Kubernetes version. For Kubernetes `1.10.0`, this would be `v1.10.0-1` or `v1.10.0-2`, for example.

The `master` branch follows the latest release branch, so if you are deploying the latest release, proceed. Otherwise, do a git checkout of the release branch or desired release tag.

```
> git checkout v1.9
```

## Deploy with plain ol' CloudFormation

The `stack/cloudformation` directory contains three CloudFormation templates: `common.yml`, `master.yml`, and `node.yml`. The stacks must be deployed in that order.

### common.yml

This contains security groups and IAM roles for the cluster.

Some of the outputs of the stack will be needed to pass to `master.yml` and `node.yml`.

These are: `LoadBalancerSecurityGroup`, `MasterSecurityGroup`, `NodeSecurityGroup`, `MasterInstanceProfile`, `NodeInstanceProfile`, and `LambdaRoleArn`.

### master.yml

This contains the master stack. It will require as inputs the `LoadBalancerSecurityGroup`, `MasterSecurityGroup`, `MasterInstanceProfile`, and `LambdaRoleArn` outputs from the `common.yml` stack.

The `MasterSecurityGroup` output from `common.yml` should go into the `MasterSecurityGroups` parameter, though you may add additional groups to the list.

### node.yml

This contains the node stack. You can deploy as many of these as you like, giving each one its own parameters for the node labels. It will require as inputs the `NodeSecurityGroup` and `NodeInstanceProfile` outputs from the `common.yml` stack, and the `LoadBalancerDnsName` output from the `master.yml` stack.

The `NodeSecurityGroup` output from `common.yml` should go into the `NodeSecurityGroups` parameter, though you may add additional groups to the list.

## Deploy with Ansible

The `stack/ansible` directory contains an Ansible playbook which deploys all CloudFormation stacks together, passing the outputs of one stack as inputs to another as needed.

It should be called with the `deploy` script in the same directory, which installs Ansible in a virtualenv and does a bit of sanity checking before running the playbook.

In `stack/ansible/vars`, there is an example file, `otto.yml`, containing the variables for one cluster. Each cluster deployed by Ansible will require such a file. The file is documented with all of the available options. If it isn't flexible enough, you can modify `stack/ansible/playbook.yml`.

Choose a name for the cluster. For our purpose, it is `legbegbe`.

Make a copy of `otto.yml` and name it according to the cluster name, i.e. `legbegbe.yml`. All of the identifiers in `otto.yml` except for the AMI ID are fake, so edit the file to include a real VPC ID, subnet IDs, and so on. The AMI ID is the ID of the build for that branch in `us-east-1`. It is a public AMI, so you can copy it to your own account and region if needed.

You need either Python 3, or Python 2 with the `virtualenv` command, installed and on your `PATH`. Python 3 now includes virtual environments using the `venv` standard library module.

You also need AWS credentials and region to be in scope. You can do this with environment variables or a credentials file.

The cluster name must be set as an environment variable called `CLUSTER`.

Now run `deploy`:

```
> export AWS_DEFAULT_REGION=us-east-1
> export AWS_PROFILE=keights
> export CLUSTER=legbegbe
> ./deploy
```

The `deploy` scripts passes any additional arguments on to `ansible-playbook`, so for example you can increase the logging by passing `-vvv` to `deploy`.

In a few minutes, your cluster should be ready.

## Connecting to the cluster

In the `stack` directory is a script called `kubesetup`. It sets up `~/.kube/config` for your cluster. To use it, you need both kubectl and the AWS CLI to be installed and on your `PATH`, and the AWS region and credentials must be in scope.

Run `kubesetup` with the cluster name as its only argument:

```
> export AWS_DEFAULT_REGION=us-east-1
> export AWS_PROFILE=keights

> ./kubesetup legbegbe
```

Now you can use kubectl to connect to the cluster.

```
> kubectl get no
```

It is normal to get connection errors at first, give it a few minutes to come up.

# AMI

A Debian AMI is created for each release containing the installed keights package and all dependencies, including required docker images.

The software should in theory run on any systemd based Linux distribution, such as RHEL, though it is currently only tested on Debian.

Keights may be deployed using plain CloudFormation templates located in `stack/cloudformation`.
