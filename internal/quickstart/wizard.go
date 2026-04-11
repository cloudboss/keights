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
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
)

var clusterNameRE = regexp.MustCompile(`^[a-z][a-z0-9-]{0,61}[a-z0-9]$`)

var instanceTypes = []string{
	"t3.medium", "t3.large", "t3.xlarge",
	"t3a.medium", "t3a.large", "t3a.xlarge",
	"m5.large", "m5.xlarge", "m5.2xlarge", "m5.4xlarge",
	"m5a.large", "m5a.xlarge", "m5a.2xlarge",
	"m6i.large", "m6i.xlarge", "m6i.2xlarge", "m6i.4xlarge",
	"m6a.large", "m6a.xlarge", "m6a.2xlarge",
	"m7i.large", "m7i.xlarge", "m7i.2xlarge",
	"m7a.large", "m7a.xlarge", "m7a.2xlarge",
	"c5.large", "c5.xlarge", "c5.2xlarge",
	"c6i.large", "c6i.xlarge", "c6i.2xlarge",
	"c7i.large", "c7i.xlarge", "c7i.2xlarge",
	"r5.large", "r5.xlarge",
	"r6i.large", "r6i.xlarge",
	"r7i.large", "r7i.xlarge",
}

// RunWizard collects cluster configuration interactively.
// Returns (nil, nil) if the user aborts before the final confirmation.
func RunWizard(ctx context.Context, disc *Discoverer, defs Defaults) (*Answers, error) {
	a := &Answers{
		Region:      defs.Region,
		ClusterName: defs.ClusterName,
		AccessCIDR:  "0.0.0.0/0",
		IRSAEnabled: true,
	}

	if err := runBasics(a); err != nil {
		return nil, err
	}

	vpcs, err := disc.VPCs(ctx)
	if err != nil {
		return nil, err
	}
	if len(vpcs) == 0 {
		return nil, errors.New("no vpcs found in region " + a.Region)
	}

	subnets, err := runVPCAndSubnets(ctx, disc, vpcs, a)
	if err != nil {
		return nil, err
	}

	amis, err := disc.AMIs(ctx)
	if err != nil {
		return nil, err
	}
	if len(amis) == 0 {
		return nil, errors.New("no keights ami found in this region")
	}

	kmsKeys, err := disc.KMSKeys(ctx)
	if err != nil {
		return nil, err
	}
	if len(kmsKeys) == 0 {
		return nil, errors.New("no customer-managed kms keys found in this region")
	}

	keyPairs, err := disc.KeyPairs(ctx)
	if err != nil {
		return nil, err
	}

	if err := runAMI(amis, a); err != nil {
		return nil, err
	}

	if err := runKMSKey(kmsKeys, a); err != nil {
		return nil, err
	}

	if err := runControlPlane(subnets, a); err != nil {
		return nil, err
	}

	if err := runCredentials(keyPairs, a); err != nil {
		return nil, err
	}

	if a.SSHKeyPair != "" {
		if err := runSSHAccess(a); err != nil {
			return nil, err
		}
	}

	if err := runNodeGroups(subnets, a); err != nil {
		return nil, err
	}

	if a.OutputDir == "" {
		a.OutputDir = filepath.Clean("./" + a.ClusterName)
	}

	confirmed, err := runConfirm(a)
	if err != nil {
		return nil, err
	}
	if !confirmed {
		return nil, nil
	}

	return a, nil
}

func runBasics(a *Answers) error {
	nodePortsInput := joinCIDRs(a.NodePortsCIDRs)
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Cluster name").
				Description("Lowercase letters, digits, and hyphens.").
				Value(&a.ClusterName).
				Validate(validateK8sName),
			huh.NewInput().
				Title("AWS region").
				Description("The AWS region where the cluster is located.").
				Placeholder("us-east-1").
				Value(&a.Region).
				Validate(nonEmpty("region")),
			huh.NewInput().
				Title("API access CIDR").
				Description("Who can reach the Kubernetes API.").
				Value(&a.AccessCIDR).
				Validate(validateCIDR),
			huh.NewInput().
				Title("Node port access CIDRs").
				Description("Comma separated list. Leave blank to disable.").
				Value(&nodePortsInput).
				Validate(validateCIDRList),
			huh.NewConfirm().
				Title("Enable IAM Roles for Service Accounts (IRSA)?").
				Description("Creates an OIDC provider for pods to assume IAM roles.").
				Affirmative("Yes").Negative("No").
				Value(&a.IRSAEnabled),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}
	a.NodePortsCIDRs = parseCIDRList(nodePortsInput)
	return nil
}

