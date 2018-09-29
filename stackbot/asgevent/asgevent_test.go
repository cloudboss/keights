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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetailUnmarshal(t *testing.T) {
	var tests = []struct {
		js     []byte
		detail AutoScalingDetail
		errMsg string
	}{
		{
			[]byte(`{}`),
			AutoScalingDetail{},
			"",
		},
		{
			[]byte(`
                        {
                          "Details": {
                            "Availability Zone": "us-east-1f",
                            "Subnet ID": "subnet-01234567abcdef012"
                          }
                        }
                        `),
			AutoScalingDetail{
				AutoScalingGroupName: "",
				Details: DetailDetails{
					AvailabilityZone: "us-east-1f",
					SubnetID:         "subnet-01234567abcdef012",
				},
			},
			"",
		},
		{
			[]byte(`
                        {
                          "Details": {}
                        }
                        `),
			AutoScalingDetail{
				Details: DetailDetails{},
			},
			"",
		},
		{
			[]byte(`
                        {
                          "EC2InstanceId": "i-01234567abcdef012",
                          "Details": {}
                        }
                        `),
			AutoScalingDetail{
				EC2InstanceID: "i-01234567abcdef012",
				Details:       DetailDetails{},
			},
			"",
		},
		{
			[]byte(`
                        {
                          "Something": "",
                          "Details": {
                            "Availability Zone": "us-east-1f",
                            "Subnet ID": "subnet-01234567abcdef012",
                            "Junk": "stuff"
                          }
                        }
                        `),
			AutoScalingDetail{
				AutoScalingGroupName: "",
				Details: DetailDetails{
					AvailabilityZone: "us-east-1f",
					SubnetID:         "subnet-01234567abcdef012",
				},
			},
			"",
		},
		{
			[]byte(`
                        {
                          "StatusCode": "InProgress",
                          "AutoScalingGroupName": "sampleLuanchSucASG",
                          "ActivityId": "9cabb81f-42de-417d-8aa7-ce16bf026590",
                          "Details": {
                            "Availability Zone": "us-east-1b",
                            "Subnet ID": "subnet-95bfcebe"
                          },
                          "RequestId": "9cabb81f-42de-417d-8aa7-ce16bf026590",
                          "EndTime": "2015-11-11T21:31:47.208Z",
                          "EC2InstanceId": "i-b188560f",
                          "StartTime": "2015-11-11T21:31:13.671Z",
                          "Cause": "At 2015-11-11T21:31:10Z a user request created an AutoScalingGroup changing the desired capacity from 0 to 1.  At 2015-11-11T21:31:11Z an instance was started in response to a difference between desired and actual capacity, increasing the capacity from 0 to 1."
                        }
                        `),
			AutoScalingDetail{
				StatusCode:           "InProgress",
				AutoScalingGroupName: "sampleLuanchSucASG",
				ActivityID:           "9cabb81f-42de-417d-8aa7-ce16bf026590",
				RequestID:            "9cabb81f-42de-417d-8aa7-ce16bf026590",
				EndTime:              "2015-11-11T21:31:47.208Z",
				EC2InstanceID:        "i-b188560f",
				StartTime:            "2015-11-11T21:31:13.671Z",
				Cause:                "At 2015-11-11T21:31:10Z a user request created an AutoScalingGroup changing the desired capacity from 0 to 1.  At 2015-11-11T21:31:11Z an instance was started in response to a difference between desired and actual capacity, increasing the capacity from 0 to 1.",
				Details: DetailDetails{
					AvailabilityZone: "us-east-1b",
					SubnetID:         "subnet-95bfcebe",
				},
			},
			"",
		},
		{
			[]byte(`{`),
			AutoScalingDetail{},
			"unexpected end of JSON input",
		},
	}
	for _, test := range tests {
		var detail AutoScalingDetail
		err := json.Unmarshal(test.js, &detail)
		if err != nil {
			assert.Equal(t, err.Error(), test.errMsg)
		} else {
			assert.Equal(t, detail, test.detail)
			t.Logf("%#v\n", detail)
		}
	}
}

func TestValidateEvent(t *testing.T) {
	var tests = []struct {
		event       string
		validEvents []string
		errMsg      string
	}{
		{
			"EC2 Instance Launch Successful",
			[]string{
				"EC2 Instance Launch Successful",
			},
			"",
		},
		{
			"EC2 Instance Launch Successful",
			[]string{
				"EC2 Instance Launch Successful",
				"EC2 Instance Terminate Successful",
			},
			"",
		},
		{
			"EC2 Instance Launch UnSuccessful",
			[]string{
				"EC2 Instance Launch Successful",
				"EC2 Instance Terminate Successful",
			},
			`event "EC2 Instance Launch UnSuccessful" not expected`,
		},
		{
			"",
			[]string{},
			`event "" not expected`,
		},
	}
	for _, test := range tests {
		err := ValidateEvent(test.event, test.validEvents)
		if err != nil {
			assert.Equal(t, err.Error(), test.errMsg)
		}
	}
}
