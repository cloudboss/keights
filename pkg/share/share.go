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

package share

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/cloudboss/keights/pkg/helpers"
)

const (
	HostsHeader = "# Before keights host entries"
	HostsFooter = "# After keights host entries"
)

func SplitHosts(hostsFile string) ([]byte, []string, []string, error) {
	var beforeLines, afterLines []string
	var before, after bool

	original, err := ioutil.ReadFile(hostsFile)
	if err != nil {
		return original, beforeLines, afterLines, err
	}

	before = true
	for _, line := range strings.Split(string(original), "\n") {
		if line == HostsHeader {
			before = false
			continue
		}
		if line == HostsFooter {
			after = true
			continue
		}
		if before && after {
			err := fmt.Errorf("Malformed hosts file")
			return original, beforeLines, afterLines, err
		}
		if !before && !after {
			continue
		}
		if before {
			beforeLines = append(beforeLines, line)
		}
		if after {
			afterLines = append(afterLines, line)
		}
	}
	return original, beforeLines, afterLines, nil
}

func FormatHosts(mapping map[string]string, prefix, domain string) []string {
	hosts := []string{}
	keys := helpers.SortMapKeys(mapping)
	for _, key := range keys {
		host := fmt.Sprintf("%s %s-%s.%s", mapping[key], prefix, key, domain)
		hosts = append(hosts, host)
	}
	return hosts
}

func NewHosts(before, after, shareHosts []string) []byte {
	hosts := before
	hosts = append(hosts, HostsHeader)
	hosts = append(hosts, shareHosts...)
	hosts = append(hosts, HostsFooter)
	hosts = append(hosts, after...)
	hostsBytes := []byte(strings.Join(hosts, "\n"))
	lastByte := hostsBytes[len(hostsBytes)-1]
	if string(lastByte) != "\n" {
		hostsBytes = append(hostsBytes, []byte("\n")...)
	}
	return hostsBytes
}

func DoIt(inputFile, hostsFile, prefix, domain string) error {
	original, before, after, err := SplitHosts(hostsFile)
	if err != nil {
		return err
	}
	mapping, err := helpers.InputToMapping(inputFile)
	if err != nil {
		return err
	}
	shareHosts := FormatHosts(mapping, prefix, domain)
	newHosts := NewHosts(before, after, shareHosts)
	if !bytes.Equal(original, newHosts) {
		return ioutil.WriteFile(hostsFile, newHosts, os.FileMode(0644))
	}
	return nil
}
