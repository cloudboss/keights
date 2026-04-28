// Copyright © 2026 Joseph Wright <joseph@cloudboss.co>
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

package ami

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	intami "github.com/cloudboss/keights/internal/ami"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubEC2 struct {
	images []ec2types.Image
	lastIn *ec2.DescribeImagesInput
}

func (s *stubEC2) DescribeImages(
	_ context.Context, in *ec2.DescribeImagesInput, _ ...func(*ec2.Options),
) (*ec2.DescribeImagesOutput, error) {
	s.lastIn = in
	if len(in.ImageIds) > 0 {
		ids := map[string]bool{}
		for _, id := range in.ImageIds {
			ids[id] = true
		}
		var matched []ec2types.Image
		for _, img := range s.images {
			if img.ImageId != nil && ids[*img.ImageId] {
				matched = append(matched, img)
			}
		}
		return &ec2.DescribeImagesOutput{Images: matched}, nil
	}
	return &ec2.DescribeImagesOutput{Images: s.images}, nil
}

func (*stubEC2) CopyImage(
	context.Context, *ec2.CopyImageInput, ...func(*ec2.Options),
) (*ec2.CopyImageOutput, error) {
	return nil, nil
}

func strp(s string) *string { return &s }

func TestResolveCopySource_AMIIDFromDifferentMinor(t *testing.T) {
	s := &stubEC2{
		images: []ec2types.Image{
			{
				ImageId:      strp("ami-old-minor"),
				Name:         strp("keights-v1.9.3-k8s-1.30.1-20250101T000000Z"),
				OwnerId:      strp(intami.OfficialOwnerID),
				CreationDate: strp("2025-01-01T00:00:00Z"),
			},
			{
				ImageId:      strp("ami-current"),
				Name:         strp("keights-v2.0.5-k8s-1.34.5-20260427T120000Z"),
				OwnerId:      strp(intami.OfficialOwnerID),
				CreationDate: strp("2026-04-27T12:00:00Z"),
			},
		},
	}
	got, err := resolveCopySource(context.Background(), s, "v2.0.5", "ami-old-minor")
	require.NoError(t, err)
	assert.Equal(t, "ami-old-minor", got.ID,
		"--ami-id should select across minors, not gated by running version",
	)
	assert.Equal(t, []string{"ami-old-minor"}, s.lastIn.ImageIds,
		"explicit --ami-id should look up by ID, not pattern-search",
	)
	assert.Empty(t, s.lastIn.Filters,
		"the by-ID lookup should not pass any name filter",
	)
}

func TestResolveCopySource_AMIIDFromWrongAccount(t *testing.T) {
	s := &stubEC2{
		images: []ec2types.Image{
			{
				ImageId:      strp("ami-foreign"),
				Name:         strp("keights-v2.0.5-k8s-1.34.5-20260427T120000Z"),
				OwnerId:      strp("999999999999"),
				CreationDate: strp("2026-04-27T12:00:00Z"),
			},
		},
	}
	_, err := resolveCopySource(context.Background(), s, "v2.0.5", "ami-foreign")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "official keights account")
}

func TestResolveCopySource_DefaultPicksLatestMatchingMinor(t *testing.T) {
	s := &stubEC2{
		images: []ec2types.Image{
			{
				ImageId:      strp("ami-newer"),
				Name:         strp("keights-v2.0.5-k8s-1.34.5-20260427T120000Z"),
				OwnerId:      strp(intami.OfficialOwnerID),
				CreationDate: strp("2026-04-27T12:00:00Z"),
			},
			{
				ImageId:      strp("ami-older"),
				Name:         strp("keights-v2.0.4-k8s-1.34.3-20260101T000000Z"),
				OwnerId:      strp(intami.OfficialOwnerID),
				CreationDate: strp("2026-01-01T00:00:00Z"),
			},
		},
	}
	got, err := resolveCopySource(context.Background(), s, "v2.0.5", "")
	require.NoError(t, err)
	assert.Equal(t, "ami-newer", got.ID)
	assert.Equal(t, []string{"keights-v2.0.*"}, s.lastIn.Filters[0].Values,
		"no --ami-id keeps the running-minor filter",
	)
}

func TestResolveCopySource_AMIIDNotFound(t *testing.T) {
	s := &stubEC2{
		images: []ec2types.Image{
			{
				ImageId:      strp("ami-real"),
				Name:         strp("keights-v2.0.5-k8s-1.34.5-20260427T120000Z"),
				OwnerId:      strp(intami.OfficialOwnerID),
				CreationDate: strp("2026-04-27T12:00:00Z"),
			},
		},
	}
	_, err := resolveCopySource(context.Background(), s, "v2.0.5", "ami-missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ami-missing")
	assert.Contains(t, err.Error(), "not found")
}
