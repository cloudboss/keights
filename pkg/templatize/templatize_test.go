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

package templatize

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVarsToMap(t *testing.T) {
	var cases = []struct {
		input  []string
		output map[string]interface{}
		error
	}{
		{
			[]string{},
			map[string]interface{}{},
			nil,
		},
		{
			[]string{"k=v"},
			map[string]interface{}{"k": "v"},
			nil,
		},
		{
			[]string{"k=v", "l=w"},
			map[string]interface{}{"k": "v", "l": "w"},
			nil,
		},
		{
			[]string{"k=v=w"},
			map[string]interface{}{"k": "v=w"},
			nil,
		},
		{
			[]string{"k=v", "list=,"},
			map[string]interface{}{
				"k":    "v",
				"list": []string{},
			},
			nil,
		},
		{
			[]string{"k=v", "list=one,two,three"},
			map[string]interface{}{
				"k":    "v",
				"list": []string{"one", "two", "three"},
			},
			nil,
		},
		{
			[]string{"k=v=w,"},
			map[string]interface{}{"k": []string{"v=w"}},
			nil,
		},
		{
			[]string{"k=v=w,a=b=c"},
			map[string]interface{}{"k": []string{"v=w", "a=b=c"}},
			nil,
		},
		{
			[]string{"k"},
			map[string]interface{}{},
			fmt.Errorf("Malformed variable k"),
		},
	}
	for _, tc := range cases {
		result, err := VarsToMap(tc.input)
		if err != nil {
			assert.Equal(t, err, tc.error)
			continue
		}
		assert.Equal(t, result, tc.output)
	}
}
