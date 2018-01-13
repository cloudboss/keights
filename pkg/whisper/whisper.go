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

package whisper

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/cloudboss/keights/pkg/helpers"
)

func getSecret(ssmClient ssmiface.SSMAPI, path string) (*string, error) {
	withDecryption := true
	response, err := ssmClient.GetParameters(&ssm.GetParametersInput{
		Names:          []*string{&path},
		WithDecryption: &withDecryption,
	})
	if err != nil {
		return nil, err
	}
	for _, parameter := range response.Parameters {
		return parameter.Value, nil
	}
	return nil, fmt.Errorf("Secret %s not found", path)
}

func getSecrets(ssmClient ssmiface.SSMAPI, pathSpecs []map[string]string) ([]map[string]*string, error) {
	maps := []map[string]*string{}
	for _, pathSpec := range pathSpecs {
		secret, err := getSecret(ssmClient, pathSpec["path"])
		if err != nil {
			return nil, err
		}
		maps = append(maps, map[string]*string{
			"secret": secret, "dest": aws.String(pathSpec["dest"]),
		})
	}
	return maps, nil
}

func writeSecrets(secretSpecs []map[string]*string) error {
	for _, secretSpec := range secretSpecs {
		err := helpers.WriteIfChanged(*secretSpec["dest"], []byte(*secretSpec["secret"]), 0400)
		if err != nil {
			return err
		}
	}
	return nil
}

func parsePaths(paths []string) ([]map[string]string, error) {
	maps := []map[string]string{}
	for _, path := range paths {
		parts := strings.Split(path, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Malformed path %s", path)
		}
		maps = append(maps, map[string]string{"path": parts[0], "dest": parts[1]})
	}
	return maps, nil
}

func DoIt(paths []string) error {
	sess, err := session.NewSession()
	if err != nil {
		return err
	}
	ssmClient := ssm.New(sess)
	pathSpecs, err := parsePaths(paths)
	if err != nil {
		return err
	}
	secretSpecs, err := getSecrets(ssmClient, pathSpecs)
	if err != nil {
		return err
	}
	return writeSecrets(secretSpecs)
}
