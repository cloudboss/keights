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

package collect

import (
	"bytes"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/cloudboss/keights/pkg/helpers"
)

type Collector struct {
	asgName     *string
	volumeTag   *string
	autoscaling *autoscaling.AutoScaling
	ec2         *ec2.EC2
	asg         *autoscaling.Group
	instances   []*ec2.Instance
	volumes     []*ec2.Volume
}

func NewCollector(sess *session.Session, asgName, volumeTag *string) *Collector {
	asgClient := autoscaling.New(sess)
	collector := &Collector{
		asgName:     asgName,
		volumeTag:   volumeTag,
		autoscaling: asgClient,
		ec2:         ec2.New(sess),
	}
	return collector
}

func (c *Collector) Describe() (*autoscaling.Group, error) {
	if c.asg == nil {
		if err := c.Refresh(); err != nil {
			return nil, err
		}
	}
	return c.asg, nil
}

func (c *Collector) Refresh() error {
	input := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{c.asgName},
	}
	out, err := c.autoscaling.DescribeAutoScalingGroups(&input)
	if err != nil {
		return err
	}
	numGroups := len(out.AutoScalingGroups)
	if numGroups != 1 {
		return fmt.Errorf("Expected 1 group, found %d", numGroups)
	}
	c.asg = out.AutoScalingGroups[0]
	return nil
}

func (c *Collector) Instances() ([]*ec2.Instance, error) {
	if err := c.Refresh(); err != nil {
		return nil, err
	}
	desc, err := c.ec2.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: InstanceIds(c.asg.Instances),
	})
	if err != nil {
		return nil, err
	}
	instances := []*ec2.Instance{}
	for _, reservation := range desc.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, instance)
		}
	}
	numInst := int64(len(instances))
	if numInst != *c.asg.DesiredCapacity {
		return nil, fmt.Errorf("Expected %d instances, found %d",
			c.asg.DesiredCapacity, numInst)
	}
	for _, instance := range instances {
		if instance.PrivateIpAddress == nil {
			return nil, fmt.Errorf("Instance %v has no private IP",
				instance.InstanceId)
		}
	}
	c.instances = instances
	return instances, nil
}

func (c *Collector) Volumes() ([]*ec2.Volume, error) {
	if err := c.Refresh(); err != nil {
		return nil, err
	}
	desc, err := c.ec2.DescribeVolumes(&ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("attachment.instance-id"),
				Values: InstanceIds(c.asg.Instances),
			},
			&ec2.Filter{
				Name:   aws.String("attachment.status"),
				Values: []*string{aws.String("attached")},
			},
			&ec2.Filter{
				Name:   aws.String("tag-key"),
				Values: []*string{c.volumeTag},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	volumes := []*ec2.Volume{}
	for _, volume := range desc.Volumes {
		volumes = append(volumes, volume)
	}
	numVol := int64(len(volumes))
	if numVol != *c.asg.DesiredCapacity {
		return nil, fmt.Errorf("Expected %d volumes, found %d",
			*c.asg.DesiredCapacity, numVol)
	}
	c.volumes = volumes
	return volumes, nil
}

func (c *Collector) WaitForInstances() ([]*ec2.Instance, error) {
	err := WaitFor(5*time.Minute, func() error {
		_, err := c.Instances()
		return err
	})
	return c.instances, err
}

func (c *Collector) WaitForVolumes() ([]*ec2.Volume, error) {
	err := WaitFor(10*time.Minute, func() error {
		_, err := c.Volumes()
		return err
	})
	return c.volumes, err
}

func (c *Collector) Mapping(instances []*ec2.Instance, volumes []*ec2.Volume) (map[string]string, error) {
	mapping := make(map[string]string)
	for _, volume := range volumes {
		var index string
		for _, tag := range volume.Tags {
			if *tag.Key == *c.volumeTag {
				index = *tag.Value
			}
		}
		attachment := AttachedAttachment(volume.Attachments)
		if attachment == nil {
			err := fmt.Errorf("Volume %v not attached", volume.VolumeId)
			return nil, err
		}
		for _, instance := range instances {
			if *attachment.InstanceId == *instance.InstanceId {
				mapping[index] = *instance.PrivateIpAddress
			}
		}
	}
	return mapping, nil
}

func AttachedAttachment(attachments []*ec2.VolumeAttachment) *ec2.VolumeAttachment {
	for _, attachment := range attachments {
		if *attachment.State == "attached" {
			return attachment
		}
	}
	return nil
}

func InstanceIds(instances []*autoscaling.Instance) []*string {
	instanceIds := []*string{}
	for _, instance := range instances {
		instanceIds = append(instanceIds, instance.InstanceId)
	}
	return instanceIds
}

func WaitFor(duration time.Duration, checker func() error) error {
	var err error
	c := make(chan bool)
	go func() {
		for {
			err = checker()
			if err != nil {
				time.Sleep(5 * time.Second)
			} else {
				c <- true
			}
		}
	}()
	select {
	case <-c:
		return nil
	case <-time.After(duration):
		return err
	}
}

func WriteOutput(mapping map[string]string, outputFile string) error {
	keys := helpers.SortMapKeys(mapping)
	var buf bytes.Buffer
	for _, key := range keys {
		fmt.Fprintf(&buf, "%s:%s\n", key, mapping[key])
	}
	return helpers.WriteIfChanged(outputFile, buf.Bytes())
}

func DoIt(volumeTag, outputFile string) error {
	sess := session.New()
	asgName, err := helpers.AsgName(sess)
	if err != nil {
		return err
	}
	collector := NewCollector(sess, asgName, aws.String(volumeTag))
	instances, err := collector.WaitForInstances()
	if err != nil {
		return err
	}
	volumes, err := collector.WaitForVolumes()
	if err != nil {
		return err
	}
	mapping, err := collector.Mapping(instances, volumes)
	return WriteOutput(mapping, outputFile)
}
