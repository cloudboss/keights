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
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockELBv2 struct {
	loadBalancers []types.LoadBalancer
	tagDescs      []types.TagDescription
}

func (m *mockELBv2) DescribeLoadBalancers(
	ctx context.Context,
	params *elbv2.DescribeLoadBalancersInput,
	optFns ...func(*elbv2.Options),
) (*elbv2.DescribeLoadBalancersOutput, error) {
	if params.Names != nil && len(params.Names) > 0 {
		var filtered []types.LoadBalancer
		for _, lb := range m.loadBalancers {
			for _, name := range params.Names {
				if aws.ToString(lb.LoadBalancerName) == name {
					filtered = append(filtered, lb)
				}
			}
		}
		return &elbv2.DescribeLoadBalancersOutput{LoadBalancers: filtered}, nil
	}
	return &elbv2.DescribeLoadBalancersOutput{LoadBalancers: m.loadBalancers}, nil
}

func (m *mockELBv2) DescribeTags(
	ctx context.Context,
	params *elbv2.DescribeTagsInput,
	optFns ...func(*elbv2.Options),
) (*elbv2.DescribeTagsOutput, error) {
	return &elbv2.DescribeTagsOutput{
		TagDescriptions: m.tagDescs,
	}, nil
}

func TestFindClusterNLB(t *testing.T) {
	client := &mockELBv2{
		loadBalancers: []types.LoadBalancer{
			{
				LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/other/abc"),
				DNSName:         aws.String("other.example.com"),
			},
			{
				LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/mycluster/def"),
				DNSName:         aws.String("mycluster.example.com"),
			},
		},
		tagDescs: []types.TagDescription{
			{
				ResourceArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/other/abc"),
				Tags:        []types.Tag{{Key: aws.String("Name"), Value: aws.String("other")}},
			},
			{
				ResourceArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/mycluster/def"),
				Tags:        []types.Tag{{Key: aws.String(clusterTag), Value: aws.String("mycluster")}},
			},
		},
	}

	dns, err := FindClusterNLB(context.Background(), client, "mycluster")
	require.NoError(t, err)
	assert.Equal(t, "mycluster.example.com", dns)
}

func TestFindClusterByName(t *testing.T) {
	client := &mockELBv2{
		loadBalancers: []types.LoadBalancer{
			{
				LoadBalancerArn:  aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/other/abc"),
				LoadBalancerName: aws.String("keights-other"),
				DNSName:          aws.String("other.example.com"),
			},
			{
				LoadBalancerArn:  aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/other/abc"),
				LoadBalancerName: aws.String("keights-mycluster"),
				DNSName:          aws.String("mycluster.example.com"),
			},
		},
		// Cluster tags do not match, but it should still be found by name.
		tagDescs: []types.TagDescription{
			{
				ResourceArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/other/abc"),
				Tags:        []types.Tag{{Key: aws.String(clusterTag), Value: aws.String("abc")}},
			},
			{
				ResourceArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/mycluster/def"),
				Tags:        []types.Tag{{Key: aws.String(clusterTag), Value: aws.String("xyz")}},
			},
		},
	}

	dns, err := FindClusterNLB(context.Background(), client, "mycluster")
	require.NoError(t, err)
	assert.Equal(t, "mycluster.example.com", dns)
}

func TestFindClusterByNameFallback(t *testing.T) {
	client := &mockELBv2{
		loadBalancers: []types.LoadBalancer{
			{
				LoadBalancerArn:  aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/other/abc"),
				LoadBalancerName: aws.String("keights-other"),
				DNSName:          aws.String("other.example.com"),
			},
			{
				// Load balancer has a nonstandard name, will fall back to searching by tag.
				LoadBalancerArn:  aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/mycluster/def"),
				LoadBalancerName: aws.String("keights-unexpected"),
				DNSName:          aws.String("mycluster.example.com"),
			},
		},
		tagDescs: []types.TagDescription{
			{
				ResourceArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/other/abc"),
				Tags:        []types.Tag{{Key: aws.String("Name"), Value: aws.String("other")}},
			},
			{
				ResourceArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/mycluster/def"),
				Tags:        []types.Tag{{Key: aws.String(clusterTag), Value: aws.String("mycluster")}},
			},
		},
	}

	dns, err := FindClusterNLB(context.Background(), client, "mycluster")
	require.NoError(t, err)
	assert.Equal(t, "mycluster.example.com", dns)
}

func TestFindClusterByNameNotFound(t *testing.T) {
	client := &mockELBv2{
		loadBalancers: []types.LoadBalancer{
			{
				LoadBalancerArn:  aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/other/abc"),
				LoadBalancerName: aws.String("keights-other"),
				DNSName:          aws.String("other.example.com"),
			},
		},
		tagDescs: []types.TagDescription{},
	}

	_, err := FindClusterNLB(context.Background(), client, "mycluster")
	assert.ErrorContains(t, err, "load balancer not found")
}

func TestFindClusterNLBNotFound(t *testing.T) {
	client := &mockELBv2{
		loadBalancers: []types.LoadBalancer{
			{
				LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/other/abc"),
				DNSName:         aws.String("other.example.com"),
			},
		},
		tagDescs: []types.TagDescription{
			{
				ResourceArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/net/other/abc"),
				Tags:        []types.Tag{{Key: aws.String("Name"), Value: aws.String("other")}},
			},
		},
	}

	_, err := FindClusterNLB(context.Background(), client, "mycluster")
	assert.ErrorContains(t, err, "load balancer not found")
}

func TestFindClusterNLBNoLoadBalancers(t *testing.T) {
	client := &mockELBv2{}

	_, err := FindClusterNLB(context.Background(), client, "mycluster")
	assert.ErrorContains(t, err, "load balancer not found")
}
