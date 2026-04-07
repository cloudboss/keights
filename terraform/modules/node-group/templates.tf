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

  kubelet_config = yamlencode({
    apiVersion               = "kubelet.config.k8s.io/v1beta1"
    kind                     = "KubeletConfiguration"
    cgroupDriver             = "cgroupfs"
    clusterDNS               = [local.cluster_dns_ip]
    clusterDomain            = var.cluster_domain
    containerRuntimeEndpoint = "unix:///run/containerd/containerd.sock"
    authentication = {
      anonymous = {
        enabled = false
      }
      x509 = {
        clientCAFile = "/etc/kubernetes/pki/ca.crt"
      }
      webhook = {
        enabled = true
      }
      cacheTTL = "0s"
    }
    authorization = {
      mode = "Webhook"
      webhook = {
        cacheAuthorizedTTL   = "0s"
        cacheUnauthorizedTTL = "0s"
      }
    }
    rotateCertificates = true
    healthzBindAddress = "127.0.0.1"
    healthzPort        = 10248
    memorySwap = {
      swapBehavior = "NoSwap"
    }
  })

  kubelet_bootstrap_kubeconfig = yamlencode({
    apiVersion = "v1"
    kind       = "Config"
    clusters = [
      {
        cluster = {
          server                = "https://${var.control_plane_endpoint}"
          certificate-authority = "/etc/kubernetes/pki/ca.crt"
        }
        name = "kubernetes"
      },
    ]
    contexts = [
      {
        context = {
          cluster = "kubernetes"
          user    = "aws"
        }
        name = "aws"
      },
    ]
    current-context = "aws"
    users = [
      {
        name = "aws"
        user = {
          exec = {
            apiVersion = "client.authentication.k8s.io/v1beta1"
            command    = "/usr/local/bin/aws-iam-authenticator"
            args = [
              "token",
              "-i",
              var.cluster_name,
            ]
          }
        }
      },
    ]
  })
}
