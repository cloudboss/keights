# Copyright © 2026 Joseph Wright <joseph@cloudboss.co>
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.

locals {
  cluster_dns_ip = cidrhost(var.service_subnet, 10)
  etcd_prefix    = "keights-etcd-${var.cluster_name}"

  # Sort AZ names, otherwise the launch template may change on each run.
  azs_sorted = sort([
    for _, az in local.azs_autoscaling_group : az
  ])
  etcd_initial_cluster = join(",", [
    for az in local.azs_sorted :
    "${local.etcd_prefix}-${az}=https://${local.etcd_prefix}-${az}.${var.etcd_domain}:2380"
  ])

  kubeadm_init_initconfiguration = {
    apiVersion      = "kubeadm.k8s.io/v1beta4"
    bootstrapTokens = []
    kind            = "InitConfiguration"
    localAPIEndpoint = {
      advertiseAddress = "__IPV4_ADDRESS__"
      bindPort         = 6443
    }
    nodeRegistration = {
      criSocket       = "unix:///run/containerd/containerd.sock"
      imagePullPolicy = "IfNotPresent"
      imagePullSerial = false
      kubeletExtraArgs = [
        {
          name  = "cloud-provider"
          value = "external"
        },
        {
          name  = "image-credential-provider-bin-dir"
          value = "/usr/local/bin"
        },
        {
          name  = "image-credential-provider-config"
          value = "/etc/kubernetes/ecr-credential-provider.yaml"
        },
      ]
      taints = [
        {
          effect = "NoSchedule"
          key    = "node-role.kubernetes.io/control-plane"
        },
      ]
    }
    timeouts = {
      controlPlaneComponentHealthCheck = "4m0s"
      discovery                        = "5m0s"
      etcdAPICall                      = "2m0s"
      kubeletHealthCheck               = "4m0s"
      kubernetesAPICall                = "1m0s"
      tlsBootstrap                     = "5m0s"
      upgradeManifests                 = "5m0s"
    }
  }

  kubeadm_init_clusterconfiguration = {
    apiServer = {
      certSANs = [var.load_balancer.dns_name]
      extraArgs = [
        {
          name  = "authentication-token-webhook-config-file"
          value = "/etc/kubernetes/aws-iam-authenticator/aws-iam-authenticator.conf"
        },
        {
          name  = "external-hostname"
          value = var.load_balancer.dns_name
        },
        {
          name  = "service-account-jwks-uri"
          value = "https://kubernetes.default.svc.${var.cluster_domain}/openid/v1/jwks"
        },
      ]
      extraVolumes = [
        {
          name      = "aws-iam-authenticator-config"
          hostPath  = "/etc/kubernetes/aws-iam-authenticator"
          mountPath = "/etc/kubernetes/aws-iam-authenticator"
          readOnly  = true
          pathType  = "DirectoryOrCreate"
        }
      ]
    }
    apiVersion                  = "kubeadm.k8s.io/v1beta4"
    caCertificateValidityPeriod = "87600h0m0s"
    certificateValidityPeriod   = "8760h0m0s"
    certificatesDir             = "/etc/kubernetes/pki"
    clusterName                 = "kubernetes" # Internal, not to be confused with var.cluster_name.
    controllerManager = {
      extraArgs = [
        {
          name  = "cloud-provider"
          value = "external"
        },
      ]
    }
    controlPlaneEndpoint = "${var.load_balancer.dns_name}:443"
    dns                  = {}
    encryptionAlgorithm  = var.encryption_algorithm
    etcd = {
      local = {
        dataDir = "/var/lib/etcd"
        extraArgs = [
          {
            name  = "data-dir"
            value = "/var/lib/etcd"
          },
          {
            name  = "initial-cluster"
            value = local.etcd_initial_cluster
          },
          {
            name  = "initial-cluster-token"
            value = "${var.cluster_name}-${var.etcd_domain}"
          },
          {
            name  = "name"
            value = "${local.etcd_prefix}-__AVAILABILITY_ZONE__"
          },
        ]
        peerCertSANs = [
          "__IPV4_ADDRESS__",
          "${local.etcd_prefix}-__AVAILABILITY_ZONE__.${var.etcd_domain}",
        ]
        serverCertSANs = [
          "__IPV4_ADDRESS__",
          "${local.etcd_prefix}-__AVAILABILITY_ZONE__.${var.etcd_domain}",
        ]
      }
    }
    imageRepository   = var.image_registry
    kind              = "ClusterConfiguration"
    kubernetesVersion = var.kubernetes_version
    networking = {
      dnsDomain     = var.cluster_domain
      serviceSubnet = var.service_subnet
    }
    proxy     = {}
    scheduler = {}
  }

  kubeadm_init_kubeletconfiguration = {
    apiVersion = "kubelet.config.k8s.io/v1beta1"
    authentication = {
      anonymous = {
        enabled = false
      }
      webhook = {
        cacheTTL = "0s"
        enabled  = true
      }
      x509 = {
        clientCAFile = "/etc/kubernetes/pki/ca.crt"
      }
    }
    authorization = {
      mode = "Webhook"
      webhook = {
        cacheAuthorizedTTL   = "0s"
        cacheUnauthorizedTTL = "0s"
      }
    }
    cgroupDriver                     = "cgroupfs"
    clusterDNS                       = [local.cluster_dns_ip]
    clusterDomain                    = var.cluster_domain
    containerRuntimeEndpoint         = "unix:///run/containerd/containerd.sock"
    cpuManagerReconcilePeriod        = "0s"
    crashLoopBackOff                 = {}
    evictionPressureTransitionPeriod = "0s"
    fileCheckFrequency               = "0s"
    healthzBindAddress               = "127.0.0.1"
    healthzPort                      = 10248
    httpCheckFrequency               = "0s"
    imageMaximumGCAge                = "0s"
    imageMinimumGCAge                = "0s"
    kind                             = "KubeletConfiguration"
    logging = {
      flushFrequency = 0
      options = {
        json = {
          infoBufferSize = "0"
        }
        text = {
          infoBufferSize = "0"
        }
      }
      verbosity : 0
    }
    memorySwap = {
      swapBehavior = "NoSwap"
    }
    nodeStatusReportFrequency       = "0s"
    nodeStatusUpdateFrequency       = "0s"
    rotateCertificates              = true
    runtimeRequestTimeout           = "0s"
    shutdownGracePeriod             = "0s"
    shutdownGracePeriodCriticalPods = "0s"
    staticPodPath                   = "/etc/kubernetes/manifests"
    streamingConnectionIdleTimeout  = "0s"
    syncFrequency                   = "0s"
    volumeStatsAggPeriod            = "0s"
  }

  kubeadm_init_config = <<-EOS
    ${yamlencode(local.kubeadm_init_initconfiguration)}
    ---
    ${yamlencode(local.kubeadm_init_clusterconfiguration)}
    ---
    ${yamlencode(local.kubeadm_init_kubeletconfiguration)}
  EOS
}
