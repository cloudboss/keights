k8s-ami-core
=========

A role to install Kubernetes components on an EC2 instance, for the purpose of creating an AMI.

Requirements
------------

CentOS 7.x or Redhat Enterprise Linux 7.x base image.

Role Variables
--------------

```
# All variables are contained in a top level dictionary called `k8s_ami_core`.
k8s_ami_core:

  # Version of Kubernetes components to install
  k8s_version: 1.7.0
```

Dependencies
------------


Example Playbook
----------------

```
- hosts: build
  roles:
     - role: cloudboss.k8s-ami-core
       k8s_ami_core:
         k8s_version: 1.7.0
```

License
-------

MIT

Author Information
------------------

Joseph Wright <joseph@cloudboss.co>
