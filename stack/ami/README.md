# k8s-ami

Code for building a Kubernetes AMI.

The `debian` directory contains a [Packer](https://www.packer.io/) build definition based on [Debian Buster](https://wiki.debian.org/DebianBuster).

# Requirements

This has been tested with Packer 1.4.4.

It is recommended to install Ansible into a [virtualenv](https://virtualenv.pypa.io/en/stable/).

First create a virtualenv if you do not already have one for this purpose:

```
python3 -m venv /path/to/virtualenv
```

Then install the dependencies (including Ansible) into the virtualenv:

```
/path/to/virtualenv/bin/pip install -r requirements.txt
```

# Usage

Activate the virtualenv (if using) so Packer can find the dependencies:

```
. /path/to/virtualenv/bin/activate
```

Set the AWS region and credentials in your environment.

```
export AWS_REGION=us-east-1
export AWS_PROFILE=cloudboss-corp
```

Pass the following variables to packer:

`k8s-version`: The version of Kubernetes, from https://dl.k8s.io.

`keights-version`: The version of keights, from https://github.com/cloudboss/keights.

`vpc-id`: The ID of the VPC in which the build instance will be created.

`subnet-id`: The VPC subnet in which the build instance will be created.

See `debian/build.json` for other variables that can be set.

To build, run:

```
cd debian
packer build \
  -var k8s-version=1.8.2 \
  -var keights-version=0.4.0 \
  -var vpc-id=vpc-c71371be \
  -var subnet-id=subnet-37ef771b \
  debian/build.json
```

License
-------

MIT

Author Information
------------------

Joseph Wright <joseph@cloudboss.co>
