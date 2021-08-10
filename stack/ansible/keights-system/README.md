# keights-system

An Ansible role to set up kube-system extras on a Kubernetes cluster, including the network plugin and the dashboard. This is designed to work in conjunction with the `keights-stack` Ansible role.

# Requirements

A working Kubernetes cluster built according to the conventions of the `keights-stack` Ansible role.

Python dependencies are listed in `requirements.txt`.

# Role Variables

All role variables go under a top level dictionary `keights_system`.

`cluster_name`: (Required, type *string*) - Name of Kubernetes cluster.

`cluster_apiserver`: (Required, type *string*) - Hostname or IP address of Kubernetes APIserver, may use optional port.

`kubernetes_dashboard_image`: (Optional, type *string*, default `kubernetesui/dashboard:v2.0.3`) - The Kubernetes dashboard container image.

`kubernetes_dashboard_metrics_image`: (Optional, type _string_, default `kubernetesui/metrics-scraper:v1.0.4`) - The metrics scraper image used by Kubernetes dashboard.

`network`: (Required, type *dict*) - A dictionary to configure the network plugin, see below.

### network

The `network` dictionary may contain the following keys:

`rr_node_label`: (Optional, type *string*, default `''`) - If defined, nodes with this label will be configured as route reflectors (see [here](https://docs.projectcalico.org/v3.9/networking/routereflector) and [here](https://github.com/cloudnativelabs/kube-router/blob/master/docs/bgp.md#route-reflector-setup--without-full-mesh)). To set the label on nodes, define it within [`node_labels`](https://github.com/cloudboss/keights/tree/master/stack/ansible/keights-stack#node_groups) when creating a node group. The value of the label is not important, only the presence of the label. Caution: if `rr_node_label` is defined and no nodes have a matching label, there will be no networking!

`rr_cluster_id`: (Optional, type *string*, default `10.0.0.1` for calico, `42` for kube-router) - Route reflector cluster ID when `rr_node_label` is defined. Calico expects the cluster ID to be an IPv4 address, while kube-router expects an integer.

`plugin`: (Required, type *string*) - May be one of `calico`, `kube-router`.

If `plugin` is `calico`, you may set the following keys. These will have no effect if `plugin` is `kube-router`.

`pod_cidr`: (Required, type *string*) - Kubernetes cluster pod CIDR, which must match what was given to the `keights-stack` Ansible role.

`cni_image`: (Optional, type *string*, default `docker.io/calico/cni:v3.20.0`) - The CNI container image.

`calico_node_image`: (Optional, type *string*, default `docker.io/calico/node:v3.20.0`) - The Calico node container image.

`calico_ctl_image`: (Optional, type *string*, default `docker.io/calico/ctl:v3.15.5`) - The Calico ctl container image.

`pod2daemon_flexvol_image`: (Optional, type *string*, default `docker.io/calico/pod2daemon-flexvol:v3.20.0`) - The Calico flex volume driver container image.

`kube_controllers_image`: (Optional, type *string*, default `docker.io/calico/kube-controllers:v3.20.0`) - The Calico kube controllers container image.

`typha_image`: (Optional, type *string*, default `docker.io/calico/typha:v3.20.0`) - The [Typha](https://github.com/projectcalico/typha) container image.

`typha_autoscaler_image` (Optional, type *string*, default `k8s.gcr.io/cpa/cluster-proportional-autoscaler:1.8.4`) - The Typha autoscaler container image.

If `plugin` is `kube-router`, you may set the following keys. These will have no effect if `plugin` is `calico`.

`kube_router_image`: (Optional, type *string*, default `cloudnativelabs/kube-router:v1.2.1`) - The kube-router container image.

`busybox_image`: (Optional, type *string*, default `busybox:1.30.1`) - The busybox container image.

`kubectl_image`: (Optional, type *string*, default `bitnami/kubectl:1.18.8`) - The kubectl container image.

`replace_kube_proxy`: (Optional, type *bool*, default `false`) - Whether or not kube-router should replace kube-proxy. If `true`, this requires setting `keights_stack.enable_kube_proxy` to `false` in the `keights-stack` Ansible role.

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
