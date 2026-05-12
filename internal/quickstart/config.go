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
	"net"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds all inputs for non-interactive cluster configuration.
type Config struct {
	NonInteractive       bool
	ClusterName          string
	Region               string
	VPCID                string
	AccessCIDRsAPI       []string
	AccessCIDRsNodePorts []string
	AccessCIDRsSSH       []string
	AMIName              string
	AMIOwnerID           string
	KMSKeyID             string
	SSHKeyPair           string
	IRSAEnabled          bool
	KubernetesVersion    string
	LambdaBucket         string
	LambdaVersion        string
	ControlPlane         string
	NodeGroups           []string
	StateBackend         string
	StateBucket          string
	StateKey             string
	StateRegion          string
	OutputDir            string
	Deploy               bool
}

// BuildAnswers validates a Config and resolves it into Answers, using the
// Discoverer to fill in any values that were not provided (e.g. AMI).
func BuildAnswers(
	ctx context.Context, disc *Discoverer, cfg Config,
) (*Answers, error) {
	if cfg.Region == "" {
		return nil, fmt.Errorf("region is required")
	}
	if cfg.ClusterName == "" {
		return nil, fmt.Errorf("cluster-name is required")
	}
	if err := validateK8sName(cfg.ClusterName); err != nil {
		return nil, fmt.Errorf("cluster-name: %w", err)
	}
	if cfg.VPCID == "" {
		return nil, fmt.Errorf("vpc-id is required")
	}
	if cfg.ControlPlane == "" {
		return nil, fmt.Errorf("control-plane is required")
	}
	if len(cfg.NodeGroups) == 0 {
		return nil, fmt.Errorf("at least one node-group is required")
	}
	if err := validateCIDRSliceRequired(cfg.AccessCIDRsAPI); err != nil {
		return nil, fmt.Errorf("access-cidrs-api: %w", err)
	}
	if err := validateCIDRSlice(cfg.AccessCIDRsNodePorts); err != nil {
		return nil, fmt.Errorf("access-cidrs-node-ports: %w", err)
	}
	if err := validateCIDRSlice(cfg.AccessCIDRsSSH); err != nil {
		return nil, fmt.Errorf("access-cidrs-ssh: %w", err)
	}
	if cfg.StateBackend == "s3" {
		if cfg.StateBucket == "" {
			return nil, fmt.Errorf("state-bucket is required when state-backend is s3")
		}
	}

	cp, err := parseControlPlane(cfg.ControlPlane)
	if err != nil {
		return nil, err
	}

	nodeGroups, err := parseNodeGroups(cfg.NodeGroups)
	if err != nil {
		return nil, err
	}

	subnetIDs := append([]string(nil), cp.SubnetIDs...)
	for _, ng := range nodeGroups {
		subnetIDs = append(subnetIDs, ng.SubnetIDs...)
	}
	if err := disc.ValidateAccount(
		ctx, cfg.Region, cfg.VPCID, cfg.AMIName, subnetIDs,
	); err != nil {
		return nil, err
	}

	amiName, amiOwnerID, amiK8sVersion, err := resolveAMI(
		ctx, disc, cfg.AMIName, cfg.AMIOwnerID,
	)
	if err != nil {
		return nil, err
	}

	kubernetesVersion := cfg.KubernetesVersion
	if kubernetesVersion == "" {
		kubernetesVersion = amiK8sVersion
	}
	if kubernetesVersion == "" {
		return nil, fmt.Errorf(
			"unable to determine kubernetes version: pass --kubernetes-version",
		)
	}

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = filepath.Clean("./" + cfg.ClusterName)
	}

	if err := validateDirDoesNotExist(outputDir); err != nil {
		return nil, err
	}

	stateKey := cfg.StateKey
	if stateKey == "" && cfg.StateBackend == "s3" {
		stateKey = cfg.ClusterName + "/terraform.tfstate"
	}
	stateRegion := cfg.StateRegion
	if stateRegion == "" {
		stateRegion = cfg.Region
	}

	return &Answers{
		ClusterName:          cfg.ClusterName,
		Region:               cfg.Region,
		VPCID:                cfg.VPCID,
		AccessCIDRsAPI:       cfg.AccessCIDRsAPI,
		AccessCIDRsNodePorts: cfg.AccessCIDRsNodePorts,
		AccessCIDRsSSH:       cfg.AccessCIDRsSSH,
		AMIName:              amiName,
		AMIOwnerID:           amiOwnerID,
		KMSKeyID:             cfg.KMSKeyID,
		SSHKeyPair:           cfg.SSHKeyPair,
		IRSAEnabled:          cfg.IRSAEnabled,
		KubernetesVersion:    kubernetesVersion,
		LambdaBucket:         cfg.LambdaBucket,
		LambdaVersion:        cfg.LambdaVersion,
		ControlPlane:         cp,
		NodeGroups:           nodeGroups,
		StateBackend: StateBackend{
			Type:   cfg.StateBackend,
			Bucket: cfg.StateBucket,
			Key:    stateKey,
			Region: stateRegion,
		},
		OutputDir: outputDir,
		DeployNow: cfg.Deploy,
	}, nil
}

func validateCIDRSlice(cidrs []string) error {
	for _, c := range cidrs {
		if _, _, err := net.ParseCIDR(c); err != nil {
			return fmt.Errorf("invalid CIDR %q", c)
		}
	}
	return nil
}

func validateCIDRSliceRequired(cidrs []string) error {
	if len(cidrs) == 0 {
		return fmt.Errorf("at least one CIDR is required")
	}
	return validateCIDRSlice(cidrs)
}

