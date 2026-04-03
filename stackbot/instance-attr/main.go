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

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/cloudboss/keights/stackbot/asgevent"
)

const (
	ec2InstanceLaunchSuccessful = "EC2 Instance Launch Successful"
)

var requiredEnvironment = []string{}

var validEvents = []string{
	ec2InstanceLaunchSuccessful,
}

func realHandler(
	ctx context.Context,
	detail asgevent.AutoScalingDetail,
	env map[string]string,
) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	ec2Client := ec2.NewFromConfig(cfg)
	_, err = ec2Client.ModifyInstanceAttribute(
		ctx,
		&ec2.ModifyInstanceAttributeInput{
			InstanceId: &detail.EC2InstanceID,
			SourceDestCheck: &ec2types.AttributeBooleanValue{
				Value: aws.Bool(false),
			},
		},
	)
	return err
}

func main() {
	lambda.Start(func(ctx context.Context, event events.CloudWatchEvent) error {
		return asgevent.Handle(ctx, event, validEvents, requiredEnvironment, realHandler)
	})
}
