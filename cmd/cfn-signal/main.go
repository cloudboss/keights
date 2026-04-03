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

package main

import (
	"context"
	"fmt"

	cfnsignal "github.com/cloudboss/keights/internal/cfn-signal"
	"github.com/spf13/cobra"
)

var (
	stackName    string
	status       string
	resource     string
	cfnSignalCmd = &cobra.Command{
		Use:   "cfn-signal",
		Short: "Signal success or failure to CloudFormation stack",
		RunE: func(cmd *cobra.Command, args []string) error {
			if status != "SUCCESS" && status != "FAILURE" {
				return fmt.Errorf("status must be one of SUCCESS or FAILURE")
			}
			return cfnsignal.DoIt(context.Background(), stackName, status, resource)
		},
	}
)

func init() {
	cfnSignalCmd.Flags().StringVarP(&stackName, "stack-name", "n",
		"", "Name of CloudFormation stack to signal")
	cfnSignalCmd.Flags().StringVarP(&status, "status", "s",
		"", `Status to send, either "SUCCESS" or "FAILURE"`)
	cfnSignalCmd.Flags().StringVarP(&resource, "resource", "r",
		"AutoScalingGroup", "Resource in CloudFormation stack to signal")
}

func main() {
	cfnSignalCmd.Execute()
}
