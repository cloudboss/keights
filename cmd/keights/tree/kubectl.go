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
	"errors"
	"fmt"
	"os"
	"os/exec"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cloudboss/keights/internal/deps"
	"github.com/cloudboss/keights/internal/nlb"
	"github.com/cloudboss/keights/internal/whisperer"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

var KubectlCmd = &cobra.Command{
	Use:   "kubectl [-- kubectl-args...]",
	Short: "Run kubectl against a keights cluster",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		kubectlPath, err := deps.Ensure(deps.Kubectl)
		if err != nil {
			return fmt.Errorf("unable to cache kubectl: %w", err)
		}

		if ClusterName != "" {
			kubeconfigPath, err := generateTempKubeconfig()
			if err != nil {
				return err
			}
			defer os.Remove(kubeconfigPath)
			os.Setenv("KUBECONFIG", kubeconfigPath)
		}

		kc := exec.Command(kubectlPath, args...)
		kc.Stdin = os.Stdin
		kc.Stdout = os.Stdout
		kc.Stderr = os.Stderr

		if err := kc.Run(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				os.Exit(exitErr.ExitCode())
			}
			return fmt.Errorf("kubectl failed: %w", err)
		}
		return nil
	},
}

func generateTempKubeconfig() (string, error) {
	ctx := RootCmd.Context()

	cfg, err := loadAWSConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to load aws config: %w", err)
	}

	serverURL, err := nlb.FindClusterNLB(
		ctx, elbv2.NewFromConfig(cfg), ClusterName,
	)
	if err != nil {
		return "", fmt.Errorf("unable to find cluster api server: %w", err)
	}

	w := whisperer.NewSSMWhisperer(cfg)
	ssmPath := fmt.Sprintf("/keights/%s/cluster/ca.crt", ClusterName)
	caCert, err := w.GetParameter(ctx, ssmPath)
	if err != nil {
		return "", fmt.Errorf("unable to retrieve ca certificate: %w", err)
	}

	kc := buildKubeconfig(ClusterName, serverURL, []byte(*caCert), Region)

	tmpFile, err := os.CreateTemp("", "keights-kubeconfig-*.yaml")
	if err != nil {
		return "", fmt.Errorf("unable to create temp kubeconfig: %w", err)
	}
	defer tmpFile.Close()

	if err := clientcmd.WriteToFile(kc, tmpFile.Name()); err != nil {
		return "", fmt.Errorf("unable to write kubeconfig: %w", err)
	}

	return tmpFile.Name(), nil
}
