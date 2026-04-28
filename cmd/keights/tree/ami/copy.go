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

package ami

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cloudboss/keights/internal/ami"
	"github.com/spf13/cobra"
)

func NewCopy(version string, loadAWSConfig loadAWSConfigFunc) *cobra.Command {
	var (
		sourceID string
		name     string
		wait     bool
	)
	cmd := &cobra.Command{
		Use:   "copy",
		Short: "Copy an official keights AMI from us-east-1 to another region",
		Long: `Copy an official keights AMI from us-east-1 to another region.

By default the latest official AMI matching the running keights minor
version is copied. Pass --ami-id to choose a specific source AMI.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			dstCfg, err := loadAWSConfig(ctx)
			if err != nil {
				return fmt.Errorf("unable to load AWS config: %w", err)
			}
			dstRegion := dstCfg.Region
			if dstRegion == "" {
				return fmt.Errorf(
					"no AWS region configured; set --region or AWS_REGION",
				)
			}

			srcCfg, err := config.LoadDefaultConfig(
				ctx, config.WithRegion(ami.OfficialRegion),
			)
			if err != nil {
				return fmt.Errorf(
					"unable to load AWS config for %s: %w",
					ami.OfficialRegion, err,
				)
			}
			srcClient := ec2.NewFromConfig(srcCfg)

			src, err := resolveCopySource(ctx, srcClient, version, sourceID)
			if err != nil {
				return err
			}

			dstClient := ec2.NewFromConfig(dstCfg)
			newID, err := ami.Copy(
				ctx, dstClient, src, ami.OfficialRegion, name, wait,
			)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if _, err := fmt.Fprintf(out, "Copied %s -> %s in %s\n",
				src.ID, newID, dstRegion); err != nil {
				return err
			}
			if !wait {
				if _, err := fmt.Fprintf(out,
					"The new AMI is pending. Check status with:\n"+
						"  keights ami status %s --region %s\n",
					newID, dstRegion,
				); err != nil {
					return err
				}
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&sourceID, "ami-id", "",
		"source AMI ID (default: latest matching official AMI)")
	f.StringVar(&name, "name", "",
		"name for the copied AMI (default: source AMI name)")
	f.BoolVar(&wait, "wait", false,
		"wait until the copied AMI is available")
	return cmd
}

func resolveCopySource(
	ctx context.Context, src ami.EC2API, version, amiID string,
) (ami.AMI, error) {
	if amiID != "" {
		a, err := ami.Get(ctx, src, ami.OfficialRegion, amiID)
		if err != nil {
			return ami.AMI{}, err
		}
		if a.OwnerID != ami.OfficialOwnerID {
			return ami.AMI{}, fmt.Errorf(
				"ami %s is not in the official keights account", amiID,
			)
		}
		return a, nil
	}
	candidates, err := ami.Find(
		ctx, src, ami.OfficialRegion, version,
		[]string{ami.OfficialOwnerID},
	)
	if err != nil {
		return ami.AMI{}, err
	}
	if len(candidates) == 0 {
		minor := ami.KeightsMinor(version)
		if minor == "" {
			return ami.AMI{}, fmt.Errorf(
				"no official keights AMIs found in %s", ami.OfficialRegion,
			)
		}
		return ami.AMI{}, fmt.Errorf(
			"no official keights AMIs found for %s in %s",
			minor, ami.OfficialRegion,
		)
	}
	return candidates[0], nil
}
