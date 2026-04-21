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

package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudboss/keights/internal/deps"
)

type Options struct {
	Dir         string
	AutoApprove bool
}

func Run(opts Options) error {
	if err := validateDir(opts.Dir); err != nil {
		return err
	}

	tfPath, err := deps.Ensure(deps.Terraform)
	if err != nil {
		return fmt.Errorf("unable to cache terraform: %w", err)
	}

	if err := runTerraform(tfPath, opts.Dir, "init"); err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	args := []string{"apply"}
	if opts.AutoApprove {
		args = append(args, "-auto-approve")
	}

	if err := runTerraform(tfPath, opts.Dir, args...); err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nDeploy complete.\n\n"+
		"To connect to the cluster, create a kubeconfig file by running:\n\n"+
		"keights kubeconfig --cluster-name <cluster> --output <kubeconfig>\n\n"+
		"or run:\n\n"+
		"keights kubectl --cluster-name <cluster> -- [kubectl args]\n\n")

	return nil
}

func runTerraform(tfPath, dir string, args ...string) error {
	fullArgs := append(
		[]string{"-chdir=" + dir}, args...,
	)
	cmd := exec.Command(tfPath, fullArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func validateDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("directory %s does not exist", dir)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.tf"))
	if err != nil {
		return fmt.Errorf("unable to check for terraform files: %w", err)
	}
	if len(matches) == 0 {
		return fmt.Errorf(
			"no .tf files found in %s", dir,
		)
	}

	return nil
}
