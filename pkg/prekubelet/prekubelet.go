// Copyright © 2018 Joseph Wright <joseph@cloudboss.co>
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

package prekubelet

import (
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/cloudboss/keights/pkg/helpers"
)

const (
	healthz = "http://localhost:8080/healthz"
)

func start(command string, args ...string) (*exec.Cmd, error) {
	cmd := exec.Command(command, args...)
	err := cmd.Start()
	if err != nil {
		return nil, err
	}
	return cmd, nil
}

func waitAndKill(cmd *exec.Cmd) error {
	defer cmd.Process.Kill()
	return helpers.WaitFor(30*time.Minute, func() error {
		response, err := http.Get(healthz)
		if err != nil {
			return err
		}
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("health check returned %s", response.Status)
		}
		return nil
	})
}

func DoIt(manifestsPath string) error {
	cmd, err := start("kubelet", "--pod-manifest-path", manifestsPath)
	if err != nil {
		return err
	}
	return waitAndKill(cmd)
}
