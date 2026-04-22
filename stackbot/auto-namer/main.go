// Copyright 2026 Joseph Wright <joseph@cloudboss.co>
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
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/cloudboss/keights/stackbot/asgevent"
)

const (
	ec2InstanceLaunchSuccessful = "EC2 Instance Launch Successful"
	asgNameEnv                  = "ASG_NAME"
	dnsTTLEnv                   = "DNS_TTL"
	hostBaseNameEnv             = "HOST_BASE_NAME"
	hostedZoneNameEnv           = "HOSTED_ZONE_NAME"
	hostedZoneIDEnv             = "HOSTED_ZONE_ID"
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

func privateIP(ctx context.Context, ec2Client *ec2.Client, instanceID string) (string, error) {
	var instanceIPs []string

	output, err := ec2Client.DescribeInstances(ctx,
		&ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceID},
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

func newARecordSet(
	hostBaseName, az, hostedZoneName, ip string,
	ttl int64,
) *route53types.ResourceRecordSet {
	hostName := fmt.Sprintf("%s-%s.%s", hostBaseName, az, hostedZoneName)
	return &route53types.ResourceRecordSet{
		Name: &hostName,
		Type: route53types.RRTypeA,
		TTL:  &ttl,
		ResourceRecords: []route53types.ResourceRecord{
			{
				Value: &ip,
			},
		},
	}
}

func handleRecord(
	ctx context.Context,
	ec2Client *ec2.Client,
	r53Client *route53.Client,
	instanceID, az string,
	env map[string]string,
) error {
	ip, err := privateIP(ctx, ec2Client, instanceID)
	if err != nil {
		return err
	}

	// asgevent.Handle() has already validated map values
	dnsTTL := env[dnsTTLEnv]
	hostBaseName := env[hostBaseNameEnv]
	hostedZoneName := env[hostedZoneNameEnv]
	hostedZoneID := env[hostedZoneIDEnv]

	ttl, err := strconv.ParseInt(dnsTTL, 10, 64)
	if err != nil {
		return err
	}

	recordSet := newARecordSet(hostBaseName, az, hostedZoneName, ip, ttl)

	action := route53types.ChangeActionUpsert
	_, err = r53Client.ChangeResourceRecordSets(
		ctx,
		&route53.ChangeResourceRecordSetsInput{
			HostedZoneId: &hostedZoneID,
			ChangeBatch: &route53types.ChangeBatch{
				Changes: []route53types.Change{
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

func realHandler(
	ctx context.Context,
	detail asgevent.AutoScalingDetail,
	env map[string]string,
) error {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return err
	}

	ec2Client := ec2.NewFromConfig(cfg)
	r53Client := route53.NewFromConfig(cfg)

	return handleRecord(ctx, ec2Client, r53Client, detail.EC2InstanceID,
		detail.Details.AvailabilityZone, env)
}

func main() {
	lambda.Start(func(ctx context.Context, event events.CloudWatchEvent) error {
		return asgevent.Handle(ctx, event, validEvents, requiredEnvironment, realHandler)
	})
}
