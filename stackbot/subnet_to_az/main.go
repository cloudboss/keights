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
	"strings"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/mapstructure"
)

func Handle(_ctx context.Context, event cfn.Event) (string, map[string]interface{}, error) {
	fmt.Printf("Event: %v\n", event)

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
	azs, err := subnetsToAZs(client, props.SubnetIDs)
	if err != nil {
		return physicalResourceID, data, err
	}

	data["AvailabilityZones"] = azs

	fmt.Printf("Data: %v\n", data)

	return physicalResourceID, data, err
}

type resourceProperties struct {
	ServiceToken *string
	SubnetIDs    []*string `mapstructure:"SubnetIds"`
}

func subnetsToAZs(client *ec2.EC2, subnetIDs []*string) (string, error) {
	azs := make([]string, len(subnetIDs))
	for i, subnetID := range subnetIDs {
		subnetsOutput, err := client.DescribeSubnets(
			&ec2.DescribeSubnetsInput{
				SubnetIds: []*string{subnetID},
			},
		)
		if err != nil {
			return "", fmt.Errorf("error describing subnets: %v", err.Error())
		}
		subnets := subnetsOutput.Subnets
		numSubnets := len(subnets)
		if numSubnets != 1 {
			return "", fmt.Errorf("expected 1 subnet, found %d", numSubnets)
		}
		azs[i] = *subnets[0].AvailabilityZone
	}
	return strings.Join(azs, ","), nil
}

func main() {
	lambda.Start(cfn.LambdaWrap(Handle))
}
