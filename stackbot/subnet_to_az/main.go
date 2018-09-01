// Copyright 2017 Joseph Wright <joseph@cloudboss.co>
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

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/mapstructure"
)

func Handle(_ctx context.Context, event cfn.Event) (string, map[string]interface{}, error) {
	physicalResourceID := ""
	data := make(map[string]interface{})

	if event.RequestType == "Delete" {
		return physicalResourceID, data, nil
	}

	var props resourceProperties
	err := mapstructure.Decode(event.ResourceProperties, &props)
	if err != nil {
		return physicalResourceID, data, err
	}

	sess, err := session.NewSession()
	if err != nil {
		return physicalResourceID, data, err
	}

	client := ec2.New(sess)
	az, err := subnetToAZ(client, *props.SubnetID)
	if err != nil {
		return physicalResourceID, data, err
	}

	data["AvailabilityZone"] = az

	return physicalResourceID, data, err
}

type resourceProperties struct {
	ServiceToken *string
	SubnetID     *string `mapstructure:"SubnetId"`
}

func subnetToAZ(client *ec2.EC2, subnetID string) (string, error) {
	subnetsOutput, err := client.DescribeSubnets(
		&ec2.DescribeSubnetsInput{
			SubnetIds: []*string{aws.String(subnetID)},
		},
	)
	if err != nil {
		return "", err
	}
	numSubnets := len(subnetsOutput.Subnets)
	if numSubnets != 1 {
		return "", fmt.Errorf("Expected 1 subnet, found %d", numSubnets)
	}
	subnet := subnetsOutput.Subnets[0]
	return *subnet.AvailabilityZone, nil
}

func main() {
	lambda.Start(cfn.LambdaWrap(Handle))
}
