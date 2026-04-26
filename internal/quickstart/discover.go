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

package quickstart

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

// VPC describes a VPC for selection in the wizard.
type VPC struct {
	ID   string
	CIDR string
	Name string
}

func (v VPC) Label() string {
	if v.Name != "" {
		return fmt.Sprintf("%s  %s  (%s)", v.ID, v.CIDR, v.Name)
	}
	return fmt.Sprintf("%s  %s", v.ID, v.CIDR)
}

// Subnet describes a subnet for selection in the wizard.
type Subnet struct {
	ID   string
	AZ   string
	CIDR string
	Name string
}

func (s Subnet) Label() string {
	if s.Name != "" {
		return fmt.Sprintf("%s  %s  %s  (%s)", s.ID, s.AZ, s.CIDR, s.Name)
	}
	return fmt.Sprintf("%s  %s  %s", s.ID, s.AZ, s.CIDR)
}

// KeyPair describes an EC2 key pair for selection in the wizard.
type KeyPair struct {
	Name string
}

// AMI describes a keights AMI candidate for selection in the wizard.
type AMI struct {
	ID                string
	Name              string
	OwnerID           string
	CreationDate      string
	ContainerImage    string
	KubernetesVersion string
}

func (a AMI) Label() string {
	from := a.ContainerImage
	if from == "" {
		from = "unknown"
	}
	k8s := a.KubernetesVersion
	if k8s == "" {
		k8s = "unknown"
	}
	return fmt.Sprintf("id: %s, from: %s, kubernetes: %s", a.ID, from, k8s)
}

// AMI tag keys used by the AMI build pipeline and read by tooling here.
const (
	// kubernetesVersionTag records which Kubernetes patch the AMI was built
	// for. Used to render the kubernetes_version into Terraform.
	kubernetesVersionTag = "cloudboss.co/keights/kubernetes-version"

	// keightsVersionMinorTag is the minor-scoped compatibility filter, e.g.,
	// "v2.0". A keights v2.0.x CLI only sees AMIs blessed for v2.0.
	keightsVersionMinorTag = "cloudboss.co/keights/keights-version-minor"

	// containerImageTag is the easyto-set tag carrying the source container
	// image ref (e.g., ghcr.io/cloudboss/keights:...).
	containerImageTag = "cloudboss.co/easyto/container-image"
)

// KMSKey describes a KMS key alias for selection in the wizard.
type KMSKey struct {
	AliasName   string
	TargetKeyID string
}

func (k KMSKey) Label() string {
	if k.TargetKeyID != "" {
		return fmt.Sprintf("%s  (%s)", k.AliasName, k.TargetKeyID)
	}
	return k.AliasName
}

