// Copyright © 2018 Joseph Wright <joseph@cloudboss.co>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"github.com/cloudboss/keights/pkg/kubeconfig"
	"github.com/spf13/cobra"
)

var (
	certDir        string
	serverURL      string
	clusterName    string
	bootstrapToken string
	bootstrapPath  string
	kubeRouterPath string
	kubeconfigCmd  = &cobra.Command{
		Use:   "kubeconfig",
		Short: "Create kubeconfigs for cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return kubeconfig.DoIt(certDir, serverURL, clusterName, bootstrapToken, bootstrapPath, kubeRouterPath)
		},
	}
)

func init() {
	RootCmd.AddCommand(kubeconfigCmd)
	kubeconfigCmd.Flags().StringVarP(&certDir, "certDir", "C",
		"/run/kubernetes/pki", "Directory to find certificates")
	kubeconfigCmd.Flags().StringVarP(&serverURL, "serverURL", "s",
		"", "URL of apiserver")
	kubeconfigCmd.Flags().StringVarP(&clusterName, "cluster-name", "c",
		"", "Name of Kubernetes cluster")
	kubeconfigCmd.Flags().StringVarP(&bootstrapToken, "bootstrap-token", "t",
		"/run/kubernetes/bootstrap-token", "Path to bootstrap token file")
	kubeconfigCmd.Flags().StringVarP(&bootstrapPath, "bootstrap-path", "b",
		"/run/kubernetes/bootstrap.kubeconfig", "Path to bootstrap kubeconfig file")
	kubeconfigCmd.Flags().StringVarP(&kubeRouterPath, "kube-router-path", "r",
		"/var/lib/kube-router/kubeconfig", "Path to kube-router kubeconfig file")
}