// resolveAMI returns the AMI name, owner ID, and Kubernetes version. If both
// name and ownerID are provided, discovery is skipped and the Kubernetes
// version comes back empty -- the caller must supply it another way. If only
// name is provided, the AMI is looked up directly by name across the caller's
// account and the official keights account; the name is not required to
// follow the keights naming convention. Otherwise discovery runs over keights
// AMIs and the result that matches the supplied ownerID (or the newest, when
// neither is provided) wins.
func resolveAMI(
	ctx context.Context,
	disc *Discoverer,
	name, ownerID string,
) (string, string, string, error) {
	if name != "" && ownerID != "" {
		return name, ownerID, "", nil
	}
	if name != "" {
		a, ok, err := disc.AMIByName(ctx, name)
		if err != nil {
			return "", "", "", fmt.Errorf("unable to look up AMI: %w", err)
		}
		if !ok {
			return "", "", "", fmt.Errorf(
				"no AMI found matching ami-name=%q in caller account or official keights account",
				name,
			)
		}
		return a.Name, a.OwnerID, a.KubernetesVersion, nil
	}
	amis, err := disc.AMIs(ctx)
	if err != nil {
		return "", "", "", fmt.Errorf("unable to discover AMIs: %w", err)
	}
	match := amis[0]
	if ownerID != "" {
		found := false
		for _, a := range amis {
			if a.OwnerID != ownerID {
				continue
			}
			match = a
			found = true
			break
		}
		if !found {
			return "", "", "", fmt.Errorf(
				"no keights AMI found matching ami-owner-id=%q",
				ownerID,
			)
		}
	}
	return match.Name, match.OwnerID, match.KubernetesVersion, nil
}

func parseControlPlane(spec string) (ControlPlane, error) {
	cp := ControlPlane{
		InstanceType: "m5.large",
		Internal:     true,
	}
	fields := strings.Split(spec, ",")
	for _, field := range fields {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			return cp, fmt.Errorf(
				"control-plane: invalid field %q, expected key=value",
				field,
			)
		}
		switch key {
		case "type":
			cp.InstanceType = value
		case "subnets":
			cp.SubnetIDs = strings.Split(value, ":")
		case "internal":
			internal, err := strconv.ParseBool(value)
			if err != nil {
				return cp, fmt.Errorf(
					"control-plane: invalid internal %q: %w", value, err,
				)
			}
			cp.Internal = internal
		default:
			return cp, fmt.Errorf("control-plane: unknown key %q", key)
		}
	}
	if len(cp.SubnetIDs) == 0 {
		return cp, fmt.Errorf("control-plane: subnets is required")
	}
	if len(cp.SubnetIDs)%2 == 0 {
		return cp, fmt.Errorf(
			"control-plane: subnet count must be odd, got %d",
			len(cp.SubnetIDs),
		)
	}
	if !instanceTypeRE.MatchString(cp.InstanceType) {
		return cp, fmt.Errorf(
			"control-plane: invalid instance type %q", cp.InstanceType,
		)
	}
	return cp, nil
}

func parseNodeGroups(specs []string) ([]NodeGroup, error) {
	groups := make([]NodeGroup, 0, len(specs))
	for i, spec := range specs {
		ng, err := parseNodeGroup(spec)
		if err != nil {
			return nil, fmt.Errorf("node-group[%d]: %w", i, err)
		}
		groups = append(groups, ng)
	}
	return groups, nil
}

func parseNodeGroup(spec string) (NodeGroup, error) {
	ng := NodeGroup{
		MinSize:     1,
		DesiredSize: 2,
		MaxSize:     5,
	}
	fields := strings.Split(spec, ",")
	for _, field := range fields {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			return ng, fmt.Errorf(
				"invalid field %q, expected key=value", field,
			)
		}
		switch key {
		case "name":
			ng.Name = value
		case "type":
			ng.InstanceType = value
		case "min":
			n, err := strconv.Atoi(value)
			if err != nil {
				return ng, fmt.Errorf("invalid min %q: %w", value, err)
			}
			ng.MinSize = n
		case "desired":
			n, err := strconv.Atoi(value)
			if err != nil {
				return ng, fmt.Errorf(
					"invalid desired %q: %w", value, err,
				)
			}
			ng.DesiredSize = n
		case "max":
			n, err := strconv.Atoi(value)
			if err != nil {
				return ng, fmt.Errorf("invalid max %q: %w", value, err)
			}
			ng.MaxSize = n
		case "subnets":
			ng.SubnetIDs = strings.Split(value, ":")
		default:
			return ng, fmt.Errorf("unknown key %q", key)
		}
	}
	if ng.Name == "" {
		return ng, fmt.Errorf("name is required")
	}
	if err := validateK8sName(ng.Name); err != nil {
		return ng, fmt.Errorf("name: %w", err)
	}
	if ng.InstanceType == "" {
		return ng, fmt.Errorf("type is required")
	}
	if !instanceTypeRE.MatchString(ng.InstanceType) {
		return ng, fmt.Errorf("invalid instance type %q", ng.InstanceType)
	}
	if len(ng.SubnetIDs) == 0 {
		return ng, fmt.Errorf("subnets is required")
	}
	if ng.MinSize > ng.MaxSize {
		return ng, fmt.Errorf("min (%d) > max (%d)", ng.MinSize, ng.MaxSize)
	}
	if ng.DesiredSize < ng.MinSize || ng.DesiredSize > ng.MaxSize {
		return ng, fmt.Errorf(
			"desired (%d) must be between min (%d) and max (%d)",
			ng.DesiredSize, ng.MinSize, ng.MaxSize,
		)
	}
	return ng, nil
}
