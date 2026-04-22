// Copyright © 2026 Joseph Wright <joseph@cloudboss.co>
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

package tree

import (
	"fmt"
	"os"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cloudboss/keights/internal/nlb"
	"github.com/cloudboss/keights/internal/whisperer"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var kubeconfigOutput string

var KubeconfigCmd = &cobra.Command{
	Use:   "kubeconfig",
	Short: "Generate a kubeconfig for a keights cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		if ClusterName == "" {
			return fmt.Errorf("--cluster-name is required")
		}

		ctx := cmd.Context()

		cfg, err := loadAWSConfig(ctx)
		if err != nil {
			return fmt.Errorf("unable to load aws config: %w", err)
		}

		serverURL, err := nlb.FindClusterNLB(
			ctx, elbv2.NewFromConfig(cfg), ClusterName,
		)
		if err != nil {
			return fmt.Errorf("unable to find cluster api server: %w", err)
		}

		w := whisperer.NewSSMWhisperer(cfg)
		ssmPath := fmt.Sprintf("/keights/%s/cluster/ca.crt", ClusterName)
		caCert, err := w.GetParameter(ctx, ssmPath)
		if err != nil {
			return fmt.Errorf("unable to retrieve ca certificate: %w", err)
		}

		kc := buildKubeconfig(
			ClusterName, serverURL, []byte(*caCert), Region,
		)

		if kubeconfigOutput == "-" {
			out, err := clientcmd.Write(kc)
			if err != nil {
				return fmt.Errorf("unable to marshal kubeconfig: %w", err)
			}
			_, err = os.Stdout.Write(out)
			return err
		}

		return clientcmd.WriteToFile(kc, kubeconfigOutput)
	},
}

func init() {
	KubeconfigCmd.Flags().StringVarP(&kubeconfigOutput, "output", "o", "",
		`output file path, or "-" for stdout`)
	_ = KubeconfigCmd.MarkFlagRequired("output")
}

func buildKubeconfig(
	clusterName, serverURL string,
	caPEM []byte,
	region string,
) clientcmdapi.Config {
	tokenArgs := []string{"token", "--cluster-name", clusterName}
	if region != "" {
		tokenArgs = append(tokenArgs, "--region", region)
	}

	return clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			clusterName: {
				Server:                   fmt.Sprintf("https://%s", serverURL),
				CertificateAuthorityData: caPEM,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			clusterName: {
				Cluster:  clusterName,
				AuthInfo: clusterName,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			clusterName: {
				Exec: &clientcmdapi.ExecConfig{
					APIVersion: "client.authentication.k8s.io/v1beta1",
					Command:    "keights",
					Args:       tokenArgs,
				},
			},
		},
		CurrentContext: clusterName,
	}
}
