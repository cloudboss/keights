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

package nlb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

const clusterTag = "keights.cloudboss.co/cluster"

type ELBv2API interface {
	DescribeLoadBalancers(
		ctx context.Context,
		params *elbv2.DescribeLoadBalancersInput,
		optFns ...func(*elbv2.Options),
	) (*elbv2.DescribeLoadBalancersOutput, error)
	DescribeTags(
		ctx context.Context,
		params *elbv2.DescribeTagsInput,
		optFns ...func(*elbv2.Options),
	) (*elbv2.DescribeTagsOutput, error)
}

func FindClusterNLB(ctx context.Context, client ELBv2API, clusterName string) (string, error) {
	// First check for a single load balancer with the expected name.
	lbName := fmt.Sprintf("keights-%s", clusterName)
	lbOut, err := client.DescribeLoadBalancers(ctx, &elbv2.DescribeLoadBalancersInput{
		Names: []string{lbName},
	})
	// Ignore any error, we will try again if not found.
	if err == nil {
		if len(lbOut.LoadBalancers) == 1 {
			dnsName := aws.ToString(lbOut.LoadBalancers[0].DNSName)
			return dnsName, nil
		}
	}

	// Now try the slow way, searching by tag until it is found.
	arns, dnsNames, err := listLoadBalancers(ctx, client)
	if err != nil {
		return "", err
	}

	if len(arns) == 0 {
		return "", fmt.Errorf("load balancer not found")
	}

	return findTaggedNLB(ctx, client, clusterName, arns, dnsNames)
}

func listLoadBalancers(ctx context.Context, client ELBv2API) ([]string, map[string]string, error) {
	var arns []string
	dnsNames := make(map[string]string)

	paginator := elbv2.NewDescribeLoadBalancersPaginator(
		client, &elbv2.DescribeLoadBalancersInput{},
	)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to list load balancers: %w", err)
		}
		for _, lb := range page.LoadBalancers {
			arns = append(arns, aws.ToString(lb.LoadBalancerArn))
			dnsNames[aws.ToString(lb.LoadBalancerArn)] = aws.ToString(lb.DNSName)
		}
	}

	return arns, dnsNames, nil
}

func findTaggedNLB(
	ctx context.Context,
	client ELBv2API,
	clusterName string,
	arns []string,
	dnsNames map[string]string,
) (string, error) {
	const maxARNsPerCall = 20
	for i := 0; i < len(arns); i += maxARNsPerCall {
		end := i + maxARNsPerCall
		if end > len(arns) {
			end = len(arns)
		}

		tagsOut, err := client.DescribeTags(ctx, &elbv2.DescribeTagsInput{
			ResourceArns: arns[i:end],
		})
		if err != nil {
			return "", fmt.Errorf("unable to describe tags: %w", err)
		}

		if dns, ok := matchTag(tagsOut.TagDescriptions,
			clusterName, dnsNames); ok {
			return dns, nil
		}
	}

	return "", fmt.Errorf("load balancer not found with tag %s=%s", clusterTag, clusterName)
}

func matchTag(
	descriptions []types.TagDescription,
	clusterName string,
	dnsNames map[string]string,
) (string, bool) {
	for _, td := range descriptions {
		for _, tag := range td.Tags {
			if aws.ToString(tag.Key) == clusterTag &&
				aws.ToString(tag.Value) == clusterName {
				dns := dnsNames[aws.ToString(td.ResourceArn)]
				return dns, true
			}
		}
	}
	return "", false
}
