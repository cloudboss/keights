// Copyright © 2017 Joseph Wright <joseph@cloudboss.co>
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

package helpers

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"syscall"
)

type CommandOutput struct {
	ExitStatus int
	Stdout     string
	Stderr     string
}

func RunCommand(command string, args ...string) *CommandOutput {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(command, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Run()
	waitStatus := cmd.ProcessState.Sys().(syscall.WaitStatus)
	return &CommandOutput{
		ExitStatus: waitStatus.ExitStatus(),
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
	}
}

func AtomicWrite(path string, contents []byte, mode os.FileMode) error {
	tempFile, err := ioutil.TempFile("", "keights")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempFile.Name())

	err = ioutil.WriteFile(tempFile.Name(), contents, mode)
	if err != nil {
		return err
	}
	return os.Rename(tempFile.Name(), path)
}

func WriteIfChanged(path string, contents []byte) error {
	var original []byte
	var err error
	if _, err = os.Stat(path); os.IsNotExist(err) {
		original = []byte{}
	} else {
		original, err = ioutil.ReadFile(path)
		if err != nil {
			return err
		}
	}
	if bytes.Equal(contents, original) {
		return nil
	}
	return AtomicWrite(path, contents, os.FileMode(0644))
}

func SortMapKeys(mapping map[string]string) []string {
	keys := []string{}
	for key, _ := range mapping {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
