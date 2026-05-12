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

package account

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cloudboss/keights/internal/validate"
	"github.com/spf13/cobra"
)

type loadAWSConfigFunc = func(ctx context.Context) (aws.Config, error)

func NewValidate(version string, loadAWSConfig loadAWSConfigFunc) *cobra.Command {
	var (
		vpcID     string
		subnetIDs []string
	)
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate that the AWS account is set up for keights",
		Long: `Run preflight checks against the AWS account.

Checks that the selected VPC has DNS support and DNS hostnames enabled,
that any --subnet-id values belong to the VPC, and that a keights AMI is
visible in the configured region. Exits non-zero on any failure, with a
remediation hint per check.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			cfg, err := loadAWSConfig(ctx)
			if err != nil {
				return fmt.Errorf("unable to load AWS config: %w", err)
			}
			if cfg.Region == "" {
				return fmt.Errorf(
					"no AWS region configured; set --region or AWS_REGION",
				)
			}

			client := ec2.NewFromConfig(cfg)
			vpc, err := validate.CheckVPC(ctx, client, vpcID)
			if err != nil {
				return err
			}
			results := []validate.Result{vpc}
			if vpc.OK {
				dns, err := validate.CheckVPCDNS(ctx, client, cfg.Region, vpcID)
				if err != nil {
					return err
				}
				results = append(results, dns)
				if len(subnetIDs) > 0 {
					subnets, err := validate.CheckSubnets(ctx, client, vpcID, subnetIDs)
					if err != nil {
						return err
					}
					results = append(results, subnets)
				}
			}
			amiResult, err := validate.CheckAMI(ctx, client, cfg.Region, version)
			if err != nil {
				return err
			}
			results = append(results, amiResult)

			out := cmd.OutOrStdout()
			if validate.PrintResults(out, results) {
				return nil
			}
			return fmt.Errorf("validation failed")
		},
	}
	cmd.Flags().StringVar(&vpcID, "vpc-id", "", "vpc id to validate")
	cmd.Flags().StringSliceVar(&subnetIDs, "subnet-id", nil,
		"subnet id to validate, repeatable")
	_ = cmd.MarkFlagRequired("vpc-id")
	return cmd
}
