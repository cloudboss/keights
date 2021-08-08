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
	"github.com/deniswernert/go-fstab"
)

const (
	Blkid = "blkid"
	Mkfs  = "mkfs"
	Fstab = "/etc/fstab"
)

type Volumizer struct {
	autoscaling      *autoscaling.AutoScaling
	ec2              *ec2.EC2
	availabilityZone string
	instanceID       string
}

func NewVolumizer(sess *session.Session, availabilityZone, instanceID string) *Volumizer {
	return &Volumizer{
		autoscaling:      autoscaling.New(sess),
		ec2:              ec2.New(sess),
		availabilityZone: availabilityZone,
		instanceID:       instanceID,
	}
}

func (v *Volumizer) AttachedVolume(clusterName, volumeTag, device *string) (*ec2.Volume, error) {
	filters := []*ec2.Filter{
		{
			Name:   aws.String("tag:Name"),
			Values: []*string{clusterName},
		},
		{
			Name:   aws.String("tag-key"),
			Values: []*string{volumeTag},
		},
		{
			Name:   aws.String("attachment.instance-id"),
			Values: []*string{aws.String(v.instanceID)},
		},
		{
			Name:   aws.String("attachment.device"),
			Values: []*string{device},
		},
	}
	input := &ec2.DescribeVolumesInput{Filters: filters}
	output, err := v.ec2.DescribeVolumes(input)
	if err != nil {
		return nil, err
	}
	numVol := len(output.Volumes)
	if numVol == 0 {
		return nil, nil
	}
	return output.Volumes[0], nil
}

func (v *Volumizer) WaitForVolume(clusterName, volumeTag *string, minutes time.Duration) (*ec2.Volume, error) {
	var output *ec2.DescribeVolumesOutput
	filters := []*ec2.Filter{
		{
			Name:   aws.String("tag:Name"),
			Values: []*string{clusterName},
		},
		{
			Name:   aws.String("tag-key"),
			Values: []*string{volumeTag},
		},
		{
			Name:   aws.String("availability-zone"),
			Values: []*string{aws.String(v.availabilityZone)},
		},
		{
			Name:   aws.String("status"),
			Values: []*string{aws.String("available")},
		},
	}
	input := &ec2.DescribeVolumesInput{Filters: filters}
	err := helpers.WaitFor(minutes*time.Minute, func() error {
		var err error
		output, err = v.ec2.DescribeVolumes(input)
		if err != nil {
			return err
		}
		numVol := len(output.Volumes)
		if numVol != 1 {
			return fmt.Errorf("Expected 1 volume, found %d", numVol)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return output.Volumes[0], nil
}

func (v *Volumizer) AttachVolume(volume *ec2.Volume, device string) error {
	input := &ec2.AttachVolumeInput{
		Device:     aws.String(device),
		InstanceId: &v.instanceID,
		VolumeId:   volume.VolumeId,
	}
	return helpers.WaitFor(10*time.Minute, func() error {
		_, err := v.ec2.AttachVolume(input)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "VolumeInUse" {
					return nil
				}
			}
		}
		return err
	})
}

func (v *Volumizer) WaitForDevice(device string) error {
	return helpers.WaitFor(10*time.Minute, func() error {
		_, err := os.Stat(device)
		if err != nil {
			return fmt.Errorf("No device %s found", device)
		}
		return nil
	})
}

func (v *Volumizer) HasFilesystem(device, fstype string) (bool, error) {
	blkid := helpers.RunCommand(Blkid, "-s", "TYPE", "-o", "value", device)
	if blkid.ExitStatus == 0 {
		stdout := strings.TrimSpace(blkid.Stdout)
		if stdout != fstype {
			return false, fmt.Errorf("Expected filesystem type %s, got %s", fstype, stdout)
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
	mkfs := helpers.RunCommand(Mkfs, "-t", fstype, device)
	if mkfs.ExitStatus != 0 {
		return fmt.Errorf(mkfs.Stderr)
	}
	return nil
}

func NormalizeDevice(device string) string {
	if !strings.HasPrefix(device, "/dev/") {
		return fmt.Sprintf("/dev/%s", device)
	}
	return device
}

func GetUUID(device string) (string, error) {
	blkid := helpers.RunCommand(Blkid, "-s", "UUID", "-o", "value", device)
	if blkid.ExitStatus == 0 {
		return strings.TrimSpace(blkid.Stdout), nil
	}
	return "", fmt.Errorf("Failed to get UUID of device %s: %s", device, blkid.Stderr)
}

func PersistFilesystem(uuid, fsType, mountPoint, fstabPath string) error {
	spec := fmt.Sprintf("UUID=%s", uuid)
	ourMount := &fstab.Mount{
		Spec:    spec,
		File:    mountPoint,
		VfsType: fsType,
		MntOps: map[string]string{
			"noatime": "",
			"errors":  "remount-ro",
		},
	}
	mounts, err := fstab.ParseFile(fstabPath)
	if err != nil {
		return err
	}
	for _, mount := range mounts {
		if mount.Spec == spec {
			return nil
		}
	}
	ourMountLine := []byte(ourMount.String() + "\n")
	return helpers.AppendToFile(fstabPath, ourMountLine, 0644)
}

func EnsureFilesystemsMounted() error {
	out := helpers.RunCommand("mount", "-a")
	if out.ExitStatus != 0 {
		return fmt.Errorf(out.Stderr)
	}
	return nil
}

func DoIt(device, volumeTag, fsType, mountPoint, clusterName string, minutes int) error {
	device = NormalizeDevice(device)
	sess := session.New()
	metadata := ec2metadata.New(sess)
	identity, err := metadata.GetInstanceIdentityDocument()
	if err != nil {
		return err
	}
	var volume *ec2.Volume
	volumizer := NewVolumizer(sess, identity.AvailabilityZone, identity.InstanceID)
	volume, err = volumizer.AttachedVolume(&clusterName, &volumeTag, &device)
	if err != nil {
		return err
	}
	if volume == nil {
		volume, err = volumizer.WaitForVolume(&clusterName, &volumeTag, time.Duration(minutes))
		if err != nil {
			return err
		}
		if err = volumizer.AttachVolume(volume, device); err != nil {
			return err
		}
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
	uuid, err := GetUUID(device)
	if err != nil {
		return err
	}
	if err = PersistFilesystem(uuid, fsType, mountPoint, Fstab); err != nil {
		return err
	}
	if err = os.MkdirAll(mountPoint, 0755); err != nil {
		return err
	}
	return EnsureFilesystemsMounted()
}
