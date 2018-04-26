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
	"fmt"
	"os"

	"github.com/cloudboss/keights/pkg/signal"
	"github.com/spf13/cobra"
)

var (
	stackName string
	status    string
	resource  string
	signalCmd = &cobra.Command{
		Use:   "signal",
		Short: "Signal success or failure to CloudFormation stack",
		RunE: func(cmd *cobra.Command, args []string) error {
			region := os.Getenv("AWS_REGION")
			if region == "" {
				return fmt.Errorf("AWS_REGION must be set")
			}
			if status != "SUCCESS" && status != "FAILURE" {
				return fmt.Errorf("status must be one of SUCCESS or FAILURE")
			}
			return signal.DoIt(stackName, status, resource)
		},
	}
)

func init() {
	RootCmd.AddCommand(signalCmd)
	signalCmd.Flags().StringVarP(&stackName, "stack-name", "n",
		"", "Name of CloudFormation stack to signal")
	signalCmd.Flags().StringVarP(&status, "status", "s",
		"", `Status to send, either "SUCCESS" or "FAILURE"`)
	signalCmd.Flags().StringVarP(&resource, "resource", "r",
		"AutoScalingGroup", "Resource in CloudFormation stack to signal")
}
