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

package cfnsignal

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
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

func dynamic(ctx context.Context, imdsClient *imds.Client, path string) ([]byte, error) {
	response, err := imdsClient.GetDynamicData(ctx, &imds.GetDynamicDataInput{Path: path})
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Content.Close() }()
	return io.ReadAll(response.Content)
}

func metadata(ctx context.Context, imdsClient *imds.Client, path string) ([]byte, error) {
	response, err := imdsClient.GetMetadata(ctx, &imds.GetMetadataInput{Path: path})
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Content.Close() }()
	return io.ReadAll(response.Content)
}

func cfnHeader(ctx context.Context, imdsClient *imds.Client) (string, error) {
	identityDoc, err := dynamic(ctx, imdsClient, "instance-identity/document")
	if err != nil {
		return "", err
	}
	identitySig, err := dynamic(ctx, imdsClient, "instance-identity/signature")
	if err != nil {
		return "", err
	}
	identitySigStripped := bytes.ReplaceAll(identitySig, []byte("\n"), []byte(""))
	b64IdentityDoc := base64.StdEncoding.EncodeToString(identityDoc)
	headerVal := fmt.Sprintf("CFN_V1 %s:%s", b64IdentityDoc, string(identitySigStripped))
	return headerVal, nil
}

func DoIt(ctx context.Context, stackName, status, resource string) error {
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	imdsClient := imds.NewFromConfig(awsConfig)
	doc, err := imdsClient.GetInstanceIdentityDocument(ctx,
		&imds.GetInstanceIdentityDocumentInput{},
	)
	if err != nil {
		return err
	}
	header, err := cfnHeader(ctx, imdsClient)
	if err != nil {
		return err
	}
	region, err := metadata(ctx, imdsClient, "placement/region")
	if err != nil {
		return err
	}
	earl := constructURL(stackName, status, resource, doc.InstanceID, string(region))
	request, err := http.NewRequest("GET", earl, nil)
	if err != nil {
		return err
	}
	request.Header.Add("Authorization", header)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return err
}
