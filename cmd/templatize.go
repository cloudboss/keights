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
	"github.com/cloudboss/keights/pkg/templatize"
	"github.com/spf13/cobra"
)

var (
	templateFile  string
	ip            string
	dest          string
	owner         string
	group         string
	mode          int
	vars          []string
	templatizeCmd = &cobra.Command{
		Use:   "template",
		Short: "Process file templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return templatize.DoIt(templateFile, inputFile, ip, dest, owner, group, mode, vars)
		},
	}
)

func init() {
	RootCmd.AddCommand(templatizeCmd)
	templatizeCmd.Flags().StringVarP(&templateFile, "template-file", "t",
		"", "Template file to be expanded")
	templatizeCmd.Flags().StringVarP(&inputFile, "input-file", "i",
		"", "Input file containing host IPs and indexes")
	templatizeCmd.Flags().StringVarP(&ip, "ip", "I",
		"", "IP address of host; if not given, EC2 metadata will be searched")
	templatizeCmd.Flags().StringVarP(&dest, "dest", "D",
		"", "Destination path for expanded file")
	templatizeCmd.Flags().StringVarP(&owner, "owner", "o",
		"root", "Owner of destination file")
	templatizeCmd.Flags().StringVarP(&group, "group", "g",
		"root", "Group of destination file")
	templatizeCmd.Flags().IntVarP(&mode, "mode", "m",
		00644, "Mode of destination file")
	templatizeCmd.Flags().StringSliceVarP(&vars, "var", "v",
		[]string{}, "Variable to pass to template")
}