func runVPCAndSubnets(
	ctx context.Context,
	disc *Discoverer,
	vpcs []VPC,
	a *Answers,
) ([]Subnet, error) {
	vpcOpts := make([]huh.Option[string], 0, len(vpcs))
	for _, v := range vpcs {
		vpcOpts = append(vpcOpts, huh.NewOption(v.Label(), v.ID))
	}

	vpcForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("VPC").
				Description("The VPC where the cluster will be deployed.").
				Options(vpcOpts...).
				Height(8).
				Value(&a.VPCID),
		),
	)
	if err := vpcForm.Run(); err != nil {
		return nil, err
	}

	subnets, err := disc.Subnets(ctx, a.VPCID)
	if err != nil {
		return nil, err
	}
	if len(subnets) == 0 {
		return nil, fmt.Errorf("no subnets found in %s", a.VPCID)
	}
	return subnets, nil
}

func runControlPlane(subnets []Subnet, a *Answers) error {
	instanceType := "m5.large"
	cpCountStr := "1"
	subnetOpts := subnetOptions(subnets)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Control plane instance count").
				Description("Enter the number of control plane instances. "+
					"Use 1 for development or short lived clusters. 3 is "+
					"recommended to tolerate control plane failures.").
				Value(&cpCountStr).
				Validate(validateOddPositive),
			huh.NewSelect[string]().
				Title("Instance type").
				Description("The EC2 instance type for control plane instances.").
				Options(instanceTypeOptions(instanceTypes)...).
				Height(8).
				Value(&instanceType),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}
	cpCount, _ := strconv.Atoi(cpCountStr)

	s := ""
	if cpCount > 1 {
		s = "s"
	}
	subnetDescription := "Select " + cpCountStr + " subnet" + s
	if cpCount > 1 {
		subnetDescription += ", one per availability zone."
	}

	subnetForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Control plane subnets").
				Description(subnetDescription).
				Options(subnetOpts...).
				Height(8).
				Value(&a.ControlPlane.SubnetIDs).
				Validate(func(ids []string) error {
					if len(ids) != cpCount {
						return fmt.Errorf("select exactly %d subnet%s",
							cpCount, s)
					}
					return nil
				}),
		),
	)
	if err := subnetForm.Run(); err != nil {
		return err
	}
	a.ControlPlane.InstanceType = instanceType
	return nil
}

func runAMI(amis []AMI, a *Answers) error {
	opts := make([]huh.Option[string], 0, len(amis))
	byID := make(map[string]AMI, len(amis))
	for _, img := range amis {
		opts = append(opts, huh.NewOption(img.Label(), img.ID))
		byID[img.ID] = img
	}
	selected := amis[0].ID
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("AMI").
				Description("Keights AMIs available in this region.").
				Options(opts...).
				Height(8).
				Value(&selected),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}
	img := byID[selected]
	a.AMIName = img.Name
	a.AMIOwnerID = img.OwnerID
	return nil
}

func runKMSKey(keys []KMSKey, a *Answers) error {
	opts := make([]huh.Option[string], 0, len(keys))
	for _, k := range keys {
		value := k.AliasName
		if value == "" {
			value = k.TargetKeyID
		}
		opts = append(opts, huh.NewOption(k.Label(), value))
	}
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("KMS key").
				Description("Customer managed KMS key for cluster encryption.").
				Options(opts...).
				Height(8).
				Value(&a.KMSKeyID),
		),
	)
	return form.Run()
}

func runSSHAccess(a *Answers) error {
	input := joinCIDRs(a.SSHCIDRs)
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("SSH access CIDRs").
				Description("Comma separated. Leave blank to block SSH access.").
				Value(&input).
				Validate(validateCIDRList),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}
	a.SSHCIDRs = parseCIDRList(input)
	return nil
}

func runCredentials(keyPairs []KeyPair, a *Answers) error {
	const noneKey = ""
	keyOpts := []huh.Option[string]{huh.NewOption("(none)", noneKey)}
	for _, k := range keyPairs {
		keyOpts = append(keyOpts, huh.NewOption(k.Name, k.Name))
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("SSH key pair").
				Description("SSH key pair for instance access. " +
					"Choose (none) for no SSH access.").
				Options(keyOpts...).
				Height(8).
				Value(&a.SSHKeyPair),
		),
	).Run()
}

