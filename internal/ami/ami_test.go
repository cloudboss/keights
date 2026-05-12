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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeightsMinor(t *testing.T) {
	cases := []struct{ in, want string }{
		{"v2.0.5", "v2.0"},
		{"v10.20.30", "v10.20"},
		{"v2.0", "v2.0"},
		{"dev", ""},
		{"", ""},
		{"2.0.5", ""},
		{"v2", ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, KeightsMinor(c.in), "KeightsMinor(%q)", c.in)
	}
}

// fakeEC2 is a hand-rolled stand-in for an ec2.Client. It implements
// ami.EC2API so the same fake covers both Find and Copy tests.
type fakeEC2 struct {
	images []ec2types.Image
	lastIn *ec2.DescribeImagesInput
	copyIn *ec2.CopyImageInput
	newID  string
}

func (f *fakeEC2) DescribeImages(
	_ context.Context, in *ec2.DescribeImagesInput, _ ...func(*ec2.Options),
) (*ec2.DescribeImagesOutput, error) {
	f.lastIn = in
	if len(in.ImageIds) > 0 {
		ids := map[string]bool{}
		for _, id := range in.ImageIds {
			ids[id] = true
		}
		var matched []ec2types.Image
		for _, img := range f.images {
			if img.ImageId != nil && ids[*img.ImageId] {
				matched = append(matched, img)
			}
		}
		return &ec2.DescribeImagesOutput{Images: matched}, nil
	}
	return &ec2.DescribeImagesOutput{Images: f.images}, nil
}

func (f *fakeEC2) CopyImage(
	_ context.Context, in *ec2.CopyImageInput, _ ...func(*ec2.Options),
) (*ec2.CopyImageOutput, error) {
	f.copyIn = in
	return &ec2.CopyImageOutput{ImageId: p(f.newID)}, nil
}

func TestFind_FiltersByNameAndParsesNameForVersions(t *testing.T) {
	f := &fakeEC2{
		images: []ec2types.Image{
			{
				ImageId:      p("ami-1"),
				Name:         p("keights-v2.0.5-k8s-v1.34.5-20260427T120000Z"),
				OwnerId:      p("256008164056"),
				CreationDate: p("2026-04-27T12:00:00Z"),
			},
			{
				ImageId:      p("ami-2"),
				Name:         p("keights-v2.0.4-k8s-v1.34.3-20260301T120000Z"),
				OwnerId:      p("123456789012"),
				CreationDate: p("2026-03-01T12:00:00Z"),
			},
			{
				ImageId: p("ami-3"),
				Name:    p("some-unrelated-ami"),
			},
		},
	}
	got, err := Find(
		context.Background(), f, "us-east-1", "v2.0.5",
		[]string{"self", OfficialOwnerID},
	)
	require.NoError(t, err)
	require.Len(t, got, 2, "non-keights name should be skipped")

	require.Len(t, f.lastIn.Filters, 1)
	assert.Equal(t, "name", *f.lastIn.Filters[0].Name)
	assert.Equal(t, []string{"keights-v2.0.*"}, f.lastIn.Filters[0].Values)
	assert.Equal(t, []string{"self", OfficialOwnerID}, f.lastIn.Owners)

	assert.Equal(t, "ami-1", got[0].ID, "newest first")
	assert.Equal(t, "ami-2", got[1].ID)

	assert.Equal(t, "1.34.5", got[0].KubernetesVersion)
	assert.Equal(t, "v2.0", got[0].KeightsVersionMinor)
	assert.Equal(t, "us-east-1", got[0].Region)
	assert.Equal(t, "256008164056", got[0].OwnerID)
}

func TestFind_DevVersionUsesUnboundedNameFilter(t *testing.T) {
	f := &fakeEC2{}
	_, err := Find(context.Background(), f, "us-east-1", "dev", nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"keights-*"}, f.lastIn.Filters[0].Values)
}

func TestCopy_TagsImageAndSnapshotsViaTagSpecifications(t *testing.T) {
	src := AMI{
		ID:                  "ami-src",
		Name:                "keights-v2.0.5-k8s-v1.34.5-20260427T120000Z",
		KubernetesVersion:   "1.34.5",
		KeightsVersion:      "v2.0.5",
		KeightsVersionMinor: "v2.0",
	}
	f := &fakeEC2{newID: "ami-new"}
	newID, err := Copy(
		context.Background(), f, src, "us-east-1", "custom-name", false,
	)
	require.NoError(t, err)
	assert.Equal(t, "ami-new", newID)

	require.NotNil(t, f.copyIn)
	assert.Equal(t, "us-east-1", *f.copyIn.SourceRegion)
	assert.Equal(t, "ami-src", *f.copyIn.SourceImageId)
	assert.Equal(t, "custom-name", *f.copyIn.Name)

	require.Len(t, f.copyIn.TagSpecifications, 2,
		"one TagSpecification for image, one for snapshot",
	)
	want := map[string]string{
		KubernetesVersionTag:   "1.34.5",
		KeightsVersionTag:      "v2.0.5",
		KeightsVersionMinorTag: "v2.0",
	}
	byType := map[ec2types.ResourceType]map[string]string{}
	for _, ts := range f.copyIn.TagSpecifications {
		byType[ts.ResourceType] = tagMap(ts.Tags)
	}
	assert.Equal(t, want, byType[ec2types.ResourceTypeImage])
	assert.Equal(t, want, byType[ec2types.ResourceTypeSnapshot])
}

