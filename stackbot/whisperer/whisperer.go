// Copyright 2019 Joseph Wright <joseph@cloudboss.co>
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

package whisperer

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

type Whisperer interface {
	ForceStoreParameter(path, kmsKeyID, content *string) error
	StoreParameter(path, kmsKeyID, content *string) error
	DeleteParameter(path *string) error
	HasParameters(paths ...*string) (bool, error)
	GetParameter(path *string) (*string, error)
}

type whisperer struct {
	SSMClient ssmiface.SSMAPI
}

func NewSSMWhisperer(sess *session.Session) *whisperer {
	return &whisperer{SSMClient: ssm.New(sess)}
}

func (w *whisperer) storeParameter(path, kmsKeyID, content *string, overwrite bool) error {
	input := &ssm.PutParameterInput{
		Name:      path,
		Type:      aws.String("SecureString"),
		Value:     content,
		Overwrite: &overwrite,
	}
	if *kmsKeyID != "" {
		input.KeyId = kmsKeyID
	}
	_, err := w.SSMClient.PutParameter(input)
	return err
}

func (w *whisperer) StoreParameter(path, kmsKeyID, content *string) error {
	return w.storeParameter(path, kmsKeyID, content, false)
}

func (w *whisperer) ForceStoreParameter(path, kmsKeyID, content *string) error {
	return w.storeParameter(path, kmsKeyID, content, true)
}

func (w *whisperer) DeleteParameter(path *string) error {
	_, err := w.SSMClient.DeleteParameter(&ssm.DeleteParameterInput{Name: path})
	return err
}

// HasParameters checks SSM Parameter Store for the existence of the given parameters. All parameters
// must exist in order to return true. An error is returned if the SSM service returns an error.
func (w *whisperer) HasParameters(paths ...*string) (bool, error) {
	filters := []*ssm.ParameterStringFilter{
		{
			Key:    aws.String("Name"),
			Option: aws.String("Equals"),
			Values: paths,
		},
	}
	parameters := []*ssm.ParameterMetadata{}
	input := &ssm.DescribeParametersInput{ParameterFilters: filters}
	err := w.SSMClient.DescribeParametersPages(input, func(out *ssm.DescribeParametersOutput, lastPage bool) bool {
		parameters = append(parameters, out.Parameters...)
		return !lastPage
	})
	if err != nil {
		return false, err
	}
	fmt.Printf("Parameters: %+v\n", parameters)
	return len(parameters) == len(paths), nil
}

func (w *whisperer) GetParameter(path *string) (*string, error) {
	withDecryption := true
	response, err := w.SSMClient.GetParameters(&ssm.GetParametersInput{
		Names:          []*string{path},
		WithDecryption: &withDecryption,
	})
	if err != nil {
		return nil, err
	}
	for _, parameter := range response.Parameters {
		return parameter.Value, nil
	}
	return nil, fmt.Errorf("secret %s not found", *path)
}
