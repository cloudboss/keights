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

package whisperer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type Whisperer interface {
	ForceStoreParameter(ctx context.Context, path, kmsKeyID, content string) error
	StoreParameter(ctx context.Context, path, kmsKeyID, content string) error
	DeleteParameter(ctx context.Context, path string) error
	DeleteByPath(ctx context.Context, path string) error
	HasParameters(ctx context.Context, paths ...string) (bool, error)
	GetParameter(ctx context.Context, path string) (*string, error)
}

type ssmAPI interface {
	ssm.DescribeParametersAPIClient
	ssm.GetParametersByPathAPIClient
	DeleteParameter(context.Context, *ssm.DeleteParameterInput,
		...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error)
	DeleteParameters(context.Context, *ssm.DeleteParametersInput,
		...func(*ssm.Options)) (*ssm.DeleteParametersOutput, error)
	GetParameters(context.Context, *ssm.GetParametersInput,
		...func(*ssm.Options)) (*ssm.GetParametersOutput, error)
	PutParameter(context.Context, *ssm.PutParameterInput,
		...func(*ssm.Options)) (*ssm.PutParameterOutput, error)
}

var _ Whisperer = (*whisperer)(nil)

type whisperer struct {
	ssm ssmAPI
}

func NewSSMWhisperer(cfg aws.Config) *whisperer {
	return &whisperer{ssm: ssm.NewFromConfig(cfg)}
}

func (w *whisperer) storeParameter(
	ctx context.Context,
	path, kmsKeyID, content string,
	overwrite bool,
) error {
	input := &ssm.PutParameterInput{
		Name:      &path,
		Type:      types.ParameterTypeSecureString,
		Value:     &content,
		Overwrite: &overwrite,
	}
	if kmsKeyID != "" {
		input.KeyId = &kmsKeyID
	}
	_, err := w.ssm.PutParameter(ctx, input)
	return err
}

func (w *whisperer) StoreParameter(ctx context.Context, path, kmsKeyID, content string) error {
	return w.storeParameter(ctx, path, kmsKeyID, content, false)
}

func (w *whisperer) ForceStoreParameter(
	ctx context.Context,
	path, kmsKeyID, content string,
) error {
	return w.storeParameter(ctx, path, kmsKeyID, content, true)
}

func (w *whisperer) DeleteParameter(ctx context.Context, path string) error {
	_, err := w.ssm.DeleteParameter(ctx, &ssm.DeleteParameterInput{Name: &path})
	return err
}

// DeleteByPath deletes every parameter under the given path prefix, recursively. It is a
// no-op when no parameters are found.
func (w *whisperer) DeleteByPath(ctx context.Context, path string) error {
	recursive := true
	paginator := ssm.NewGetParametersByPathPaginator(w.ssm,
		&ssm.GetParametersByPathInput{Path: &path, Recursive: &recursive})
	names := []string{}
	for paginator.HasMorePages() {
		next, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, parameter := range next.Parameters {
			if parameter.Name != nil {
				names = append(names, *parameter.Name)
			}
		}
	}
	// DeleteParameters accepts at most 10 names per call.
	const batchSize = 10
	for start := 0; start < len(names); start += batchSize {
		end := start + batchSize
		if end > len(names) {
			end = len(names)
		}
		_, err := w.ssm.DeleteParameters(ctx,
			&ssm.DeleteParametersInput{Names: names[start:end]})
		if err != nil {
			return err
		}
	}
	return nil
}

// HasParameters checks SSM Parameter Store for the existence of the given parameters. All
// parameters must exist in order to return true. An error is returned if the SSM service
// returns an error.
func (w *whisperer) HasParameters(ctx context.Context, paths ...string) (bool, error) {
	filters := []types.ParameterStringFilter{
		{
			Key:    aws.String("Name"),
			Option: aws.String("Equals"),
			Values: paths,
		},
	}
	parameters := []types.ParameterMetadata{}
	paginator := ssm.NewDescribeParametersPaginator(w.ssm,
		&ssm.DescribeParametersInput{ParameterFilters: filters})
	for paginator.HasMorePages() {
		next, err := paginator.NextPage(ctx)
		if err != nil {
			return false, err
		}
		parameters = append(parameters, next.Parameters...)
	}
	fmt.Printf("Parameters: %+v\n", parameters)
	return len(parameters) == len(paths), nil
}

func (w *whisperer) GetParameter(ctx context.Context, path string) (*string, error) {
	withDecryption := true
	response, err := w.ssm.GetParameters(ctx, &ssm.GetParametersInput{
		Names:          []string{path},
		WithDecryption: &withDecryption,
	})
	if err != nil {
		return nil, err
	}
	for _, parameter := range response.Parameters {
		return parameter.Value, nil
	}
	return nil, fmt.Errorf("secret %s not found", path)
}
