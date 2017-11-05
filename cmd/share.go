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
	"github.com/cloudboss/keights/pkg/share"
	"github.com/spf13/cobra"
)

var (
	inputFile string
	hostsFile string
	prefix    string
	domain    string
	shareCmd  = &cobra.Command{
		Use:   "share",
		Short: "Share hosts within autoscaling group using hosts file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return share.DoIt(inputFile, hostsFile, prefix, domain)
		},
	}
)

func init() {
	RootCmd.AddCommand(shareCmd)
	shareCmd.Flags().StringVarP(&inputFile, "input-file", "i",
		"", "Input file containing host IPs and indexes")
	shareCmd.Flags().StringVarP(&hostsFile, "volume-tag", "H",
		"/etc/hosts", "Hosts file")
	shareCmd.Flags().StringVarP(&prefix, "prefix", "p",
		"", "Hostname prefix")
	shareCmd.Flags().StringVarP(&domain, "domain", "d",
		"local", "Domain to give hosts")
}
