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
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
)

var clusterNameRE = regexp.MustCompile(`^[a-z][a-z0-9-]{0,61}[a-z0-9]$`)

var instanceTypeRE = regexp.MustCompile(`^[a-z0-9]+\.[a-z0-9-]+$`)

const customInstanceType = "__other__"

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
//
// The wizard runs in three consolidated forms separated by AWS discovery:
// pre-discovery (basics + VPC), main (AMI + KMS + key pair + SSH + control
// plane), and tail (state backend + output + confirms). shift+tab provides
// back-navigation between groups within each form. The node-group loop and
// the boundaries between the three forms are one-way.
func RunWizard(ctx context.Context, disc *Discoverer, defs Defaults) (*Answers, error) {
	a := &Answers{
		Region:         defs.Region,
		ClusterName:    defs.ClusterName,
		AccessCIDRsAPI: []string{"0.0.0.0/0"},
		IRSAEnabled:    true,
		ControlPlane:   ControlPlane{Internal: true},
	}

	vpcs, err := disc.VPCs(ctx)
	if err != nil {
		return nil, err
	}
	if len(vpcs) == 0 {
		return nil, errors.New("no vpcs found in region " + a.Region)
	}

	if err := runPreDiscovery(vpcs, a); err != nil {
		return nil, err
	}

	if err := disc.ValidateAccount(ctx, a.Region, a.VPCID, "", nil); err != nil {
		return nil, err
	}

	subnets, err := disc.Subnets(ctx, a.VPCID)
	if err != nil {
		return nil, err
	}

	amis, err := disc.AMIs(ctx)
	if err != nil {
		return nil, err
	}

	kmsKeys, err := disc.KMSKeys(ctx)
	if err != nil {
		return nil, err
	}

	keyPairs, err := disc.KeyPairs(ctx)
	if err != nil {
		return nil, err
	}

	if err := runMain(amis, kmsKeys, keyPairs, subnets, a); err != nil {
		return nil, err
	}

	if err := runNodeGroups(subnets, a); err != nil {
		return nil, err
	}

	if a.OutputDir == "" {
		a.OutputDir = filepath.Clean("./" + a.ClusterName)
	}

	confirmed, err := runTail(a)
	if err != nil {
		return nil, err
	}
	if !confirmed {
		return nil, nil
	}

	return a, nil
}

// runPreDiscovery collects the fields that must be known before AWS
// discovery can populate the remaining options: cluster identity, API/
// node-port CIDRs, IRSA, and VPC.
func runPreDiscovery(vpcs []VPC, a *Answers) error {
	accessInput := joinCIDRs(a.AccessCIDRsAPI)
	nodePortsInput := joinCIDRs(a.AccessCIDRsNodePorts)
	vpcOpts := make([]huh.Option[string], 0, len(vpcs))
	for _, v := range vpcs {
		vpcOpts = append(vpcOpts, huh.NewOption(v.Label(), v.ID))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Cluster name").
				Description("Lowercase letters, digits, and hyphens.").
				Value(&a.ClusterName).
				Validate(validateClusterName),
			huh.NewInput().
				Title("AWS region").
				Description("The AWS region where the cluster is located.").
				Placeholder("us-east-1").
				Value(&a.Region).
				Validate(nonEmpty("region")),
			huh.NewInput().
				Title("API access CIDRs").
				Description("Comma separated list of CIDRs that can reach the "+
					"Kubernetes API.").
				Value(&accessInput).
				Validate(validateCIDRListRequired),
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
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("VPC").
				Description("The VPC where the cluster will be deployed.").
				Options(vpcOpts...).
				Height(8).
				Value(&a.VPCID),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}
	a.AccessCIDRsAPI = parseCIDRList(accessInput)
	a.AccessCIDRsNodePorts = parseCIDRList(nodePortsInput)
	return nil
}

