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
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

const (
	// OfficialOwnerID is the AWS account where official keights AMIs are
	// published.
	OfficialOwnerID = "256008164056"

	// OfficialRegion is the AWS region where official keights AMIs are
	// published. Copies to other regions are made via CopyImage.
	OfficialRegion = "us-east-1"

	// Keights tag keys applied by hack/ami-build. The CLI does not read
	// these (cross-account public AMIs hide tags from non-owners), but it
	// does write them back onto copies so the user's local copies carry
	// the same metadata as the maintainer's originals.
	KubernetesVersionTag   = "cloudboss.co/keights/kubernetes-version"
	KeightsVersionTag      = "cloudboss.co/keights/keights-version"
	KeightsVersionMinorTag = "cloudboss.co/keights/keights-version-minor"
)

// keightsAMINameRE matches the AMI naming convention enforced by hack/ami-build:
// "keights-vMAJOR.MINOR.PATCH-k8s-MAJOR.MINOR.PATCH-TIMESTAMP". The first two
// groups are the keights and kubernetes versions; we rely on the AMI name to
// carry these because AWS does not expose AMI tags to accounts other than the
// owner, even on public AMIs.
var keightsAMINameRE = regexp.MustCompile(
	`^keights-(v\d+\.\d+\.\d+)-k8s-(\d+\.\d+\.\d+)-`,
)

// AMI describes a keights AMI candidate.
type AMI struct {
	ID                  string
	Name                string
	OwnerID             string
	CreationDate        string
	KubernetesVersion   string
	KeightsVersion      string
	KeightsVersionMinor string
	Region              string
}

func (a AMI) Label() string {
	k8s := a.KubernetesVersion
	if k8s == "" {
		k8s = "unknown"
	}
	return fmt.Sprintf("id: %s, kubernetes: %s", a.ID, k8s)
}

