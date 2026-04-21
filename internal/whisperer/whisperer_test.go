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
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSSM struct {
	ssmAPI
	pages        []*ssm.GetParametersByPathOutput
	pageErr      error
	pageCalls    []*ssm.GetParametersByPathInput
	deleteErr    error
	deleteCalls  []*ssm.DeleteParametersInput
}

func (f *fakeSSM) GetParametersByPath(
	_ context.Context,
	in *ssm.GetParametersByPathInput,
	_ ...func(*ssm.Options),
) (*ssm.GetParametersByPathOutput, error) {
	f.pageCalls = append(f.pageCalls, in)
	if f.pageErr != nil {
		return nil, f.pageErr
	}
	if len(f.pages) == 0 {
		return &ssm.GetParametersByPathOutput{}, nil
	}
	out := f.pages[0]
	f.pages = f.pages[1:]
	return out, nil
}

func (f *fakeSSM) DeleteParameters(
	_ context.Context,
	in *ssm.DeleteParametersInput,
	_ ...func(*ssm.Options),
) (*ssm.DeleteParametersOutput, error) {
	f.deleteCalls = append(f.deleteCalls, in)
	if f.deleteErr != nil {
		return nil, f.deleteErr
	}
	return &ssm.DeleteParametersOutput{}, nil
}

func parameters(names ...string) []types.Parameter {
	out := make([]types.Parameter, len(names))
	for i, n := range names {
		out[i] = types.Parameter{Name: aws.String(n)}
	}
	return out
}

func TestDeleteByPath_EmptyPrefix(t *testing.T) {
	fake := &fakeSSM{}
	w := &whisperer{ssm: fake}

	err := w.DeleteByPath(context.Background(), "/keights/c1/cluster")

	require.NoError(t, err)
	require.Len(t, fake.pageCalls, 1)
	assert.Equal(t, "/keights/c1/cluster", *fake.pageCalls[0].Path)
	assert.True(t, *fake.pageCalls[0].Recursive)
	assert.Empty(t, fake.deleteCalls)
}

func TestDeleteByPath_SinglePage(t *testing.T) {
	fake := &fakeSSM{
		pages: []*ssm.GetParametersByPathOutput{
			{Parameters: parameters("/keights/c1/cluster/ca.crt",
				"/keights/c1/cluster/bootstrap-token")},
		},
	}
	w := &whisperer{ssm: fake}

	err := w.DeleteByPath(context.Background(), "/keights/c1/cluster")

	require.NoError(t, err)
	require.Len(t, fake.deleteCalls, 1)
	assert.ElementsMatch(t,
		[]string{"/keights/c1/cluster/ca.crt", "/keights/c1/cluster/bootstrap-token"},
		fake.deleteCalls[0].Names)
}

func TestDeleteByPath_Paginated(t *testing.T) {
	fake := &fakeSSM{
		pages: []*ssm.GetParametersByPathOutput{
			{
				Parameters: parameters("/keights/c1/controller/a"),
				NextToken:  aws.String("tok"),
			},
			{Parameters: parameters("/keights/c1/controller/b")},
		},
	}
	w := &whisperer{ssm: fake}

	err := w.DeleteByPath(context.Background(), "/keights/c1/controller")

	require.NoError(t, err)
	require.Len(t, fake.pageCalls, 2)
	assert.Nil(t, fake.pageCalls[0].NextToken)
	require.NotNil(t, fake.pageCalls[1].NextToken)
	assert.Equal(t, "tok", *fake.pageCalls[1].NextToken)
	require.Len(t, fake.deleteCalls, 1)
	assert.ElementsMatch(t,
		[]string{"/keights/c1/controller/a", "/keights/c1/controller/b"},
		fake.deleteCalls[0].Names)
}

func TestDeleteByPath_BatchesDeletesInGroupsOfTen(t *testing.T) {
	names := make([]string, 25)
	for i := range names {
		names[i] = fmt.Sprintf("/keights/c1/cluster/p%02d", i)
	}
	fake := &fakeSSM{
		pages: []*ssm.GetParametersByPathOutput{
			{Parameters: parameters(names...)},
		},
	}
	w := &whisperer{ssm: fake}

	err := w.DeleteByPath(context.Background(), "/keights/c1/cluster")

	require.NoError(t, err)
	require.Len(t, fake.deleteCalls, 3)
	assert.Len(t, fake.deleteCalls[0].Names, 10)
	assert.Len(t, fake.deleteCalls[1].Names, 10)
	assert.Len(t, fake.deleteCalls[2].Names, 5)

	var seen []string
	for _, call := range fake.deleteCalls {
		seen = append(seen, call.Names...)
	}
	assert.ElementsMatch(t, names, seen)
}

func TestDeleteByPath_ListError(t *testing.T) {
	sentinel := errors.New("list failed")
	fake := &fakeSSM{pageErr: sentinel}
	w := &whisperer{ssm: fake}

	err := w.DeleteByPath(context.Background(), "/keights/c1/cluster")

	assert.ErrorIs(t, err, sentinel)
	assert.Empty(t, fake.deleteCalls)
}

func TestDeleteByPath_DeleteError(t *testing.T) {
	sentinel := errors.New("delete failed")
	fake := &fakeSSM{
		pages: []*ssm.GetParametersByPathOutput{
			{Parameters: parameters("/keights/c1/cluster/ca.crt")},
		},
		deleteErr: sentinel,
	}
	w := &whisperer{ssm: fake}

	err := w.DeleteByPath(context.Background(), "/keights/c1/cluster")

	assert.ErrorIs(t, err, sentinel)
}

func TestDeleteByPath_SkipsParametersWithNilName(t *testing.T) {
	fake := &fakeSSM{
		pages: []*ssm.GetParametersByPathOutput{
			{Parameters: []types.Parameter{
				{Name: aws.String("/keights/c1/cluster/a")},
				{Name: nil},
			}},
		},
	}
	w := &whisperer{ssm: fake}

	err := w.DeleteByPath(context.Background(), "/keights/c1/cluster")

	require.NoError(t, err)
	require.Len(t, fake.deleteCalls, 1)
	assert.Equal(t, []string{"/keights/c1/cluster/a"}, fake.deleteCalls[0].Names)
}