// runMain collects the middle section of the wizard — AMI, KMS key, SSH
// key pair, SSH access CIDRs (shown only when a key pair is chosen), and
// the control plane configuration (count, instance type, optional custom
// type, subnets) — in a single form with multiple groups so shift+tab
// provides back-navigation across all of them.
func runMain(
	amis []AMI,
	kmsKeys []KMSKey,
	keyPairs []KeyPair,
	subnets []Subnet,
	a *Answers,
) error {
	amiOpts := make([]huh.Option[string], 0, len(amis))
	amisByID := make(map[string]AMI, len(amis))
	for _, img := range amis {
		amiOpts = append(amiOpts, huh.NewOption(img.Label(), img.ID))
		amisByID[img.ID] = img
	}
	amiID := amis[0].ID

	kmsOpts := []huh.Option[string]{huh.NewOption("(create new)", "")}
	for _, k := range kmsKeys {
		value := k.AliasName
		if value == "" {
			value = k.TargetKeyID
		}
		kmsOpts = append(kmsOpts, huh.NewOption(k.Label(), value))
	}

	keyOpts := []huh.Option[string]{huh.NewOption("(none)", "")}
	for _, k := range keyPairs {
		keyOpts = append(keyOpts, huh.NewOption(k.Name, k.Name))
	}

	sshInput := joinCIDRs(a.AccessCIDRsSSH)
	cpCountStr := "1"
	cpTypeSelected := "m5.large"
	cpTypeCustom := ""
	subnetOpts := subnetOptions(subnets)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("AMI").
				Description("Keights AMIs available in this region.").
				Options(amiOpts...).
				Height(8).
				Value(&amiID),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("KMS key").
				Description("KMS key for cluster encryption. "+
					"Choose (create new) to have one created automatically.").
				Options(kmsOpts...).
				Height(8).
				Value(&a.KMSKeyID),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("SSH key pair").
				Description("SSH key pair for instance access. "+
					"Choose (none) to disable sshd on instances.").
				Options(keyOpts...).
				Height(8).
				Value(&a.SSHKeyPair),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("SSH access CIDRs").
				Description("Comma separated. Leave blank to block SSH access.").
				Value(&sshInput).
				Validate(validateCIDRList),
		).WithHideFunc(func() bool { return a.SSHKeyPair == "" }),
		huh.NewGroup(
			huh.NewInput().
				Title("Control plane instance count").
				Description("Use 1 for development or short lived clusters. 3 "+
					"is recommended to tolerate control plane failures.").
				Value(&cpCountStr).
				Validate(validateOddPositive),
			huh.NewSelect[string]().
				Title("Control plane instance type").
				Description("The EC2 instance type for control plane instances.").
				Options(instanceTypeOptions(instanceTypes)...).
				Height(8).
				Value(&cpTypeSelected),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Custom control plane instance type").
				Description("Enter an EC2 instance type, e.g. m6a.4xlarge.").
				Value(&cpTypeCustom).
				Validate(validateInstanceType),
		).WithHideFunc(func() bool { return cpTypeSelected != customInstanceType }),
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Control plane subnets").
				DescriptionFunc(func() string {
					return cpSubnetDescription(cpCountStr)
				}, &cpCountStr).
				Options(subnetOpts...).
				Height(8).
				Value(&a.ControlPlane.SubnetIDs).
				Validate(func(ids []string) error {
					cpCount, _ := strconv.Atoi(cpCountStr)
					if len(ids) != cpCount {
						s := ""
						if cpCount != 1 {
							s = "s"
						}
						return fmt.Errorf("%d subnet%s required",
							cpCount, s)
					}
					return nil
				}),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}

	img := amisByID[amiID]
	a.AMIName = img.Name
	a.AMIOwnerID = img.OwnerID
	a.KubernetesVersion = img.KubernetesVersion
	a.AccessCIDRsSSH = parseCIDRList(sshInput)
	if cpTypeSelected == customInstanceType {
		a.ControlPlane.InstanceType = cpTypeCustom
	} else {
		a.ControlPlane.InstanceType = cpTypeSelected
	}
	return nil
}