func TestCopy_DefaultNameComesFromSource(t *testing.T) {
	src := AMI{
		ID:                  "ami-src",
		Name:                "keights-v2.0.5-k8s-v1.34.5-20260427T120000Z",
		KubernetesVersion:   "1.34.5",
		KeightsVersion:      "v2.0.5",
		KeightsVersionMinor: "v2.0",
	}
	f := &fakeEC2{newID: "ami-new"}
	_, err := Copy(context.Background(), f, src, "us-east-1", "", false)
	require.NoError(t, err)
	assert.Equal(t, src.Name, *f.copyIn.Name)
}

func TestGet_FetchesByIDAndParsesName(t *testing.T) {
	f := &fakeEC2{
		images: []ec2types.Image{
			{
				ImageId:      p("ami-1"),
				Name:         p("keights-v2.0.5-k8s-v1.34.5-20260427T120000Z"),
				OwnerId:      p(OfficialOwnerID),
				CreationDate: p("2026-04-27T12:00:00Z"),
			},
		},
	}
	got, err := Get(context.Background(), f, "us-east-1", "ami-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"ami-1"}, f.lastIn.ImageIds)
	assert.Empty(t, f.lastIn.Filters, "Get must not apply a name filter")
	assert.Equal(t, "ami-1", got.ID)
	assert.Equal(t, "1.34.5", got.KubernetesVersion)
	assert.Equal(t, "v2.0.5", got.KeightsVersion)
	assert.Equal(t, "v2.0", got.KeightsVersionMinor)
}

func TestGet_LegacyNameWithoutVOnK8s(t *testing.T) {
	f := &fakeEC2{
		images: []ec2types.Image{
			{
				ImageId:      p("ami-legacy"),
				Name:         p("keights-v2.0.5-k8s-1.34.5-20260427T120000Z"),
				OwnerId:      p(OfficialOwnerID),
				CreationDate: p("2026-04-27T12:00:00Z"),
			},
		},
	}
	got, err := Get(context.Background(), f, "us-east-1", "ami-legacy")
	require.NoError(t, err)
	assert.Equal(t, "1.34.5", got.KubernetesVersion)
	assert.Equal(t, "v2.0.5", got.KeightsVersion)
}

func TestLookupByName_KeightsConventionPopulatesVersions(t *testing.T) {
	f := &fakeEC2{
		images: []ec2types.Image{
			{
				ImageId:      p("ami-1"),
				Name:         p("keights-v2.0.5-k8s-v1.34.5-20260427T120000Z"),
				OwnerId:      p(OfficialOwnerID),
				CreationDate: p("2026-04-27T12:00:00Z"),
			},
		},
	}
	got, ok, err := LookupByName(
		context.Background(), f, "us-east-1",
		"keights-v2.0.5-k8s-v1.34.5-20260427T120000Z",
		[]string{"self", OfficialOwnerID},
	)
	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, f.lastIn.Filters, 1)
	assert.Equal(t, "name", *f.lastIn.Filters[0].Name)
	assert.Equal(t, "ami-1", got.ID)
	assert.Equal(t, OfficialOwnerID, got.OwnerID)
	assert.Equal(t, "1.34.5", got.KubernetesVersion)
	assert.Equal(t, "v2.0.5", got.KeightsVersion)
}

func TestLookupByName_NonKeightsNameSucceedsWithEmptyVersions(t *testing.T) {
	f := &fakeEC2{
		images: []ec2types.Image{
			{
				ImageId:      p("ami-custom"),
				Name:         p("ghcr.io--cloudboss--keights--v2.0.1-alpha.1-k8s-1.34.5"),
				OwnerId:      p("123456789012"),
				CreationDate: p("2026-04-27T12:00:00Z"),
			},
		},
	}
	got, ok, err := LookupByName(
		context.Background(), f, "us-east-1",
		"ghcr.io--cloudboss--keights--v2.0.1-alpha.1-k8s-1.34.5",
		[]string{"self", OfficialOwnerID},
	)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "ami-custom", got.ID)
	assert.Equal(t, "123456789012", got.OwnerID)
	assert.Empty(t, got.KubernetesVersion)
	assert.Empty(t, got.KeightsVersion)
}

func TestLookupByName_NotFound(t *testing.T) {
	f := &fakeEC2{}
	_, ok, err := LookupByName(
		context.Background(), f, "us-east-1", "missing",
		[]string{"self", OfficialOwnerID},
	)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestGet_NotFound(t *testing.T) {
	f := &fakeEC2{}
	_, err := Get(context.Background(), f, "us-east-1", "ami-nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGet_BadName(t *testing.T) {
	f := &fakeEC2{
		images: []ec2types.Image{
			{ImageId: p("ami-x"), Name: p("not-a-keights-ami")},
		},
	}
	_, err := Get(context.Background(), f, "us-east-1", "ami-x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "naming convention")
}

func TestTagsForCopy_BuildsKeightsTagsFromParsedFields(t *testing.T) {
	src := AMI{
		KubernetesVersion:   "1.34.5",
		KeightsVersion:      "v2.0.5",
		KeightsVersionMinor: "v2.0",
	}
	assert.Equal(t,
		map[string]string{
			KubernetesVersionTag:   "1.34.5",
			KeightsVersionTag:      "v2.0.5",
			KeightsVersionMinorTag: "v2.0",
		},
		tagMap(tagsForCopy(src)),
	)
}

func TestTagsForCopy_SkipsEmptyValues(t *testing.T) {
	out := tagsForCopy(AMI{KubernetesVersion: "1.34.5"})
	assert.Equal(t,
		map[string]string{KubernetesVersionTag: "1.34.5"},
		tagMap(out),
	)
}

func tagMap(tags []ec2types.Tag) map[string]string {
	m := map[string]string{}
	for _, t := range tags {
		if t.Key != nil && t.Value != nil {
			m[*t.Key] = *t.Value
		}
	}
	return m
}
