# keights-system

An Ansible role to set up kube-system extras on a Kubernetes cluster, including the network plugin and the dashboard. This is designed to work in conjunction with the `keights-stack` Ansible role.

# Requirements

A working Kubernetes cluster built according to the conventions of the `keights-stack` Ansible role.

Python dependencies are listed in `requirements.txt`.

# Role Variables

All role variables go under a top level dictionary `keights_system`.

`cluster_name`: (Required, type *string*) - Name of Kubernetes cluster.

`cluster_apiserver`: (Required, type *string*) - Hostname or IP address of Kubernetes APIserver, may use optional port.

`cert_manager`: (Optional, type *dict*) - A dictionary to configure cert-manager, see below.

`csi_driver`: (Optional, type *dict*) - A dictionary to configure the [AWS EBS CSI driver](https://github.com/kubernetes-sigs/aws-ebs-csi-driver) plugin, see below.

`kubernetes_dashboard_image`: (Optional, type *string*, default `kubernetesui/dashboard:v2.0.3`) - The Kubernetes dashboard container image.

`kubernetes_dashboard_metrics_image`: (Optional, type _string_, default `kubernetesui/metrics-scraper:v1.0.4`) - The metrics scraper image used by Kubernetes dashboard.

`network`: (Required, type *dict*) - A dictionary to configure the network plugin, see below.

### cert_manager

`enable`: (Optional, type *bool*, default `true`) - Whether or not to enable cert-manager. The CSI driver, if enabled with `keights_system.csi_driver.enable`, needs cert-manager to manage the certificate for the validation webhook.

`cainjector_image`: (Optional, default `quay.io/jetstack/cert-manager-cainjector:v1.7.1`) - The cert-manager cainjector image.

`cainjector_replicas`: (Optional, type *int*,  default `2`) - The number of cainjector replicas.

`controller_extra_annotations`: (Optional, type *dict*, default `{}`) - Extra annotations to assign to the cert-manager controller pods.

`controller_extra_args`: (Optional, type *list*, default `[]`) - Extra arguments to pass to the cert-manager controller pods.

`controller_image`: (Optional, default `quay.io/jetstack/cert-manager-controller:v1.7.1`) - The cert-manager controller image.

`controller_replicas`: (Optional, type *int*,  default `2`) - The number of controller replicas.

`namespace`: (Optional, default `kube-system`) - The namespace used for cert-manager.

`webhook_image`: (Optional, default `quay.io/jetstack/cert-manager-webhook:v1.7.1`) - The cert-manager webhook image.

`webhook_replicas`: (Optional, type *int*,  default `2`) - The number of webhook replicas.

### csi_driver

This configures the [AWS EBS CSI driver](https://github.com/kubernetes-sigs/aws-ebs-csi-driver) and associated resources, such as the snapshot controller and validation webhook.

`aws_ebs_csi_driver_affinity`: (Optional, type *dict*, default `{"nodeAffinity": {"requiredDuringSchedulingIgnoredDuringExecution": {"nodeSelectorTerms": [{"matchExpressions": [{"key": "node-role.kubernetes.io/control-plane", "operator": "Exists"}]}]}}}`) - Affinity for the AWS EBS CSI driver. The default value causes the driver to run on control plane nodes so it can use the permissions from the IAM instance profile. This can be disabled by setting the value to `{}` or `null`.

`aws_ebs_csi_driver_extra_annotations`: (Optional, type *dict*, default `{}`) - Annotations to add to the AWS EBS CSI driver, such as for assigning an IAM role.

`aws_ebs_csi_driver_image`: (Optional, type *string*, default `registry.k8s.io/provider-aws/aws-ebs-csi-driver:v1.4.0`) - The AWS EBS CSI driver container image.

`aws_ebs_csi_driver_replicas`: (Optional, type *int*,  default `2`) - The number of AWS EBS CSI driver replicas. Set to `1` if `aws_ebs_csi_driver_affinity` schedules the driver on the control plane and there is a single control plane node.

`csi_attacher_image`: (Optional, type *string*, default `registry.k8s.io/sig-storage/csi-attacher:v3.1.0`) - The CSI attacher container image.

`csi_node_driver_registrar_image` (Optional, type *string*, default `registry.k8s.io/sig-storage/csi-node-driver-registrar:v2.1.0`) - The CSI node driver registrar container image.

`csi_provisioner_image`: (Optional, type *string*, default `registry.k8s.io/sig-storage/csi-provisioner:v2.1.1`) - The CSI provisioner container image.

`csi_resizer` (Optional, type *string*, default `registry.k8s.io/sig-storage/csi-resizer:v1.1.0`) - The CSI resizer container image.

`csi_snapshotter_image`: (Optional, type *string*, default `registry.k8s.io/sig-storage/csi-snapshotter:v3.0.3`) - The CSI snapshotter container image.

`ebs_plugin_image`: (Optional, type *string*, default `registry.k8s.io/provider-aws/aws-ebs-csi-driver:v1.4.0`) - The EBS plugin container image.

`enable`: (Optional, type *bool*, default `true`) - Whether or not to enable the EBS CSI driver.

`livenessprobe_image`: (Optional, type *string*,  default `registry.k8s.io/sig-storage/livenessprobe:v2.2.0`) - The liveness probe container image.

`namespace`: (Optional, default `kube-system`) - The namespace used for csi driver.

`snapshot_controller_image`: (Optional, type *string*, default `registry.k8s.io/sig-storage/snapshot-controller:v5.0.0`) - The common snapshot controller container image.

`snapshot_controller_replicas`: (Optional, type *int*,  default `2`) - The number of snapshot controller replicas.

`snapshot_validation_image`: (Optional, type *string*, default `registry.k8s.io/sig-storage/snapshot-validation-webhook:v5.0.1`) - The snapshot validation webhook container image.

`snapshot_validation_replicas`: (Optional, type *int*,  default `2`) - The number of snapshot validation replicas.

### network

The `network` dictionary may contain the following keys:

`rr_node_label`: (Optional, type *string*, default `''`) - If defined, nodes with this label will be configured as route reflectors (see [here](https://projectcalico.docs.tigera.io/networking/bgp#configure-a-node-to-act-as-a-route-reflector) and [here](https://github.com/cloudnativelabs/kube-router/blob/master/docs/bgp.md#route-reflector-setup--without-full-mesh)). To set the label on nodes, define it within [`node_labels`](https://github.com/cloudboss/keights/tree/master/stack/ansible/keights-stack#node_groups) when creating a node group. The value of the label is not important, only the presence of the label. Caution: if `rr_node_label` is defined and no nodes have a matching label, there will be no networking!

`rr_cluster_id`: (Optional, type *string*, default `10.0.0.1` for calico, `42` for kube-router) - Route reflector cluster ID when `rr_node_label` is defined. Calico expects the cluster ID to be an IPv4 address, while kube-router expects an integer.

`plugin`: (Required, type *string*) - May be one of `calico`, `kube-router`.

If `plugin` is `calico`, you may set the following keys. These will have no effect if `plugin` is `kube-router`.

`pod_cidr`: (Required, type *string*) - Kubernetes cluster pod CIDR, which must match what was given to the `keights-stack` Ansible role.

`cni_image`: (Optional, type *string*, default `docker.io/calico/cni:v3.24.1`) - The CNI container image.

`calico_node_image`: (Optional, type *string*, default `docker.io/calico/node:v3.24.1`) - The Calico node container image.

`calico_ctl_image`: (Optional, type *string*, default `docker.io/calico/ctl:v3.15.5`) - The Calico ctl container image.

`kube_controllers_image`: (Optional, type *string*, default `docker.io/calico/kube-controllers:v3.24.1`) - The Calico kube controllers container image.

`typha_image`: (Optional, type *string*, default `docker.io/calico/typha:v3.24.1`) - The [Typha](https://github.com/projectcalico/typha) container image.

`typha_autoscaler_image` (Optional, type *string*, default `registry.k8s.io/cpa/cluster-proportional-autoscaler:1.8.6`) - The Typha autoscaler container image.

If `plugin` is `kube-router`, you may set the following keys. These will have no effect if `plugin` is `calico`.

`kube_router_image`: (Optional, type *string*, default `cloudnativelabs/kube-router:v1.5.1`) - The kube-router container image.

`busybox_image`: (Optional, type *string*, default `busybox:1.35.0`) - The busybox container image.

`kubectl_image`: (Optional, type *string*, default `bitnami/kubectl:1.24.4`) - The kubectl container image.

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
