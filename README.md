# k8s-ami

Code for building a Kubernetes AMI.

The `debian` directory contains a [Packer](https://www.packer.io/) build definition based on [Debian Stretch](https://wiki.debian.org/DebianStretch).

# Requirements

This has been tested with Packer 1.1.0 and Ansible 2.4.1.

It is recommended to install Ansible into a [virtualenv](https://virtualenv.pypa.io/en/stable/).

First create a virtualenv if you do not already have one for this purpose:

```
virtualenv /path/to/virtualenv
```

Then install the dependencies into the virtualenv:

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

`ami-version`: The version of the AMI

`k8s-version`: The version of Kubernetes, from https://dl.k8s.io

`keights-version`: The version of keights, from https://github.com/cloudboss/keights

`docker-version`: The version of docker, from `apt.dockerproject.org`

The name of the AMI will be customized according to `ami-version`.

To build, run:

```
cd debian
packer build \
  -var docker-version=17.03.1~ce-0~debian-stretch \
  -var k8s-version=1.8.2 \
  -var keights-version=0.4.0 \
  -var ami-version=1710.2 \
  debian/build.json
```

License
-------

MIT

Author Information
------------------

Joseph Wright <joseph@cloudboss.co>