// EC2API is the subset of the EC2 client the discoverer needs.
type EC2API interface {
	DescribeVpcs(ctx context.Context, in *ec2.DescribeVpcsInput,
		opts ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DescribeSubnets(ctx context.Context, in *ec2.DescribeSubnetsInput,
		opts ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	DescribeKeyPairs(ctx context.Context, in *ec2.DescribeKeyPairsInput,
		opts ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error)
	DescribeImages(ctx context.Context, in *ec2.DescribeImagesInput,
		opts ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
}

// KMSAPI is the subset of the KMS client the discoverer needs.
type KMSAPI interface {
	ListAliases(ctx context.Context, in *kms.ListAliasesInput,
		opts ...func(*kms.Options)) (*kms.ListAliasesOutput, error)
	ListKeys(ctx context.Context, in *kms.ListKeysInput,
		opts ...func(*kms.Options)) (*kms.ListKeysOutput, error)
	DescribeKey(ctx context.Context, in *kms.DescribeKeyInput,
		opts ...func(*kms.Options)) (*kms.DescribeKeyOutput, error)
}

// Discoverer loads selectable AWS resources for the wizard.
type Discoverer struct {
	ec2 EC2API
	kms KMSAPI
	// keightsVersion is the running CLI version (e.g., "v2.0.5"), used to
	// scope AMI discovery to the matching keights minor. Empty disables the
	// minor filter (e.g., for dev builds where Version == "dev").
	keightsVersion string
}

func NewDiscoverer(ec2Client EC2API, kmsClient KMSAPI, keightsVersion string) *Discoverer {
	return &Discoverer{
		ec2:            ec2Client,
		kms:            kmsClient,
		keightsVersion: keightsVersion,
	}
}

// keightsMinor returns the vMAJOR.MINOR prefix of a vMAJOR.MINOR.PATCH string,
// e.g., "v2.0.5" -> "v2.0". Returns "" for inputs that aren't a vMAJOR.MINOR.*
// shape (such as "dev"), in which case callers should skip minor filtering.
func keightsMinor(version string) string {
	if !strings.HasPrefix(version, "v") {
		return ""
	}
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "." + parts[1]
}

func (d *Discoverer) VPCs(ctx context.Context) ([]VPC, error) {
	out, err := d.ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, fmt.Errorf("unable to describe VPCs: %w", err)
	}
	vpcs := make([]VPC, 0, len(out.Vpcs))
	for _, v := range out.Vpcs {
		vpcs = append(vpcs, VPC{
			ID:   aws(v.VpcId),
			CIDR: aws(v.CidrBlock),
			Name: tagValue(v.Tags, "Name"),
		})
	}
	sort.Slice(vpcs, func(i, j int) bool { return vpcs[i].ID < vpcs[j].ID })
	return vpcs, nil
}

func (d *Discoverer) Subnets(ctx context.Context, vpcID string) ([]Subnet, error) {
	out, err := d.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{
			{Name: strPtr("vpc-id"), Values: []string{vpcID}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to describe subnets: %w", err)
	}
	subs := make([]Subnet, 0, len(out.Subnets))
	for _, s := range out.Subnets {
		subs = append(subs, Subnet{
			ID:   aws(s.SubnetId),
			AZ:   aws(s.AvailabilityZone),
			CIDR: aws(s.CidrBlock),
			Name: tagValue(s.Tags, "Name"),
		})
	}
	sort.Slice(subs, func(i, j int) bool {
		if subs[i].AZ != subs[j].AZ {
			return subs[i].AZ < subs[j].AZ
		}
		return subs[i].ID < subs[j].ID
	})
	return subs, nil
}

// AMIs returns keights AMIs visible to the caller. Filters on the easyto
// container-image tag and, when the running CLI has a release-shaped version,
// on the keights-version-minor tag, so a v2.0.x CLI only sees AMIs blessed
// for v2.0.
func (d *Discoverer) AMIs(ctx context.Context) ([]AMI, error) {
	filters := []ec2types.Filter{
		{
			Name:   strPtr("tag:" + containerImageTag),
			Values: []string{"ghcr.io/cloudboss/keights*"},
		},
	}
	if minor := keightsMinor(d.keightsVersion); minor != "" {
		filters = append(filters, ec2types.Filter{
			Name:   strPtr("tag:" + keightsVersionMinorTag),
			Values: []string{minor},
		})
	}
	out, err := d.ec2.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Owners:  []string{"self"},
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to describe images: %w", err)
	}
	amis := make([]AMI, 0, len(out.Images))
	for _, img := range out.Images {
		ci := tagValue(img.Tags, containerImageTag)
		if !strings.HasPrefix(ci, "ghcr.io/cloudboss/keights") {
			continue
		}
		amis = append(amis, AMI{
			ID:                aws(img.ImageId),
			Name:              aws(img.Name),
			OwnerID:           aws(img.OwnerId),
			CreationDate:      aws(img.CreationDate),
			ContainerImage:    ci,
			KubernetesVersion: tagValue(img.Tags, kubernetesVersionTag),
		})
	}
	sort.Slice(amis, func(i, j int) bool {
		return amis[i].CreationDate > amis[j].CreationDate
	})
	return amis, nil
}

// KMSKeys returns customer-managed KMS keys the caller can see. Keys without
// an alias are included; AWS-managed aliases (`alias/aws/*`) are excluded.
// When a key has multiple aliases, the first one wins.
func (d *Discoverer) KMSKeys(ctx context.Context) ([]KMSKey, error) {
	aliasByKey, err := d.listAliases(ctx)
	if err != nil {
		return nil, err
	}

	keys := []KMSKey{}
	seen := map[string]bool{}
	var marker *string
	for {
		out, err := d.kms.ListKeys(ctx, &kms.ListKeysInput{Marker: marker})
		if err != nil {
			return nil, fmt.Errorf("unable to list KMS keys: %w", err)
		}
		for _, k := range out.Keys {
			keyID := aws(k.KeyId)
			if seen[keyID] {
				continue
			}
			seen[keyID] = true
			customer, err := d.isCustomerManaged(ctx, keyID)
			if err != nil {
				return nil, err
			}
			if !customer {
				continue
			}
			keys = append(keys, KMSKey{
				AliasName:   aliasByKey[keyID],
				TargetKeyID: keyID,
			})
		}
		if out.Truncated && out.NextMarker != nil {
			marker = out.NextMarker
			continue
		}
		break
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].AliasName != keys[j].AliasName {
			if keys[i].AliasName == "" {
				return false
			}
			if keys[j].AliasName == "" {
				return true
			}
			return keys[i].AliasName < keys[j].AliasName
		}
		return keys[i].TargetKeyID < keys[j].TargetKeyID
	})
	return keys, nil
}

