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
	"github.com/cloudboss/keights/stackbot/asgevent"
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

var validEvents = []string{
	ec2InstanceLaunchSuccessful,
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


func handleRecord(ec2Client ec2iface.EC2API, r53Client route53iface.Route53API,
	instanceID, az string, env map[string]string) error {
	ip, err := privateIP(ec2Client, instanceID)
	if err != nil {
		return err
	}

	// asgevent.Handle() has already validated map values
	dnsTTL, _ := env[dnsTTLEnv]
	hostBaseName, _ := env[hostBaseNameEnv]
	hostedZoneName, _ := env[hostedZoneNameEnv]
	hostedZoneID, _ := env[hostedZoneIDEnv]

	ttl, err := strconv.ParseInt(dnsTTL, 10, 64)
	if err != nil {
		return err
	}

	recordSet := newARecordSet(hostBaseName, az, hostedZoneName, ip, ttl)

	action := aws.String(actionUpsert)
	_, err = r53Client.ChangeResourceRecordSets(
		&route53.ChangeResourceRecordSetsInput{
			HostedZoneId: &hostedZoneID,
			ChangeBatch: &route53.ChangeBatch{
				Changes: []*route53.Change{
					{
						Action:            action,
						ResourceRecordSet: recordSet,
					},
				},
			},
		},
	)
	return err
}

func realHandler(detail asgevent.AutoScalingDetail, env map[string]string) error {
	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	ec2Client := ec2.New(sess)
	r53Client := route53.New(sess)

	return handleRecord(ec2Client, r53Client, detail.EC2InstanceID, detail.Details.AvailabilityZone, env)
}

func main() {
	lambda.Start(func(ctx context.Context, event events.CloudWatchEvent) error {
		return asgevent.Handle(event, validEvents, requiredEnvironment, realHandler)
	})
}
