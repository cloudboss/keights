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

var qsCfg quickstart.Config

var QuickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Quick deployment for a keights cluster",
	Long: `Quick deployment for a keights cluster.

With no arguments, you will be given guided prompts to configure a cluster.

If you run with --non-interactive, you must provide configuration via flags. Example:

  keights quickstart --non-interactive \
    --cluster-name bonito \
    --access-cidrs-api 10.0.0.0/8 \
    --region us-east-1 \
    --vpc-id vpc-12029787df8d47264 \
    --control-plane type=m5.large,subnets=subnet-0f69fbd6490cf812a \
    --node-group name=default,type=m5.large,min=1,desired=2,max=5,subnets=subnet-0f69fbd6490cf812a:subnet-07a12b20088c394cd`,

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		cfg, err := loadAWSConfig(ctx)
		if err != nil {
			return fmt.Errorf("unable to load AWS config: %w", err)
		}

		disc := quickstart.NewDiscoverer(
			ec2.NewFromConfig(cfg), kms.NewFromConfig(cfg),
		)

		var answers *quickstart.Answers
		if qsCfg.NonInteractive {
			qsCfg.ClusterName = ClusterName
			if qsCfg.Region == "" {
				qsCfg.Region = cfg.Region
			}
			a, err := quickstart.BuildAnswers(ctx, disc, qsCfg)
			if err != nil {
				return err
			}
			answers = a
		} else {
			a, err := quickstart.RunWizard(ctx, disc, quickstart.Defaults{
				Region:      cfg.Region,
				ClusterName: ClusterName,
			})
			if err != nil {
				return err
			}
			if a == nil {
				fmt.Println("Cancelled.")
				return nil
			}
			answers = a
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

func init() {
	f := QuickstartCmd.Flags()
	f.BoolVar(&qsCfg.NonInteractive, "non-interactive", false,
		"run without prompts, using flags for all configuration")
	f.StringVar(&qsCfg.VPCID, "vpc-id", "", "vpc id for the cluster")
	f.StringSliceVar(&qsCfg.AccessCIDRsAPI, "access-cidrs-api", []string{"0.0.0.0/0"},
		"cidrs that can reach the kubernetes api")
	f.StringSliceVar(&qsCfg.AccessCIDRsNodePorts, "access-cidrs-node-ports",
		nil, "cidrs that can reach node ports")
	f.StringSliceVar(&qsCfg.AccessCIDRsSSH, "access-cidrs-ssh", nil,
		"cidrs that can connect to instances over ssh")
	f.StringVar(&qsCfg.AMIName, "ami-name", "", "ami name (auto-discovered if omitted)")
	f.StringVar(&qsCfg.AMIOwnerID, "ami-owner-id", "",
		"ami owner id (auto-discovered if omitted)")
	f.StringVar(&qsCfg.KMSKeyID, "kms-key-id", "", "kms key id or alias (blank to create new)")
	f.StringVar(&qsCfg.SSHKeyPair, "ssh-key-pair", "", "ec2 key pair name for ssh access")
	f.BoolVar(&qsCfg.IRSAEnabled, "irsa", true,
		"enable iam roles for service accounts")
	f.StringVar(&qsCfg.ControlPlane, "control-plane", "",
		"control plane configuration. spec: type=X,subnets=s1:s2:s3")
	f.StringArrayVar(&qsCfg.NodeGroups, "node-group", nil,
		"node group configuration, repeat for multiple groups. spec: "+
			"name=X,type=Y,min=N,desired=N,max=N,subnets=s1:s2")
	f.StringVar(&qsCfg.StateBackend, "state-backend", "",
		"terraform state backend type (s3 or blank for local)")
	f.StringVar(&qsCfg.StateBucket, "state-bucket", "", "s3 bucket for terraform state")
	f.StringVar(&qsCfg.StateKey, "state-key", "", "s3 key for terraform state")
	f.StringVar(&qsCfg.StateRegion, "state-region", "", "s3 bucket region for terraform state")
	f.StringVar(&qsCfg.OutputDir, "output-dir", "",
		"directory to write Terraform configuration")
	f.BoolVar(&qsCfg.Deploy, "deploy", false,
		"run terraform apply after generating configuration")
}