func runNodeGroups(subnets []Subnet, a *Answers) error {
	subnetOpts := subnetOptions(subnets)

	// Default: one node group named "default" on all subnets.
	defaultSubnets := make([]string, 0, len(subnets))
	for _, s := range subnets {
		defaultSubnets = append(defaultSubnets, s.ID)
	}

	for i := 0; ; i++ {
		ng := NodeGroup{
			Name:         defaultGroupName(i),
			InstanceType: "m5.large",
			MinSize:      1,
			DesiredSize:  2,
			MaxSize:      5,
			SubnetIDs:    defaultSubnets,
		}
		min := strconv.Itoa(ng.MinSize)
		desired := strconv.Itoa(ng.DesiredSize)
		max := strconv.Itoa(ng.MaxSize)

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title(fmt.Sprintf("Configuration for node group #%d", i+1)),
				huh.NewInput().
					Title("Name").
					Description("Name of the node group.").
					Value(&ng.Name).
					Validate(validateK8sName),
				huh.NewSelect[string]().
					Title("Instance type").
					Description("The EC2 instance type for the group.").
					Options(instanceTypeOptions(instanceTypes)...).
					Height(8).
					Value(&ng.InstanceType),
			),
			huh.NewGroup(
				huh.NewInput().Title("Min count").Value(&min).
					Description("Minimum number of nodes in the group.").
					Validate(validateUint),
				huh.NewInput().Title("Desired count").Value(&desired).
					Validate(validateUint).
					Description("Desired number of nodes in the group."),
				huh.NewInput().Title("Max count").Value(&max).
					Description("Maximum number of nodes in the group.").
					Validate(validateUint),
				huh.NewMultiSelect[string]().
					Title("Subnets").
					Options(subnetOpts...).
					Height(8).
					Value(&ng.SubnetIDs).
					Validate(func(ids []string) error {
						if len(ids) == 0 {
							return errors.New("select at least one subnet")
						}
						return nil
					}),
			),
		)
		if err := form.Run(); err != nil {
			return err
		}
		ng.MinSize, _ = strconv.Atoi(min)
		ng.DesiredSize, _ = strconv.Atoi(desired)
		ng.MaxSize, _ = strconv.Atoi(max)

		if err := validateNodeGroupSizes(ng); err != nil {
			fmt.Println("Invalid sizes:", err)
			i-- // Retry this group.
			continue
		}

		a.NodeGroups = append(a.NodeGroups, ng)

		another := false
		if err := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().
				Title("Add another node group?").
				Affirmative("Yes").Negative("No").
				Value(&another),
		)).Run(); err != nil {
			return err
		}
		if !another {
			return nil
		}
	}
}

func runConfirm(a *Answers) (bool, error) {
	confirmed := true
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Output directory").
				Description("Directory to write Terraform configuration to.").
				Value(&a.OutputDir).
				Validate(nonEmpty("output directory")),
			huh.NewConfirm().
				Title("Generate Terraform configuration?").
				Affirmative("Yes").Negative("Cancel").
				Value(&confirmed),
		),
	)
	if err := form.Run(); err != nil {
		return false, err
	}
	if !confirmed {
		return false, nil
	}
	return huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title("Run terraform apply now?").
			Description("If no, you can deploy later with: keights deploy "+a.OutputDir).
			Affirmative("Yes").Negative("No").
			Value(&a.DeployNow),
	)).Run() == nil, nil
}

func subnetOptions(subnets []Subnet) []huh.Option[string] {
	out := make([]huh.Option[string], 0, len(subnets))
	for _, s := range subnets {
		out = append(out, huh.NewOption(s.Label(), s.ID))
	}
	return out
}

func instanceTypeOptions(types []string) []huh.Option[string] {
	out := make([]huh.Option[string], 0, len(types))
	for _, t := range types {
		out = append(out, huh.NewOption(t, t))
	}
	return out
}

func defaultGroupName(i int) string {
	if i == 0 {
		return "default"
	}
	return fmt.Sprintf("group%d", i+1)
}

func validateK8sName(s string) error {
	if !clusterNameRE.MatchString(s) {
		return errors.New("must be lowercase letters, digits, and hyphens, 2-63 characters")
	}
	return nil
}

func validateCIDR(s string) error {
	if _, _, err := net.ParseCIDR(s); err != nil {
		return errors.New("cidr must be a network address and netmask, " +
			"such as 192.168.0.0/16")
	}
	return nil
}

func validateCIDRList(s string) error {
	for _, c := range parseCIDRList(s) {
		if err := validateCIDR(c); err != nil {
			return err
		}
	}
	return nil
}

func parseCIDRList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func joinCIDRs(cidrs []string) string {
	return strings.Join(cidrs, ", ")
}

func validateOddPositive(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return errors.New("must be a positive number")
	}
	if n%2 == 0 {
		return errors.New("must be an odd number")
	}
	return nil
}

func validateUint(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return errors.New("must be a non-negative integer")
	}
	return nil
}

func validateNodeGroupSizes(ng NodeGroup) error {
	if ng.MinSize > ng.MaxSize {
		return errors.New("min > max")
	}
	if ng.DesiredSize < ng.MinSize || ng.DesiredSize > ng.MaxSize {
		return errors.New("desired must be between min and max")
	}
	return nil
}

func nonEmpty(label string) func(string) error {
	return func(s string) error {
		if s == "" {
			return fmt.Errorf("%s is required", label)
		}
		return nil
	}
}
