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
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/spf13/cobra"
)

func NewStatus(loadAWSConfig loadAWSConfigFunc) *cobra.Command {
	return &cobra.Command{
		Use:   "status AMI_ID",
		Short: "Show the state of an AMI and its underlying snapshots",
		Long: `Show the state of an AMI and the progress of any snapshots
backing it. Useful for following an in-progress copy started by
'keights ami copy' without --wait.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			amiID := args[0]
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
			out, err := client.DescribeImages(ctx, &ec2.DescribeImagesInput{
				ImageIds: []string{amiID},
			})
			if err != nil {
				return fmt.Errorf("unable to describe %s: %w", amiID, err)
			}
			if len(out.Images) == 0 {
				return fmt.Errorf("ami %s not found in %s", amiID, cfg.Region)
			}
			img := out.Images[0]

			w := cmd.OutOrStdout()
			lines := []string{
				fmt.Sprintf("AMI:    %s\n", deref(img.ImageId)),
				fmt.Sprintf("Name:   %s\n", deref(img.Name)),
				fmt.Sprintf("Region: %s\n", cfg.Region),
				fmt.Sprintf("State:  %s\n", string(img.State)),
			}
			if img.StateReason != nil && img.StateReason.Message != nil {
				lines = append(lines,
					fmt.Sprintf("Reason: %s\n", *img.StateReason.Message))
			}
			for _, line := range lines {
				if _, err := fmt.Fprint(w, line); err != nil {
					return err
				}
			}

			snapIDs := []string{}
			for _, m := range img.BlockDeviceMappings {
				if m.Ebs != nil && m.Ebs.SnapshotId != nil {
					snapIDs = append(snapIDs, *m.Ebs.SnapshotId)
				}
			}
			if len(snapIDs) == 0 {
				return nil
			}

			snaps, err := client.DescribeSnapshots(
				ctx, &ec2.DescribeSnapshotsInput{SnapshotIds: snapIDs},
			)
			if err != nil {
				return fmt.Errorf("unable to describe snapshots: %w", err)
			}
			if _, err := fmt.Fprintln(w, "Snapshots:"); err != nil {
				return err
			}
			for _, s := range snaps.Snapshots {
				progress := deref(s.Progress)
				if progress == "" {
					progress = "-"
				}
				if _, err := fmt.Fprintf(w, "  %s  %s  %s\n",
					deref(s.SnapshotId), string(s.State), progress,
				); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
