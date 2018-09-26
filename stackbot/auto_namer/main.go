// Copyright 2018 Joseph Wright <joseph@cloudboss.co>
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
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/cloudboss/keights/pkg/helpers"
)

const (
	ec2InstanceLaunchSuccessful = "EC2 Instance Launch Successful"
	asgNameEnv                  = "ASG_NAME"
	dnsTTLEnv                   = "DNS_TTL"
	hostBaseNameEnv             = "HOST_BASE_NAME"
	hostedZoneNameEnv           = "HOSTED_ZONE_NAME"
	hostedZoneIDEnv             = "HOSTED_ZONE_ID"
	actionUpsert                = "UPSERT"
)

var requiredEnvironment = []string{
	asgNameEnv,
	dnsTTLEnv,
	hostBaseNameEnv,
	hostedZoneNameEnv,
	hostedZoneIDEnv,
}

type autoscalingDetailDetails struct {
	AvailabilityZone string `json:"Availability Zone,omitempty"`
	SubnetID         string `json:"Subnet ID,omitempty"`
}

type autoscalingDetail struct {
	StatusCode           string                    `json:"StatusCode,omitempty"`
	AutoScalingGroupName string                    `json:"AutoScalingGroupName,omitempty"`
	ActivityID           string                    `json:"ActivityId,omitempty"`
	Details              *autoscalingDetailDetails `json:",omitempty"`
	RequestID            string                    `json:"RequestId,omitempty"`
	StartTime            string                    `json:"StartTime,omitempty"`
	EndTime              string                    `json:"EndTime,omitempty"`
	EC2InstanceID        string                    `json:"EC2InstanceId,omitempty"`
	Cause                string                    `json:"Cause,omitempty"`
}

func privateIP(ec2Client ec2iface.EC2API, instanceID string) (string, error) {
	var instanceIPs []string

	output, err := ec2Client.DescribeInstances(
		&ec2.DescribeInstancesInput{
			InstanceIds: []*string{&instanceID},
		},
	)
	if err != nil {
		return "", err
	}

	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceIPs = append(instanceIPs, *instance.PrivateIpAddress)
		}

	}

	lenInstanceIPs := len(instanceIPs)
	if lenInstanceIPs != 1 {
		return "", fmt.Errorf("expected 1 instance, found %d", lenInstanceIPs)
	}

	return instanceIPs[0], nil
}

func newARecordSet(hostBaseName, az, hostedZoneName, ip string, ttl int64) *route53.ResourceRecordSet {
	hostName := fmt.Sprintf("%s-%s.%s", hostBaseName, az, hostedZoneName)
	return &route53.ResourceRecordSet{
		Name: &hostName,
		Type: aws.String("A"),
		TTL:  &ttl,
		ResourceRecords: []*route53.ResourceRecord{
			{
				Value: &ip,
			},
		},
	}
}

func modifyRecord(r53Client route53iface.Route53API, record *route53.ResourceRecordSet, hostedZoneID string) error {
	action := aws.String(actionUpsert)
	_, err := r53Client.ChangeResourceRecordSets(
		&route53.ChangeResourceRecordSetsInput{
			HostedZoneId: &hostedZoneID,
			ChangeBatch: &route53.ChangeBatch{
				Changes: []*route53.Change{
					{
						Action:            action,
						ResourceRecordSet: record,
					},
				},
			},
		},
	)
	return err
}

func handleRecord(ec2Client ec2iface.EC2API, r53Client route53iface.Route53API,
	instanceID, az string, env map[string]string) error {
	ip, err := privateIP(ec2Client, instanceID)
	if err != nil {
		return err
	}

	// helpers.EnsureEnvironment() has already validated map values
	dnsTTL, _ := env[dnsTTLEnv]
	hostBaseName, _ := env[hostBaseNameEnv]
	hostedZoneName, _ := env[hostedZoneNameEnv]
	hostedZoneID, _ := env[hostedZoneIDEnv]

	ttl, err := strconv.ParseInt(dnsTTL, 10, 64)
	if err != nil {
		return err
	}

	recordSet := newARecordSet(hostBaseName, az, hostedZoneName, ip, ttl)
	return modifyRecord(r53Client, recordSet, hostedZoneID)
}

func handle(_ctx context.Context, event events.CloudWatchEvent) error {
	fmt.Printf("event: %+v\n", event)

	if event.DetailType != ec2InstanceLaunchSuccessful {
		fmt.Printf("Received event %s, nothing to do\n", event.DetailType)
		return nil
	}

	env, err := helpers.EnsureEnvironment(requiredEnvironment)
	if err != nil {
		return err
	}

	var detail autoscalingDetail
	err = json.Unmarshal(event.Detail, &detail)
	if err != nil {
		return err
	}
	fmt.Printf("event.detail: %+v\n", detail)

	if asgName, ok := env[asgNameEnv]; ok {
		if detail.AutoScalingGroupName != asgName {
			fmt.Printf("Event does not match ASG %s, nothing to do\n", asgName)
			return nil
		}
	}

	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	ec2Client := ec2.New(sess)
	r53Client := route53.New(sess)

	return handleRecord(ec2Client, r53Client, detail.EC2InstanceID, detail.Details.AvailabilityZone, env)
}

func main() {
	lambda.Start(handle)
}
