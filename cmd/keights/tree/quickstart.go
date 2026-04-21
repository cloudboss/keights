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

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/cloudboss/keights/internal/deploy"
	"github.com/cloudboss/keights/internal/quickstart"
	"github.com/spf13/cobra"
)

var QuickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Guided wizard to deploy a keights cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		cfg, err := loadAWSConfig(ctx)
		if err != nil {
			return fmt.Errorf("unable to load AWS config: %w", err)
		}

		disc := quickstart.NewDiscoverer(ec2.NewFromConfig(cfg), kms.NewFromConfig(cfg))

		answers, err := quickstart.RunWizard(ctx, disc, quickstart.Defaults{
			Region:      cfg.Region,
			ClusterName: ClusterName,
		})
		if err != nil {
			return err
		}
		if answers == nil {
			fmt.Println("Cancelled.")
			return nil
		}

		quickstart.PrintSummary(cmd.OutOrStdout(), *answers)

		opts := quickstart.RenderOptions{
			ModuleSource: quickstart.ModuleSourceForVersion(Version),
		}
		if err := quickstart.Render(*answers, opts); err != nil {
			return fmt.Errorf("unable to render Terraform: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(),
			"Wrote Terraform configuration to %s\n", answers.OutputDir,
		)

		if answers.DeployNow {
			return deploy.Run(deploy.Options{
				Dir: answers.OutputDir,
			})
		}

		fmt.Fprintf(cmd.OutOrStdout(),
			"Next: keights deploy %s\n", answers.OutputDir,
		)
		return nil
	},
}
