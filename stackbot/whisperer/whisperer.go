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

package whisperer

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

type Whisperer interface {
	ForceStoreParameter(path, kmsKeyID string, content *string) error
	StoreParameter(path, kmsKeyID string, content *string) error
	DeleteParameter(path *string) error
}

type whisperer struct {
	SSMClient ssmiface.SSMAPI
}

func NewSSMWhisperer(sess *session.Session) *whisperer {
	return &whisperer{SSMClient: ssm.New(sess)}
}

func (w *whisperer) storeParameter(path, kmsKeyID string, content *string, overwrite bool) error {
	input := &ssm.PutParameterInput{
		Name:      aws.String(path),
		Type:      aws.String("SecureString"),
		Value:     content,
		Overwrite: &overwrite,
	}
	if kmsKeyID != "" {
		input.KeyId = aws.String(kmsKeyID)
	}
	_, err := w.SSMClient.PutParameter(input)
	return err
}

func (w *whisperer) StoreParameter(path, kmsKeyID string, content *string) error {
	return w.storeParameter(path, kmsKeyID, content, false)
}

func (w *whisperer) ForceStoreParameter(path, kmsKeyID string, content *string) error {
	return w.storeParameter(path, kmsKeyID, content, true)
}

func (w *whisperer) DeleteParameter(path *string) error {
	_, err := w.SSMClient.DeleteParameter(&ssm.DeleteParameterInput{Name: path})
	return err
}
