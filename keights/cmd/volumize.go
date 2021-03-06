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
	"github.com/cloudboss/keights/pkg/volumize"
	"github.com/spf13/cobra"
)

var (
	device      string
	fsType      string
	mountPoint  string
	clusterName string
	minutes     int
	volumizeCmd = &cobra.Command{
		Use:   "volumize",
		Short: "Attach and format EBS volumes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return volumize.DoIt(device, volumeTag, fsType, mountPoint, clusterName, minutes)
		},
	}
)

func init() {
	RootCmd.AddCommand(volumizeCmd)
	volumizeCmd.Flags().StringVarP(&device, "device", "d",
		"/dev/xvdg", "Name of device for EBS volume")
	volumizeCmd.Flags().StringVarP(&fsType, "fstype", "f",
		"ext4", "Type of filesystem to format volume")
	volumizeCmd.Flags().StringVarP(&mountPoint, "mount-point", "p",
		"/var/lib/etcd", "Filesystem path on which to mount volume")
	volumizeCmd.Flags().StringVarP(&volumeTag, "volume-tag", "v",
		"", "Tag to search on EBS volume")
	volumizeCmd.Flags().StringVarP(&clusterName, "clusterName", "c",
		"", "Name of Kubernetes cluster")
	volumizeCmd.Flags().IntVarP(&minutes, "minutes", "m",
		60, "Number of minutes to wait")
}
