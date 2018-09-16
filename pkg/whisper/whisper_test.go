// Copyright © 2018 Joseph Wright <joseph@cloudboss.co>
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

package whisper

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/cloudboss/keights/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetSecret(t *testing.T) {
	var tests = []struct {
		setGetParameters func(s *mocks.SSMAPI)
		path             string
		result           string
		errMsg           string
	}{
		{
			func(s *mocks.SSMAPI) {
				typ := "*ssm.GetParametersInput"
				output := &ssm.GetParametersOutput{
					Parameters: []*ssm.Parameter{
						{
							Value: aws.String("secretkey"),
						},
					},
				}
				s.On("GetParameters", mock.AnythingOfType(typ)).Return(output, nil)
			},
			"/otto-kube/controller/ca.key",
			"secretkey",
			"",
		},
		{
			func(s *mocks.SSMAPI) {
				err := fmt.Errorf("We have a problem")
				typ := "*ssm.GetParametersInput"
				s.On("GetParameters", mock.AnythingOfType(typ)).Return(nil, err)
			},
			"/otto-kube/controller/ca.key",
			"",
			fmt.Sprint("We have a problem"),
		},
		{
			func(s *mocks.SSMAPI) {
				typ := "*ssm.GetParametersInput"
				output := &ssm.GetParametersOutput{
					Parameters: []*ssm.Parameter{},
				}
				s.On("GetParameters", mock.AnythingOfType(typ)).Return(output, nil)
			},
			"/otto-kube/controller/ca.key",
			"",
			fmt.Sprintf("Secret /otto-kube/controller/ca.key not found"),
		},
	}
	for _, test := range tests {
		ssmClient := &mocks.SSMAPI{}
		test.setGetParameters(ssmClient)
		secret, err := getSecret(ssmClient, test.path)
		ssmClient.AssertExpectations(t)
		if test.result != "" {
			assert.Equal(t, *secret, test.result)
		} else {
			assert.EqualError(t, err, test.errMsg)
		}
	}
}

func TestParsePaths(t *testing.T) {
	mkErrMsg := func(s string) string {
		return fmt.Sprintf("Malformed path %s", s)
	}
	var tests = []struct {
		paths  []string
		parsed []map[string]string
		errMsg string
	}{
		{
			[]string{
				"",
			},
			nil,
			mkErrMsg(""),
		},
		{
			[]string{
				"/otto-kube/cluster/ca.crt:/run/keights/ca.crt",
				"/otto-kube/controller/ca.key",
			},
			nil,
			mkErrMsg("/otto-kube/controller/ca.key"),
		},
		{
			[]string{
				"/otto-kube/controller/ca.key",
				"/otto-kube/cluster/ca.crt:/run/keights/ca.crt",
			},
			nil,
			mkErrMsg("/otto-kube/controller/ca.key"),
		},
		{
			[]string{
				"/otto-kube/cluster/ca.crt:/run/keights/ca.crt",
				"/otto-kube/controller/ca.key",
				"/otto-kube/cluster/jakurb.crt:/run/keights/jakurb.crt",
			},
			nil,
			mkErrMsg("/otto-kube/controller/ca.key"),
		},
		{
			[]string{},
			[]map[string]string{},
			"",
		},
		{
			[]string{
				"/otto-kube/cluster/ca.crt:/run/keights/ca.crt",
			},
			[]map[string]string{
				{
					"path": "/otto-kube/cluster/ca.crt",
					"dest": "/run/keights/ca.crt",
				},
			},
			"",
		},
		{
			[]string{
				"/otto-kube/cluster/ca.crt:/run/keights/ca.crt",
				"/otto-kube/controller/ca.key:/run/keights/ca.key",
			},
			[]map[string]string{
				{
					"path": "/otto-kube/cluster/ca.crt",
					"dest": "/run/keights/ca.crt",
				},
				{
					"path": "/otto-kube/controller/ca.key",
					"dest": "/run/keights/ca.key",
				},
			},
			"",
		},
	}
	for _, test := range tests {
		parsed, err := parsePaths(test.paths)
		if test.parsed != nil {
			assert.Equal(t, parsed, test.parsed)
		} else {
			assert.EqualError(t, err, test.errMsg)
		}
	}
}
