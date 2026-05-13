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

// Package validate runs preflight checks against an AWS account to confirm
// it is set up for a keights cluster.
package validate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/cloudboss/keights/internal/ami"
)

// EC2API is the subset of the EC2 client used by validators.
type EC2API interface {
	ami.EC2API
	DescribeVpcs(ctx context.Context, in *ec2.DescribeVpcsInput,
		opts ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DescribeSubnets(ctx context.Context, in *ec2.DescribeSubnetsInput,
		opts ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
}

// Result describes the outcome of one check. Remediation is empty when OK.
type Result struct {
	Name        string
	OK          bool
	Detail      string
	Remediation string
}

// CheckVPC verifies that vpcID refers to an existing VPC.
func CheckVPC(ctx context.Context, c EC2API, vpcID string) (Result, error) {
	out, err := c.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	})
	if err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) && ae.ErrorCode() == "InvalidVpcID.NotFound" {
			return Result{
				Name:   "vpc",
				OK:     false,
				Detail: fmt.Sprintf("VPC %s does not exist.", vpcID),
			}, nil
		}
		return Result{}, fmt.Errorf("describe vpc %s: %w", vpcID, err)
	}
	if len(out.Vpcs) == 0 {
		return Result{
			Name:   "vpc",
			OK:     false,
			Detail: fmt.Sprintf("VPC %s does not exist.", vpcID),
		}, nil
	}
	return Result{Name: "vpc", OK: true}, nil
}

// CheckAMI verifies that at least one keights AMI is visible to the caller
// in the configured region. It looks for AMIs owned by the caller and the
// official keights publisher account. Outside us-east-1, the remediation
// directs the user to copy the official AMI.
func CheckAMI(
	ctx context.Context, c EC2API, region, keightsVersion string,
) (Result, error) {
	amis, err := ami.Find(
		ctx, c, region, keightsVersion,
		[]string{"self", ami.OfficialOwnerID},
	)
	if err != nil {
		return Result{}, err
	}
	if len(amis) > 0 {
		return Result{Name: "ami", OK: true}, nil
	}

	versionLabel := "any keights version"
	if minor := ami.KeightsMinor(keightsVersion); minor != "" {
		versionLabel = "keights " + minor
	}
	if region == ami.OfficialRegion {
		return Result{
			Name:   "ami",
			OK:     false,
			Detail: fmt.Sprintf("No AMI for %s found in %s.", versionLabel, region),
			Remediation: fmt.Sprintf(
				"  keights ami list --region %s\n", region,
			),
		}, nil
	}
	return Result{
		Name: "ami",
		OK:   false,
		Detail: fmt.Sprintf(
			"No AMI for %s found in %s, copy from %s first.",
			versionLabel, region, ami.OfficialRegion,
		),
		Remediation: fmt.Sprintf(
			"  keights ami copy --region %s\n", region,
		),
	}, nil
}

// CheckSubnets verifies the given subnet IDs all exist in vpcID.
func CheckSubnets(
	ctx context.Context, c EC2API, vpcID string, subnetIDs []string,
) (Result, error) {
	out, err := c.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{
			{Name: strPtr("vpc-id"), Values: []string{vpcID}},
			{Name: strPtr("subnet-id"), Values: subnetIDs},
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("describe subnets in %s: %w", vpcID, err)
	}
	found := make(map[string]bool, len(out.Subnets))
	for _, s := range out.Subnets {
		if s.SubnetId != nil {
			found[*s.SubnetId] = true
		}
	}
	var missing []string
	for _, id := range subnetIDs {
		if !found[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return Result{Name: "subnets", OK: true}, nil
	}
	s := ""
	if len(missing) > 1 {
		s = "s"
	}
	return Result{
		Name: "subnets",
		OK:   false,
		Detail: fmt.Sprintf("Subnet%s not found in VPC %s: %s.",
			s, vpcID, strings.Join(missing, ", ")),
	}, nil
}

func strPtr(s string) *string { return &s }

// PrintResults writes each result to w and returns whether all passed.
// Failed results are followed by their remediation block.
func PrintResults(w io.Writer, results []Result) bool {
	allOK := true
	for _, r := range results {
		status := "OK"
		if !r.OK {
			status = "FAIL"
			allOK = false
		}
		fmt.Fprintf(w, "%s: %s\n", r.Name, status)
		if r.OK {
			continue
		}
		if r.Detail != "" {
			fmt.Fprintf(w, "  %s\n", r.Detail)
		}
		if r.Remediation != "" {
			fmt.Fprintln(w, "  Remediation:")
			fmt.Fprint(w, r.Remediation)
		}
	}
	return allOK
}
