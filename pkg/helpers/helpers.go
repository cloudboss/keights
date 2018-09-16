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

package helpers

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
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
	dir := filepath.Dir(path)
	tempDir, err := ioutil.TempDir(dir, ".keights")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	base := filepath.Base(path)
	tempFile := fmt.Sprintf("%s/%s", tempDir, base)
	err = ioutil.WriteFile(tempFile, contents, mode)
	if err != nil {
		return err
	}
	return os.Rename(tempFile, path)
}

func WriteIfChanged(path string, contents []byte, mode os.FileMode) error {
	var original []byte
	var err error
	original, err = ioutil.ReadFile(path)
	if err != nil {
		if e, ok := err.(*os.PathError); ok && e.Err == syscall.ENOENT {
			original = []byte{}
		} else {
			return err
		}
	}
	if bytes.Equal(contents, original) {
		return nil
	}
	return AtomicWrite(path, contents, mode)
}

func MapKeys(mapping map[string]string) []string {
	keys := []string{}
	for key := range mapping {
		keys = append(keys, key)
	}
	return keys
}

func SortMapKeys(mapping map[string]string) []string {
	keys := MapKeys(mapping)
	sort.Strings(keys)
	return keys
}

func AsgName(sess *session.Session) (*string, error) {
	metadata := ec2metadata.New(sess)
	identity, err := metadata.GetInstanceIdentityDocument()
	if err != nil {
		return nil, err
	}
	asgClient := autoscaling.New(sess)
	input := &autoscaling.DescribeAutoScalingInstancesInput{
		InstanceIds: []*string{&identity.InstanceID},
	}
	output, err := asgClient.DescribeAutoScalingInstances(input)
	if err != nil {
		return nil, err
	}
	for _, instance := range output.AutoScalingInstances {
		return instance.AutoScalingGroupName, nil
	}
	return nil, fmt.Errorf("Autoscaling group name not found")
}

func WaitFor(duration time.Duration, checker func() error) error {
	var err error
	c := make(chan bool)
	go func() {
		for {
			err = checker()
			if err != nil {
				time.Sleep(5 * time.Second)
			} else {
				c <- true
			}
		}
	}()
	select {
	case <-c:
		return nil
	case <-time.After(duration):
		return err
	}
}

func InputToMapping(inputFile string) (map[string]string, error) {
	fd, err := os.Open(inputFile)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	mapping := map[string]string{}
	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Malformed input file")
		}
		mapping[parts[0]] = strings.Trim(parts[1], "\n")
	}
	return mapping, nil
}

func MyIP(sess *session.Session) (string, error) {
	metadata := ec2metadata.New(sess)
	identity, err := metadata.GetInstanceIdentityDocument()
	if err != nil {
		return "", err
	}
	return identity.PrivateIP, nil
}

func MyID(sess *session.Session) (string, error) {
	metadata := ec2metadata.New(sess)
	identity, err := metadata.GetInstanceIdentityDocument()
	if err != nil {
		return "", err
	}
	return identity.InstanceID, nil
}

func MyIndex(mapping map[string]string, ip string) string {
	for k, v := range mapping {
		if v == ip {
			return k
		}
	}
	return "-1"
}

func IsIndexOne(sess *session.Session, inputFile string) (bool, error) {
	mapping, err := InputToMapping(inputFile)
	if err != nil {
		return false, err
	}
	myIP, err := MyIP(sess)
	if err != nil {
		return false, err
	}
	myIndex := MyIndex(mapping, myIP)
	return myIndex == "1", nil
}

func IdempotentDo(checker func() (bool, error), doer func() error) error {
	affirmative, err := checker()
	if err != nil {
		return err
	}
	if !affirmative {
		return doer()
	}
	return nil
}
