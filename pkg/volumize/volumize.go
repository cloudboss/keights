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

package volumize

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/cloudboss/keights/pkg/helpers"
)

const (
	Blkid = "/usr/sbin/blkid"
	Mkfs  = "/usr/sbin/mkfs"
)

type Volumizer struct {
	autoscaling *autoscaling.AutoScaling
	ec2         *ec2.EC2
	identity    ec2metadata.EC2InstanceIdentityDocument
}

func NewVolumizer() (*Volumizer, error) {
	sess := session.New()
	metadata := ec2metadata.New(sess)
	identity, err := metadata.GetInstanceIdentityDocument()
	if err != nil {
		return nil, err
	}
	client := Volumizer{
		autoscaling: autoscaling.New(sess),
		ec2:         ec2.New(sess),
		identity:    identity,
	}
	return &client, nil
}

func (v *Volumizer) FindVolume(asg, volumeTag string) (*ec2.Volume, error) {
	az := v.identity.AvailabilityZone
	filters := []*ec2.Filter{
		&ec2.Filter{
			Name:   aws.String("tag:Name"),
			Values: []*string{aws.String(asg)},
		},
		&ec2.Filter{
			Name:   aws.String("tag-key"),
			Values: []*string{aws.String(volumeTag)},
		},
		&ec2.Filter{
			Name:   aws.String("availability-zone"),
			Values: []*string{aws.String(az)},
		},
	}
	input := &ec2.DescribeVolumesInput{Filters: filters}
	output, err := v.ec2.DescribeVolumes(input)
	if err != nil {
		return nil, err
	}
	numVol := len(output.Volumes)
	if numVol != 1 {
		return nil, fmt.Errorf("Expected 1 volume, found %d", numVol)
	}
	return output.Volumes[0], nil
}

func (v *Volumizer) AttachVolume(volume *ec2.Volume, device string) error {
	input := &ec2.AttachVolumeInput{
		Device:     aws.String(device),
		InstanceId: &v.identity.InstanceID,
		VolumeId:   volume.VolumeId,
	}
	_, err := v.ec2.AttachVolume(input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "VolumeInUse" {
				return nil
			}
		}
	}
	return err
}

func (v *Volumizer) WaitForVolume(volume *ec2.Volume) error {
	return v.ec2.WaitUntilVolumeInUse(
		&ec2.DescribeVolumesInput{
			VolumeIds: []*string{aws.String(*volume.VolumeId)},
		},
	)
}

func (v *Volumizer) WaitForDevice(device string) error {
	c := make(chan bool)
	devicePath := fmt.Sprintf("/dev/%s", device)
	go func() {
		for {
			if _, err := os.Stat(devicePath); err != nil {
				time.Sleep(1 * time.Second)
			} else {
				c <- true
			}
		}
	}()
	select {
	case <-c:
		return nil
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("No device %s found", devicePath)
	}
}

func (v *Volumizer) HasFilesystem(device, fstype string) (bool, error) {
	devicePath := fmt.Sprintf("/dev/%s", device)
	blkid := helpers.RunCommand(Blkid, "-p", "-s", "TYPE", "-o", "udev", devicePath)
	if blkid.ExitStatus == 0 {
		if !strings.HasPrefix(blkid.Stdout, fmt.Sprintf("ID_FS_TYPE=%v", fstype)) {
			return false, fmt.Errorf("Unexpected error reading %s", devicePath)
		}
		return true, nil
	}
	if blkid.ExitStatus == 2 {
		if blkid.Stderr != "" {
			return false, fmt.Errorf(blkid.Stderr)
		}
		return false, nil
	}
	return false, fmt.Errorf(blkid.Stderr)
}

func (v *Volumizer) MakeFilesystem(device, fstype string) error {
	devicePath := fmt.Sprintf("/dev/%s", device)
	mkfs := helpers.RunCommand(Mkfs, "-t", fstype, devicePath)
	if mkfs.ExitStatus != 0 {
		return fmt.Errorf(mkfs.Stderr)
	}
	return nil
}

func DoIt(asg, device, volumeTag, fsType string) error {
	volumizer, err := NewVolumizer()
	if err != nil {
		return err
	}
	volume, err := volumizer.FindVolume(asg, volumeTag)
	if err != nil {
		return err
	}
	if err = volumizer.AttachVolume(volume, device); err != nil {
		return err
	}
	if err = volumizer.WaitForVolume(volume); err != nil {
		return err
	}
	if err = volumizer.WaitForDevice(device); err != nil {
		return err
	}
	hasFs, err := volumizer.HasFilesystem(device, fsType)
	if err != nil {
		return err
	}
	if !hasFs {
		if err = volumizer.MakeFilesystem(device, fsType); err != nil {
			return err
		}
	}
	return nil
}
