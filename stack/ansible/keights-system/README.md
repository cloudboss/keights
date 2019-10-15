# keights-system

An Ansible role to set up kube-system extras on a Kubernetes cluster, including the network plugin and the dashboard. This is designed to work in conjunction with the `keights-stack` Ansible role.

# Requirements

A working Kubernetes cluster built according to the conventions of the `keights-stack` Ansible role.

Python dependencies are listed in `requirements.txt`.

# Role Variables

All role variables go under a top level dictionary `keights_system`.

`cluster_name`: (Required, type *string*) - Name of Kubernetes cluster.

`cluster_apiserver`: (Required, type *string*) - Hostname or IP address of Kubernetes APIserver, may use optional port.

`kubernetes_dashboard_image`: (Optional, type *string*, default `k8s.gcr.io/kubernetes-dashboard-amd64:v1.10.1`) - The Kubernetes dashboard docker image.

`network`: (Required, type *dict*) - A dictionary to configure the network plugin, see below.

### network

The `network` dictionary may contain the following keys:

`plugin`: (Required, type *string*) - May be one of `calico`, `kube-router`.

If `plugin` is `calico`, you may set the following keys. These will have no effect if `plugin` is `kube-router`.

`pod_cidr`: (Required, type *string*) - Kubernetes cluster pod CIDR, which must match what was given to the `keights-stack` Ansible role.

`cni_image`: (Optional, type *string*, default `calico/cni:v3.9.1`) - The CNI docker image.

`calico_node_image`: (Optional, type *string*, default `calico/node:v3.9.1`) - The Calico node docker image.

`pod2daemon_flexvol_image`: (Optional, type *string*, default `calico/pod2daemon-flexvol:v3.9.1`) - The Calico flex volume driver docker image.

`kube_controllers_image`: (Optional, type *string*, default `calico/kube-controllers:v3.9.1`) - The Calico kube controllers docker image.

`typha_image`: (Optional, type *string*, default `calico/typha:v3.9.1`) - The [Typha](https://github.com/projectcalico/typha) docker image.

`typha_autoscaler_image` (Optional, type *string*, default `k8s.gcr.io/cluster-proportional-autoscaler-amd64:1.7.1`) - The Typha autoscaler docker image.

If `plugin` is `kube-router`, you may set the following keys. These will have no effect if `plugin` is `calico`.

`kube_router_image`: (Optional, type *string*, default `cloudnativelabs/kube-router:v0.3.2`) - The kube-router docker image.

`busybox_image`: (Optional, type *string*, default `busybox:1.30.1`) - The busybox docker image.

# Example Playbook

```
- hosts: localhost
  connection: local
  vars:
    cluster_name: cb
    vpc_id: vpc-ba92ad08
    pod_cidr: 10.0.0.0/16
    # ... other variables here ...

  roles:
  # First build cluster using keights-stack role
  - role: keights-stack
    keights_stack:
      cluster_name: '{{ cluster_name }}'
      vpc_id: '{{ vpc_id }}'
      # ... other variables here ...

  - role: keights-system
    keights_system:
      cluster_name: '{{ cluster_name }}'
      # master_stack is defined in keights-stack role, used for outputs here
      cluster_apiserver: '{{ master_stack.stack_outputs.LoadBalancerDnsName }}'
      network:
        plugin: calico
        pod_cidr: '{{ pod_cidr }}'
        typha_replicas: 2
```

# License

MIT

# Author Information

Joseph Wright <joseph@cloudboss.co>