// EC2API is the subset of the EC2 client used by Find and Copy.
type EC2API interface {
	DescribeImages(ctx context.Context, in *ec2.DescribeImagesInput,
		opts ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
	CopyImage(ctx context.Context, in *ec2.CopyImageInput,
		opts ...func(*ec2.Options)) (*ec2.CopyImageOutput, error)
}

// KeightsMinor returns the vMAJOR.MINOR prefix of a vMAJOR.MINOR.PATCH string,
// e.g., "v2.0.5" -> "v2.0". Returns "" for inputs that aren't a vMAJOR.MINOR.*
// shape (such as "dev"), in which case callers should skip minor filtering.
func KeightsMinor(version string) string {
	if !strings.HasPrefix(version, "v") {
		return ""
	}
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "." + parts[1]
}

// Find returns keights AMIs visible to the caller. Filters by AMI name --
// "keights-*" or, when version is release-shaped, "keights-vMAJOR.MINOR.*".
// AWS does not expose AMI tags to accounts other than the owner, even on
// public AMIs, so the keights and kubernetes versions are parsed from the
// name itself. Returned AMIs have Region set to the supplied region so
// callers merging results across regions can render it.
func Find(
	ctx context.Context, c EC2API, region, version string, owners []string,
) ([]AMI, error) {
	namePattern := "keights-*"
	if minor := KeightsMinor(version); minor != "" {
		namePattern = "keights-" + minor + ".*"
	}
	out, err := c.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Owners: owners,
		Filters: []ec2types.Filter{
			{Name: p("name"), Values: []string{namePattern}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to describe images: %w", err)
	}
	amis := make([]AMI, 0, len(out.Images))
	for _, img := range out.Images {
		if a, ok := imageToAMI(img, region); ok {
			amis = append(amis, a)
		}
	}
	sort.Slice(amis, func(i, j int) bool {
		return amis[i].CreationDate > amis[j].CreationDate
	})
	return amis, nil
}

// Get fetches a single keights AMI by ID. Returns an error if the AMI does
// not exist or its name does not match the keights naming convention.
func Get(ctx context.Context, c EC2API, region, amiID string) (AMI, error) {
	out, err := c.DescribeImages(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{amiID},
	})
	if err != nil {
		return AMI{}, fmt.Errorf("unable to describe %s: %w", amiID, err)
	}
	if len(out.Images) == 0 {
		return AMI{}, fmt.Errorf("ami %s not found in %s", amiID, region)
	}
	a, ok := imageToAMI(out.Images[0], region)
	if !ok {
		return AMI{}, fmt.Errorf(
			"ami %s does not match the keights naming convention", amiID,
		)
	}
	return a, nil
}

// imageToAMI parses an EC2 image into a keights AMI. ok is false when the
// image's name does not match the convention enforced by hack/ami-build.
func imageToAMI(img ec2types.Image, region string) (AMI, bool) {
	name := deref(img.Name)
	match := keightsAMINameRE.FindStringSubmatch(name)
	if match == nil {
		return AMI{}, false
	}
	keightsFull, k8s := match[1], match[2]
	return AMI{
		ID:                  deref(img.ImageId),
		Name:                name,
		OwnerID:             deref(img.OwnerId),
		CreationDate:        deref(img.CreationDate),
		KubernetesVersion:   k8s,
		KeightsVersion:      keightsFull,
		KeightsVersionMinor: KeightsMinor(keightsFull),
		Region:              region,
	}, true
}

// Copy invokes EC2 CopyImage to copy src from srcRegion into dst's region,
// applying keights identifying tags to the new AMI and its snapshots in the
// same call via TagSpecifications. Returns the new AMI ID. If wait is true,
// blocks until the copied AMI reaches state "available".
func Copy(
	ctx context.Context, dst EC2API, src AMI, srcRegion, name string, wait bool,
) (string, error) {
	if name == "" {
		name = src.Name
	}
	in := &ec2.CopyImageInput{
		SourceRegion:  p(srcRegion),
		SourceImageId: p(src.ID),
		Name:          p(name),
		Description:   p("Copied from " + src.ID + " in " + srcRegion),
	}
	if tags := tagsForCopy(src); len(tags) > 0 {
		in.TagSpecifications = []ec2types.TagSpecification{
			{ResourceType: ec2types.ResourceTypeImage, Tags: tags},
			{ResourceType: ec2types.ResourceTypeSnapshot, Tags: tags},
		}
	}
	out, err := dst.CopyImage(ctx, in)
	if err != nil {
		return "", fmt.Errorf("unable to copy image: %w", err)
	}
	newID := deref(out.ImageId)
	if wait {
		if err := waitAvailable(ctx, dst, newID); err != nil {
			return newID, err
		}
	}
	return newID, nil
}

// tagsForCopy returns the keights identifying tags reconstructed from the
// source AMI's parsed fields. We do not propagate the source's actual tags
// because cross-account public AMIs hide their tags from non-owners, so
// src.Tags is empty in the common copy path.
func tagsForCopy(src AMI) []ec2types.Tag {
	tags := []ec2types.Tag{}
	if src.KubernetesVersion != "" {
		tags = append(tags, ec2types.Tag{
			Key:   p(KubernetesVersionTag),
			Value: p(src.KubernetesVersion),
		})
	}
	if src.KeightsVersion != "" {
		tags = append(tags, ec2types.Tag{
			Key:   p(KeightsVersionTag),
			Value: p(src.KeightsVersion),
		})
	}
	if src.KeightsVersionMinor != "" {
		tags = append(tags, ec2types.Tag{
			Key:   p(KeightsVersionMinorTag),
			Value: p(src.KeightsVersionMinor),
		})
	}
	return tags
}

func waitAvailable(ctx context.Context, c EC2API, amiID string) error {
	deadline := time.Now().Add(30 * time.Minute)
	for {
		out, err := c.DescribeImages(ctx, &ec2.DescribeImagesInput{
			ImageIds: []string{amiID},
		})
		if err != nil {
			return fmt.Errorf("unable to describe %s: %w", amiID, err)
		}
		if len(out.Images) > 0 {
			state := string(out.Images[0].State)
			switch state {
			case "available":
				return nil
			case "failed", "invalid", "deregistered", "error":
				return fmt.Errorf(
					"ami %s reached terminal state %s", amiID, state,
				)
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for ami %s", amiID)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(15 * time.Second):
		}
	}
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

func p[T any](v T) *T {
	return &v
}
