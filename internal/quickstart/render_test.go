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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	dir := t.TempDir()
	a := Answers{
		ClusterName: "demo",
		Region:      "us-east-1",
		VPCID:       "vpc-abc",
		AccessCIDRsAPI: []string{"10.0.0.0/16", "192.168.0.0/16"},
		AMIName:     "keights-1.34.1",
		AMIOwnerID:  "123456789012",
		IRSAEnabled: true,
		KMSKeyID:    "alias/keights-demo",
		SSHKeyPair:  "mykey",
		ControlPlane: ControlPlane{
			InstanceType: "m5.large",
			SubnetIDs:    []string{"subnet-a", "subnet-b", "subnet-c"},
		},
		NodeGroups: []NodeGroup{
			{
				Name: "default", InstanceType: "m5.large",
				MinSize: 1, DesiredSize: 2, MaxSize: 5,
				SubnetIDs: []string{"subnet-a", "subnet-b", "subnet-c"},
			},
			{
				Name: "big", InstanceType: "m5.xlarge",
				MinSize: 0, DesiredSize: 3, MaxSize: 10,
				SubnetIDs: []string{"subnet-a", "subnet-b"},
			},
		},
		OutputDir: dir,
	}

	require.NoError(t, Render(a, RenderOptions{ModuleSource: "v2.0.0-tarball"}))

	main, err := os.ReadFile(filepath.Join(dir, "main.tf"))
	require.NoError(t, err)
	mainStr := string(main)
	assert.Contains(t, mainStr, `module "keights"`)
	assert.Contains(t, mainStr, "v2.0.0-tarball")

	vars, err := os.ReadFile(filepath.Join(dir, "vars.tf"))
	require.NoError(t, err)
	v := string(vars)
	assert.Contains(t, v, `cluster_name = "demo"`)
	assert.Contains(t, v, `aws_region   = "us-east-1"`)
	assert.Contains(t, v, `vpc_id = "vpc-abc"`)
	assert.Contains(t, v, `name  = "keights-1.34.1"`)
	assert.Contains(t, v, `owner = "123456789012"`)
	assert.Contains(t, v, `irsa_enabled = true`)
	assert.Contains(t, v, `kms_key_id = "alias/keights-demo"`)
	assert.Contains(t, mainStr, `oidc_provider_arn`)
	assert.Contains(t, v, `api        = ["10.0.0.0/16", "192.168.0.0/16"]`)
	assert.Contains(t, v, `node_ports = []`)
	assert.Contains(t, v, `ssh        = []`)
	assert.Contains(t, v, `instance_type = "m5.large"`)
	assert.Contains(t, v, `key_pair      = "mykey"`)
	assert.Contains(t, v, `autoscaling_group = ["subnet-a", "subnet-b", "subnet-c"]`)
	assert.Contains(t, v, `default = {`)
	assert.Contains(t, v, `big = {`)
	assert.Contains(t, v, `instance_type = "m5.xlarge"`)
	assert.Contains(t, v, `desired_capacity = 3`)
}

func TestRenderNoSSHKey(t *testing.T) {
	dir := t.TempDir()
	a := Answers{
		ClusterName: "demo", Region: "us-east-1", VPCID: "vpc-x",
		AccessCIDRsAPI: []string{"0.0.0.0/0"}, KMSKeyID: "alias/k",
		ControlPlane: ControlPlane{
			InstanceType: "m5.large",
			SubnetIDs:    []string{"subnet-a"},
		},
		NodeGroups: []NodeGroup{
			{
				Name: "default", InstanceType: "m5.large",
				MinSize: 1, DesiredSize: 1, MaxSize: 1,
				SubnetIDs: []string{"subnet-a"},
			},
		},
		OutputDir: dir,
	}
	require.NoError(t, Render(a, RenderOptions{}))
	vars, err := os.ReadFile(filepath.Join(dir, "vars.tf"))
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(vars), "key_pair"),
		"expected no key_pair when SSHKeyPair is empty")
}

func TestRenderNoKMSKey(t *testing.T) {
	dir := t.TempDir()
	a := Answers{
		ClusterName: "demo", Region: "us-east-1", VPCID: "vpc-x",
		AccessCIDRsAPI: []string{"0.0.0.0/0"},
		ControlPlane: ControlPlane{
			InstanceType: "m5.large",
			SubnetIDs:    []string{"subnet-a"},
		},
		NodeGroups: []NodeGroup{
			{
				Name: "default", InstanceType: "m5.large",
				MinSize: 1, DesiredSize: 1, MaxSize: 1,
				SubnetIDs: []string{"subnet-a"},
			},
		},
		OutputDir: dir,
	}
	require.NoError(t, Render(a, RenderOptions{}))
	vars, err := os.ReadFile(filepath.Join(dir, "vars.tf"))
	require.NoError(t, err)
	v := string(vars)
	assert.Contains(t, v, `kms_key_id = null`)
	assert.False(t, strings.Contains(v, `kms_key_id = ""`),
		"expected null (unquoted) when KMSKeyID is empty")
}

func TestRenderLocalStateHasNoStateFile(t *testing.T) {
	dir := t.TempDir()
	a := Answers{
		ClusterName: "demo", Region: "us-east-1", VPCID: "vpc-x",
		AccessCIDRsAPI: []string{"0.0.0.0/0"},
		ControlPlane: ControlPlane{
			InstanceType: "m5.large",
			SubnetIDs:    []string{"subnet-a"},
		},
		NodeGroups: []NodeGroup{{
			Name: "default", InstanceType: "m5.large",
			MinSize: 1, DesiredSize: 1, MaxSize: 1,
			SubnetIDs: []string{"subnet-a"},
		}},
		OutputDir: dir,
	}
	require.NoError(t, Render(a, RenderOptions{}))
	_, err := os.Stat(filepath.Join(dir, "state.tf"))
	assert.True(t, os.IsNotExist(err),
		"expected no state.tf for local backend, got err=%v", err)
}

func TestRenderS3Backend(t *testing.T) {
	dir := t.TempDir()
	a := Answers{
		ClusterName: "demo", Region: "us-east-1", VPCID: "vpc-x",
		AccessCIDRsAPI: []string{"0.0.0.0/0"},
		ControlPlane: ControlPlane{
			InstanceType: "m5.large",
			SubnetIDs:    []string{"subnet-a"},
		},
		NodeGroups: []NodeGroup{{
			Name: "default", InstanceType: "m5.large",
			MinSize: 1, DesiredSize: 1, MaxSize: 1,
			SubnetIDs: []string{"subnet-a"},
		}},
		StateBackend: StateBackend{
			Type:   "s3",
			Bucket: "my-tfstate",
			Key:    "demo/terraform.tfstate",
			Region: "us-east-1",
		},
		OutputDir: dir,
	}
	require.NoError(t, Render(a, RenderOptions{}))
	state, err := os.ReadFile(filepath.Join(dir, "state.tf"))
	require.NoError(t, err)
	s := string(state)
	assert.Contains(t, s, `backend "s3"`)
	assert.Contains(t, s, `bucket       = "my-tfstate"`)
	assert.Contains(t, s, `key          = "demo/terraform.tfstate"`)
	assert.Contains(t, s, `region       = "us-east-1"`)
	assert.Contains(t, s, `use_lockfile = true`)
}

func TestQuotedList(t *testing.T) {
	assert.Equal(t, "[]", quotedList(nil))
	assert.Equal(t, `["a"]`, quotedList([]string{"a"}))
	assert.Equal(t, `["a", "b"]`, quotedList([]string{"a", "b"}))
}
