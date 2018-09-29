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

package asgevent

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/cloudboss/keights/pkg/helpers"
)

type DetailDetails struct {
	AvailabilityZone string `json:"Availability Zone"`
	SubnetID         string `json:"Subnet ID"`
}

type AutoScalingDetail struct {
	StatusCode           string
	AutoScalingGroupName string
	ActivityID           string `json:"ActivityId"`
	RequestID            string `json:"RequestId"`
	StartTime            string
	EndTime              string
	EC2InstanceID        string `json:"EC2InstanceId"`
	Cause                string
	Details              DetailDetails
}

type RealHandler func(detail AutoScalingDetail, env map[string]string) error

func ValidateEvent(actualEvent string, validEvents []string) error {
	found := false
	for _, event := range validEvents {
		if event == actualEvent {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf(`event "%s" not expected`, actualEvent)
	}
	return nil
}

func Handle(event events.CloudWatchEvent, validEvents, requiredEnvironment []string, doit RealHandler) error {
	fmt.Printf("event: %#v\n", event)

	err := ValidateEvent(event.DetailType, validEvents)
	if err != nil {
		return err
	}

	env, err := helpers.EnsureEnvironment(requiredEnvironment)
	if err != nil {
		return err
	}

	var detail AutoScalingDetail
	err = json.Unmarshal(event.Detail, &detail)
	if err != nil {
		return err
	}
	fmt.Printf("event.detail: %#v\n", detail)

	return doit(detail, env)
}
