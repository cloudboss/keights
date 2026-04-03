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

package environment

import (
	"fmt"
	"os"
	"strings"
)

func EnsureEnvironment(environment []string) (map[string]string, error) {
	ensured := make(map[string]string)
	missing := []string{}
	for _, envVar := range environment {
		value := os.Getenv(envVar)
		if value == "" {
			missing = append(missing, envVar)
		} else {
			ensured[envVar] = value
		}
	}
	lenMissing := len(missing)
	if lenMissing > 0 {
		var s string
		if lenMissing > 1 {
			s = "s"
		}
		err := fmt.Errorf("missing environment variable%s: %s", s, strings.Join(missing, ", "))
		return nil, err
	}
	return ensured, nil
}
