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

package validate

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubEC2 struct {
	vpcs    []ec2types.Vpc
	vpcsErr error

	images    []ec2types.Image
	imagesErr error

	subnets    []ec2types.Subnet
	subnetsErr error
}

func (s *stubEC2) DescribeVpcs(
	_ context.Context, _ *ec2.DescribeVpcsInput, _ ...func(*ec2.Options),
) (*ec2.DescribeVpcsOutput, error) {
	if s.vpcsErr != nil {
		return nil, s.vpcsErr
	}
	return &ec2.DescribeVpcsOutput{Vpcs: s.vpcs}, nil
}

func (s *stubEC2) DescribeImages(
	_ context.Context, _ *ec2.DescribeImagesInput, _ ...func(*ec2.Options),
) (*ec2.DescribeImagesOutput, error) {
	if s.imagesErr != nil {
		return nil, s.imagesErr
	}
	return &ec2.DescribeImagesOutput{Images: s.images}, nil
}

func (*stubEC2) CopyImage(
	context.Context, *ec2.CopyImageInput, ...func(*ec2.Options),
) (*ec2.CopyImageOutput, error) {
	return nil, nil
}

func (s *stubEC2) DescribeSubnets(
	_ context.Context, _ *ec2.DescribeSubnetsInput, _ ...func(*ec2.Options),
) (*ec2.DescribeSubnetsOutput, error) {
	if s.subnetsErr != nil {
		return nil, s.subnetsErr
	}
	return &ec2.DescribeSubnetsOutput{Subnets: s.subnets}, nil
}

func strp(s string) *string { return &s }

func TestCheckAMI(t *testing.T) {
	matching := ec2types.Image{
		ImageId:      strp("ami-1"),
		Name:         strp("keights-v2.0.5-k8s-v1.34.5-20260427T000000Z"),
		OwnerId:      strp("256008164056"),
		CreationDate: strp("2026-04-27T00:00:00Z"),
	}
	tests := []struct {
		name            string
		region          string
		version         string
		images          []ec2types.Image
		wantOK          bool
		wantRemediation string
	}{
		{
			name:    "AMI present passes",
			region:  "us-west-2",
			version: "v2.0.5",
			images:  []ec2types.Image{matching},
			wantOK:  true,
		},
		{
			name:            "no AMI outside us-east-1 suggests copy",
			region:          "us-west-2",
			version:         "v2.0.5",
			images:          nil,
			wantOK:          false,
			wantRemediation: "keights ami copy --region us-west-2",
		},
		{
			name:            "no AMI in us-east-1 suggests list",
			region:          "us-east-1",
			version:         "v2.0.5",
			images:          nil,
			wantOK:          false,
			wantRemediation: "keights ami list --region us-east-1",
		},
		{
			name:    "version filter excludes non-matching minor",
			region:  "us-east-1",
			version: "v3.0.0",
			images:  []ec2types.Image{matching},
			wantOK:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &stubEC2{images: tt.images}
			r, err := CheckAMI(context.Background(), c, tt.region, tt.version)
			require.NoError(t, err)
			assert.Equal(t, "ami", r.Name)
			assert.Equal(t, tt.wantOK, r.OK)
			if tt.wantOK {
				return
			}
			assert.Contains(t, r.Remediation, tt.wantRemediation)
		})
	}
}

func TestCheckAMI_APIError(t *testing.T) {
	c := &stubEC2{imagesErr: errors.New("boom")}
	_, err := CheckAMI(context.Background(), c, "us-east-1", "v2.0.5")
	require.Error(t, err)
}

func TestCheckSubnets(t *testing.T) {
	tests := []struct {
		name        string
		want        []string
		inVPC       []ec2types.Subnet
		wantOK      bool
		wantMissing []string
	}{
		{
			name:   "all subnets in VPC passes",
			want:   []string{"subnet-1", "subnet-2"},
			inVPC:  []ec2types.Subnet{{SubnetId: strp("subnet-1")}, {SubnetId: strp("subnet-2")}},
			wantOK: true,
		},
		{
			name:        "one missing reports it",
			want:        []string{"subnet-1", "subnet-2"},
			inVPC:       []ec2types.Subnet{{SubnetId: strp("subnet-1")}},
			wantOK:      false,
			wantMissing: []string{"subnet-2"},
		},
		{
			name:        "none in VPC reports all",
			want:        []string{"subnet-1", "subnet-2"},
			inVPC:       nil,
			wantOK:      false,
			wantMissing: []string{"subnet-1", "subnet-2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &stubEC2{subnets: tt.inVPC}
			r, err := CheckSubnets(context.Background(), c, "vpc-abc", tt.want)
			require.NoError(t, err)
			assert.Equal(t, "subnets", r.Name)
			assert.Equal(t, tt.wantOK, r.OK)
			if tt.wantOK {
				return
			}
			assert.Contains(t, r.Detail, "vpc-abc")
			for _, m := range tt.wantMissing {
				assert.Contains(t, r.Detail, m)
			}
		})
	}
}

func TestCheckSubnets_APIError(t *testing.T) {
	c := &stubEC2{subnetsErr: errors.New("boom")}
	_, err := CheckSubnets(context.Background(), c, "vpc-abc", []string{"subnet-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vpc-abc")
}

type notFoundErr struct{}

func (notFoundErr) Error() string                 { return "InvalidVpcID.NotFound" }
func (notFoundErr) ErrorCode() string             { return "InvalidVpcID.NotFound" }
func (notFoundErr) ErrorMessage() string          { return "VPC not found" }
func (notFoundErr) ErrorFault() smithy.ErrorFault { return smithy.FaultClient }

func TestCheckVPC(t *testing.T) {
	t.Run("present passes", func(t *testing.T) {
		c := &stubEC2{vpcs: []ec2types.Vpc{{VpcId: strp("vpc-abc")}}}
		r, err := CheckVPC(context.Background(), c, "vpc-abc")
		require.NoError(t, err)
		assert.True(t, r.OK)
	})
	t.Run("InvalidVpcID.NotFound becomes a failure result", func(t *testing.T) {
		c := &stubEC2{vpcsErr: notFoundErr{}}
		r, err := CheckVPC(context.Background(), c, "vpc-abc")
		require.NoError(t, err)
		assert.False(t, r.OK)
		assert.Contains(t, r.Detail, "vpc-abc")
	})
	t.Run("empty result becomes a failure result", func(t *testing.T) {
		c := &stubEC2{}
		r, err := CheckVPC(context.Background(), c, "vpc-abc")
		require.NoError(t, err)
		assert.False(t, r.OK)
	})
	t.Run("other API error propagates", func(t *testing.T) {
		c := &stubEC2{vpcsErr: errors.New("boom")}
		_, err := CheckVPC(context.Background(), c, "vpc-abc")
		require.Error(t, err)
	})
}