// runTail collects state backend, output directory, and final confirms.
// Groups support shift+tab back-navigation; groups conditional on earlier
// choices (S3 details, deploy confirm) are hidden when not applicable.
func runTail(a *Answers) (bool, error) {
	if a.StateBackend.Key == "" {
		a.StateBackend.Key = a.ClusterName + "/terraform.tfstate"
	}
	if a.StateBackend.Region == "" {
		a.StateBackend.Region = a.Region
	}
	confirmed := true
	backendNote := "Terraform state backend (optional, but recommended).\n\n" +
		"Remote state prevents conflicts when multiple people or machines\n" +
		"run Terraform. This wizard can configure an S3 backend directly.\n" +
		"Other backends (azurerm, gcs, remote, etc.) can be configured\n" +
		"by creating a state.tf file in the generated project."

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title("State backend").Description(backendNote),
			huh.NewSelect[string]().
				Title("Backend type").
				Options(
					huh.NewOption("Local (no remote state)", ""),
					huh.NewOption("S3", "s3"),
				).
				Value(&a.StateBackend.Type),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("S3 bucket").
				Description("Name of the existing S3 bucket for state.").
				Value(&a.StateBackend.Bucket).
				Validate(nonEmpty("bucket")),
			huh.NewInput().
				Title("State key").
				Description("Path within the bucket for the state file.").
				Value(&a.StateBackend.Key).
				Validate(nonEmpty("key")),
			huh.NewInput().
				Title("Bucket region").
				Description("AWS region where the bucket is located.").
				Value(&a.StateBackend.Region).
				Validate(nonEmpty("region")),
		).WithHideFunc(func() bool { return a.StateBackend.Type != "s3" }),
		huh.NewGroup(
			huh.NewInput().
				Title("Output directory").
				Description("Directory to write Terraform configuration to.").
				Value(&a.OutputDir).
				Validate(validateOutputDir),
			huh.NewConfirm().
				Title("Generate Terraform configuration?").
				Affirmative("Yes").Negative("Cancel").
				Value(&confirmed),
		),
		huh.NewGroup(
			huh.NewConfirm().
				TitleFunc(func() string {
					return "Run terraform apply now?"
				}, &a.OutputDir).
				DescriptionFunc(func() string {
					return "If no, you can deploy later with: keights deploy " +
						a.OutputDir
				}, &a.OutputDir).
				Affirmative("Yes").Negative("No").
				Value(&a.DeployNow),
		).WithHideFunc(func() bool { return !confirmed }),
	)
	if err := form.Run(); err != nil {
		return false, err
	}
	return confirmed, nil
}

func cpSubnetDescription(cpCountStr string) string {
	cpCount, _ := strconv.Atoi(cpCountStr)
	s := ""
	if cpCount != 1 {
		s = "s"
	}
	desc := fmt.Sprintf(
		"Use space to select %d subnet%s, enter to confirm.", cpCount, s)
	if cpCount > 1 {
		desc += " Choose one subnet per availability zone."
	}
	return desc
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
		typeSelected := ng.InstanceType
		typeCustom := ""
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
					Value(&typeSelected),
			),
			huh.NewGroup(
				huh.NewInput().
					Title("Custom instance type").
					Description("Enter an EC2 instance type, e.g. m6a.4xlarge.").
					Value(&typeCustom).
					Validate(validateInstanceType),
			).WithHideFunc(func() bool { return typeSelected != customInstanceType }),
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
		if typeSelected == customInstanceType {
			ng.InstanceType = typeCustom
		} else {
			ng.InstanceType = typeSelected
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

func subnetOptions(subnets []Subnet) []huh.Option[string] {
	out := make([]huh.Option[string], 0, len(subnets))
	for _, s := range subnets {
		out = append(out, huh.NewOption(s.Label(), s.ID))
	}
	return out
}

func instanceTypeOptions(types []string) []huh.Option[string] {
	out := make([]huh.Option[string], 0, len(types)+1)
	for _, t := range types {
		out = append(out, huh.NewOption(t, t))
	}
	return append(out, huh.NewOption("Other (type manually)", customInstanceType))
}

func validateInstanceType(s string) error {
	if !instanceTypeRE.MatchString(s) {
		return errors.New("must look like an EC2 instance type, e.g. m5.large")
	}
	return nil
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

func validateClusterName(s string) error {
	if err := validateK8sName(s); err != nil {
		return err
	}
	return validateDirDoesNotExist(filepath.Clean("./" + s))
}

func validateOutputDir(s string) error {
	if s == "" {
		return errors.New("output directory is required")
	}
	return validateDirDoesNotExist(s)
}

func validateDirDoesNotExist(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("directory %q already exists; pick a different "+
			"name or remove it", path)
	}
	return fmt.Errorf("%q exists and is not a directory", path)
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

func validateCIDRListRequired(s string) error {
	list := parseCIDRList(s)
	if len(list) == 0 {
		return errors.New("at least one cidr is required")
	}
	for _, c := range list {
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
