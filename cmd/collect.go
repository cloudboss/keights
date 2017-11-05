// Copyright © 2017 Joseph Wright <joseph@cloudboss.co>
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
	"github.com/cloudboss/keights/pkg/collect"
	"github.com/spf13/cobra"
)

var (
	outputFile string
	collectCmd = &cobra.Command{
		Use:   "collect",
		Short: "Collect information about an autoscaling group",
		RunE: func(cmd *cobra.Command, args []string) error {
			return collect.DoIt(asgName, volumeTag, outputFile)
		},
	}
)

func init() {
	RootCmd.AddCommand(collectCmd)
	collectCmd.Flags().StringVarP(&asgName, "asg-name", "a",
		"", "Name of autoscaling group")
	collectCmd.Flags().StringVarP(&volumeTag, "volume-tag", "v",
		"", "Tag to search on EBS volume")
	collectCmd.Flags().StringVarP(&outputFile, "output-file", "o",
		"", "File in which to write output")
}
