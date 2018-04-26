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

package signal

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cloudboss/keights/pkg/helpers"
)

func constructURL(stackName, status, resource, myID, region string) string {
	params := url.Values{}
	params.Set("Action", "SignalResource")
	params.Set("Version", "2010-05-15")
	params.Set("ContentType", "JSON")
	params.Set("StackName", stackName)
	params.Set("Status", status)
	params.Set("LogicalResourceId", resource)
	params.Set("UniqueId", myID)
	earl := url.URL{
		Scheme:   "https",
		Host:     fmt.Sprintf("cloudformation.%s.amazonaws.com", region),
		RawQuery: params.Encode(),
	}
	return earl.String()
}

func metadata(what string) ([]byte, error) {
	path := fmt.Sprintf("/latest/dynamic/instance-identity/%s", what)
	earl := fmt.Sprintf("http://169.254.169.254/%s", path)
	response, err := http.Get(earl)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata service returned %s", response.Status)
	}
	defer response.Body.Close()
	return ioutil.ReadAll(response.Body)
}

func signature() (string, error) {
	identityDoc, err := metadata("document")
	if err != nil {
		return "", err
	}
	identitySig, err := metadata("signature")
	if err != nil {
		return "", err
	}
	identitySigStripped := bytes.Replace(identitySig, []byte("\n"), []byte(""), -1)
	b64IdentityDoc := base64.StdEncoding.EncodeToString(identityDoc)
	headerVal := fmt.Sprintf("CFN_V1 %s:%s", b64IdentityDoc, string(identitySigStripped))
	return headerVal, nil
}

func DoIt(stackName, status, resource string) error {
	sess := session.New()
	// A bit of duplication here, since we call the metadata service to get
	// the whole document as bytes above but do not parse it for the instance ID.
	myID, err := helpers.MyID(sess)
	if err != nil {
		return err
	}
	sig, err := signature()
	if err != nil {
		return err
	}
	earl := constructURL(stackName, status, resource, myID, os.Getenv("AWS_REGION"))
	client := http.DefaultClient
	request, err := http.NewRequest("GET", earl, nil)
	request.Header.Add("Authorization", sig)
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return err
}
