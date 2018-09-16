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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cloudboss/keights/pkg/helpers"
)

func MergeMaps(first, second map[string]interface{}) map[string]interface{} {
	dest := make(map[string]interface{})
	for k, v := range first {
		dest[k] = v
	}
	for k, v := range second {
		dest[k] = v
	}
	return dest
}

func StringToList(commaSeparatedList string) []string {
	list := []string{}
	for _, part := range strings.Split(commaSeparatedList, ",") {
		item := strings.TrimSpace(part)
		if len(item) > 0 {
			list = append(list, item)
		}
	}
	return list
}

func VarsToMap(vars []string) (map[string]interface{}, error) {
	mapping := make(map[string]interface{})
	for _, v := range vars {
		parts := strings.Split(v, "=")
		if len(parts) < 2 {
			return nil, fmt.Errorf("Malformed variable %v", v)
		}
		leftSide := parts[0]
		rightSide := strings.Join(parts[1:], "=")
		if strings.Contains(rightSide, ",") {
			mapping[leftSide] = StringToList(rightSide)
		} else {
			mapping[leftSide] = rightSide
		}
	}
	return mapping, nil
}

func Render(templateFile string, mapping map[string]interface{}) (bytes.Buffer, error) {
	var b bytes.Buffer
	f, err := os.Open(templateFile)
	if err != nil {
		return b, err
	}
	defer f.Close()
	templateBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return b, err
	}
	funcs := template.FuncMap{
		"join": strings.Join,
		"keys": helpers.SortMapKeys,
	}
	tpl, err := template.New("output").Funcs(funcs).Parse(string(templateBytes))
	if err != nil {
		return b, err
	}
	err = tpl.Execute(&b, mapping)
	return b, err
}

func WriteTemplate(buf bytes.Buffer, dest, owner, group string, mode int) error {
	if dest == "-" {
		_, err := buf.WriteTo(os.Stdout)
		return err
	}
	return helpers.WriteIfChanged(dest, buf.Bytes(), os.FileMode(mode))
}

func DoIt(templateFile, inputFile, ip, dest, owner, group string, mode int, vars []string) error {
	var myIP string
	inputFileMapping, err := helpers.InputToMapping(inputFile)
	if err != nil {
		return err
	}
	cliMapping, err := VarsToMap(vars)
	if err != nil {
		return err
	}
	if ip != "" {
		myIP = ip
	} else {
		sess, err := session.NewSession()
		if err != nil {
			return err
		}
		myIP, err = helpers.MyIP(sess)
		if err != nil {
			return err
		}
	}
	myIndex := helpers.MyIndex(inputFileMapping, myIP)
	mapping := MergeMaps(cliMapping, map[string]interface{}{
		"MyIP":    myIP,
		"MyIndex": myIndex,
		"HostMap": inputFileMapping,
	})
	rendered, err := Render(templateFile, mapping)
	if err != nil {
		return err
	}
	return WriteTemplate(rendered, dest, owner, group, mode)
}
