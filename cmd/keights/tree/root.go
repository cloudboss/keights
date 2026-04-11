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
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
)

var (
	ClusterName string
	Region      string

	RootCmd = &cobra.Command{
		Use:          "keights",
		Short:        "Self hosted Kubernetes on AWS",
		SilenceUsage: true,
	}
)

func init() {
	RootCmd.PersistentFlags().StringVar(
		&ClusterName, "cluster-name", "", "name of the cluster",
	)
	RootCmd.PersistentFlags().StringVar(
		&Region, "region", "", "AWS region (overrides SDK defaults)",
	)
	RootCmd.AddCommand(KubeconfigCmd)
	RootCmd.AddCommand(TokenCmd)
	RootCmd.AddCommand(VersionCmd)
}

func loadAWSConfig(ctx context.Context) (aws.Config, error) {
	var opts []func(*config.LoadOptions) error
	if Region != "" {
		opts = append(opts, config.WithRegion(Region))
	}
	return config.LoadDefaultConfig(ctx, opts...)
}
