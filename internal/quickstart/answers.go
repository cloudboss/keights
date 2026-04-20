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
	"fmt"
	"io"
	"strings"
)

// Answers is the full set of user choices collected by the wizard.
type Answers struct {
	ClusterName    string
	Region         string
	VPCID          string
	AccessCIDRs    []string
	NodePortsCIDRs []string
	SSHCIDRs       []string
	AMIName        string
	AMIOwnerID     string
	KMSKeyID       string
	SSHKeyPair     string
	IRSAEnabled    bool
	ControlPlane   ControlPlane
	NodeGroups     []NodeGroup
	StateBackend   StateBackend
	OutputDir      string
	DeployNow      bool
}

// StateBackend describes the Terraform state backend to configure in the
// generated project. Type is "" for local state or "s3" for S3-backed state.
type StateBackend struct {
	Type   string
	Bucket string
	Key    string
	Region string
}

type ControlPlane struct {
	InstanceType string
	SubnetIDs    []string
}

type NodeGroup struct {
	Name         string
	InstanceType string
	MinSize      int
	MaxSize      int
	DesiredSize  int
	SubnetIDs    []string
}

// Defaults carries pre-populated values the wizard should seed into prompts.
type Defaults struct {
	Region      string
	ClusterName string
}

// PrintSummary writes a human-readable recap of the answers.
func PrintSummary(w io.Writer, a Answers) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Cluster configuration")
	fmt.Fprintln(w, "---------------------")
	fmt.Fprintf(w, "  Cluster name:   %s\n", a.ClusterName)
	fmt.Fprintf(w, "  Region:         %s\n", a.Region)
	fmt.Fprintf(w, "  VPC:            %s\n", a.VPCID)
	fmt.Fprintf(w, "  API access:     %s\n", orNoneList(a.AccessCIDRs))
	fmt.Fprintf(w, "  Node ports:     %s\n", orNoneList(a.NodePortsCIDRs))
	fmt.Fprintf(w, "  SSH access:     %s\n", orNoneList(a.SSHCIDRs))
	fmt.Fprintf(w, "  AMI:            %s (owner %s)\n", a.AMIName, a.AMIOwnerID)
	fmt.Fprintf(w, "  KMS key:        %s\n", orCreateNew(a.KMSKeyID))
	fmt.Fprintf(w, "  IRSA:           %s\n", enabledOrDisabled(a.IRSAEnabled))
	fmt.Fprintf(w, "  SSH key pair:   %s\n", orNone(a.SSHKeyPair))
	fmt.Fprintf(w, "  State backend:  %s\n", stateBackendSummary(a.StateBackend))
	fmt.Fprintf(w, "  Output dir:     %s\n", a.OutputDir)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Control plane")
	fmt.Fprintf(w, "  Instance type:  %s\n", a.ControlPlane.InstanceType)
	fmt.Fprintf(w, "  Nodes:          %d\n", len(a.ControlPlane.SubnetIDs))
	fmt.Fprintf(w, "  Subnets:        %s\n",
		strings.Join(a.ControlPlane.SubnetIDs, ", "))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Node groups (%d)\n", len(a.NodeGroups))
	for _, ng := range a.NodeGroups {
		fmt.Fprintf(w, "  - %s\n", ng.Name)
		fmt.Fprintf(w, "      instance_type:  %s\n", ng.InstanceType)
		fmt.Fprintf(w, "      size:           min=%d desired=%d max=%d\n",
			ng.MinSize, ng.DesiredSize, ng.MaxSize)
		fmt.Fprintf(w, "      subnets:        %s\n",
			strings.Join(ng.SubnetIDs, ", "))
	}
	fmt.Fprintln(w)
}

func enabledOrDisabled(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func stateBackendSummary(b StateBackend) string {
	if b.Type == "s3" {
		return fmt.Sprintf("s3 (bucket=%s key=%s region=%s)",
			b.Bucket, b.Key, b.Region)
	}
	return "local"
}

func orCreateNew(s string) string {
	if s == "" {
		return "(create new)"
	}
	return s
}

func orNoneList(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}
