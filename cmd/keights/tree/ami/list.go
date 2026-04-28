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
	"io"
	"sort"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cloudboss/keights/internal/ami"
	"github.com/spf13/cobra"
)

// loadAWSConfigFunc matches the signature of tree.loadAWSConfig and is used
// by both NewList and NewCopy to defer AWS config loading to the parent CLI.
type loadAWSConfigFunc = func(ctx context.Context) (aws.Config, error)

func NewList(version string, loadAWSConfig loadAWSConfigFunc) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available keights AMIs",
		Long: `List available keights AMIs.

Queries the configured region for keights AMIs that are owned by the caller's
account, and the official keights account in us-east-1. Results show only
AMIs that are compatible with the running keights version.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			cfg, err := loadAWSConfig(ctx)
			if err != nil {
				return fmt.Errorf("unable to load AWS config: %w", err)
			}
			userRegion := cfg.Region
			if userRegion == "" {
				return fmt.Errorf(
					"no AWS region configured; set --region or AWS_REGION",
				)
			}

			userClient := ec2.NewFromConfig(cfg)
			amis, err := ami.Find(
				ctx, userClient, userRegion, version,
				[]string{"self", ami.OfficialOwnerID},
			)
			if err != nil {
				return err
			}

			// Names of AMIs the caller already owns in this region.
			// Copy preserves the source AMI's Name by default, so a name
			// match against an official AMI means "already copied".
			ownNames := map[string]bool{}
			for _, a := range amis {
				if a.OwnerID != ami.OfficialOwnerID {
					ownNames[a.Name] = true
				}
			}

			uncopied := 0
			if userRegion != ami.OfficialRegion {
				officialCfg, err := config.LoadDefaultConfig(
					ctx, config.WithRegion(ami.OfficialRegion),
				)
				if err != nil {
					return fmt.Errorf(
						"unable to load AWS config for %s: %w",
						ami.OfficialRegion, err,
					)
				}
				officialClient := ec2.NewFromConfig(officialCfg)
				official, err := ami.Find(
					ctx, officialClient, ami.OfficialRegion, version,
					[]string{ami.OfficialOwnerID},
				)
				if err != nil {
					return err
				}
				for _, a := range official {
					if ownNames[a.Name] {
						continue
					}
					amis = append(amis, a)
					uncopied++
				}
			}

			sort.SliceStable(amis, func(i, j int) bool {
				return amis[i].CreationDate > amis[j].CreationDate
			})

			out := cmd.OutOrStdout()
			if len(amis) == 0 {
				minor := ami.KeightsMinor(version)
				if minor == "" {
					_, err := fmt.Fprintln(out, "No keights AMIs found.")
					return err
				}
				_, err := fmt.Fprintf(out, "No keights AMIs found for %s.\n", minor)
				return err
			}

			if err := renderTable(out, amis); err != nil {
				return err
			}

			if uncopied > 0 {
				if _, err := fmt.Fprintf(out,
					"\nTo copy an official AMI to %s, run:\n", userRegion,
				); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(out,
					"  keights ami copy --region %s\n", userRegion); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func renderTable(out io.Writer, amis []ami.AMI) error {
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(
		tw, "SOURCE\tREGION\tAMI ID\tNAME\tKUBERNETES\tCREATED",
	); err != nil {
		return err
	}
	for _, a := range amis {
		source := a.OwnerID + " (self)"
		if a.OwnerID == ami.OfficialOwnerID {
			source = a.OwnerID + " (official)"
		}
		k8s := a.KubernetesVersion
		if k8s == "" {
			k8s = "-"
		}
		if _, err := fmt.Fprintf(
			tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			source, a.Region, a.ID, a.Name, k8s, a.CreationDate,
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}