func (d *Discoverer) listAliases(ctx context.Context) (map[string]string, error) {
	out := map[string]string{}
	var marker *string
	for {
		resp, err := d.kms.ListAliases(ctx, &kms.ListAliasesInput{Marker: marker})
		if err != nil {
			return nil, fmt.Errorf("unable to list KMS aliases: %w", err)
		}
		for _, a := range resp.Aliases {
			name := aws(a.AliasName)
			if strings.HasPrefix(name, "alias/aws/") {
				continue
			}
			if a.TargetKeyId == nil {
				continue
			}
			keyID := aws(a.TargetKeyId)
			if _, exists := out[keyID]; !exists {
				out[keyID] = name
			}
		}
		if resp.Truncated && resp.NextMarker != nil {
			marker = resp.NextMarker
			continue
		}
		break
	}
	return out, nil
}

func (d *Discoverer) isCustomerManaged(ctx context.Context, keyID string) (bool, error) {
	out, err := d.kms.DescribeKey(ctx, &kms.DescribeKeyInput{KeyId: strPtr(keyID)})
	if err != nil {
		return false, fmt.Errorf("unable to describe KMS key %s: %w", keyID, err)
	}
	md := out.KeyMetadata
	if md == nil {
		return false, nil
	}
	if md.KeyManager != "CUSTOMER" {
		return false, nil
	}
	if md.KeyState != "Enabled" {
		return false, nil
	}
	return true, nil
}

func (d *Discoverer) KeyPairs(ctx context.Context) ([]KeyPair, error) {
	out, err := d.ec2.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{})
	if err != nil {
		return nil, fmt.Errorf("unable to describe key pairs: %w", err)
	}
	keys := make([]KeyPair, 0, len(out.KeyPairs))
	for _, k := range out.KeyPairs {
		keys = append(keys, KeyPair{Name: aws(k.KeyName)})
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Name < keys[j].Name })
	return keys, nil
}

func aws(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func strPtr(s string) *string { return &s }

func tagValue(tags []ec2types.Tag, key string) string {
	for _, t := range tags {
		if aws(t.Key) == key {
			return aws(t.Value)
		}
	}
	return ""
}
