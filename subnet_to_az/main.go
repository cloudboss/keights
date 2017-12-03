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
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/cloudboss/stackhand/response"
	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
	"github.com/eawsy/aws-lambda-go-event/service/lambda/runtime/event/cloudformationevt"
)

func Handle(event *cloudformationevt.Event, ctx *runtime.Context) (interface{}, error) {
	responseBody := response.NewResponseBody(event, ctx)

	if event.RequestType != "Create" {
		responseBody.Status = response.Success
		response.FireResponse(event.ResponseURL, responseBody)
		return nil, nil
	}

	var props resourceProperties
	err := json.Unmarshal(event.ResourceProperties, &props)
	if err != nil {
		responseBody.Status = response.Failed
		responseBody.Reason = err.Error()
		response.FireResponse(event.ResponseURL, responseBody)
		return nil, err
	}

	client := ec2.New(session.New())
	az, err := subnetToAZ(client, *props.Region, *props.SubnetID)
	if err != nil {
		responseBody.Status = response.Failed
		responseBody.Reason = err.Error()
		response.FireResponse(event.ResponseURL, responseBody)
		return nil, err
	}

	responseBody.Status = response.Success
	responseBody.Data = map[string]string{"AvailabilityZone": az}
	response.FireResponse(event.ResponseURL, responseBody)

	return nil, nil
}

type resourceProperties struct {
	ServiceToken *string
	Region       *string
	SubnetID     *string `json:"SubnetId"`
}

func subnetToAZ(client *ec2.EC2, region, subnetID string) (string, error) {
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
