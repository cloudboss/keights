// Copyright 2017 Joseph Wright <joseph@cloudboss.co>
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

package response

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
	"github.com/eawsy/aws-lambda-go-event/service/lambda/runtime/event/cloudformationevt"
)

const (
	Failed  = "FAILED"
	Success = "SUCCESS"
)

type responder struct {
	responseURL  string
	responseBody ResponseBody
}

type Responder interface {
	FireSuccess() error
	FireFailed(reason string) error
}

func (r *responder) FireSuccess() error {
	r.responseBody.Status = Success
	return Respond(r.responseURL, r.responseBody)
}

func (r *responder) FireFailed(reason string) error {
	r.responseBody.Status = Failed
	r.responseBody.Reason = reason
	return Respond(r.responseURL, r.responseBody)
}

func NewResponder(event *cloudformationevt.Event, ctx *runtime.Context) Responder {
	return &responder{
		responseURL: event.ResponseURL,
		responseBody: ResponseBody{
			Reason:             "",
			PhysicalResourceID: ctx.LogStreamName,
			StackID:            event.StackID,
			RequestID:          event.RequestID,
			LogicalResourceID:  event.LogicalResourceID,
			Data:               map[string]string{},
		},
	}
}

type ResponseBody struct {
	Status             string
	Reason             string
	PhysicalResourceID string `json:"PhysicalResourceId"`
	StackID            string `json:"StackId"`
	RequestID          string `json:"RequestId"`
	LogicalResourceID  string `json:"LogicalResourceId"`
	Data               map[string]string
}

func NewResponseBody(event *cloudformationevt.Event, ctx *runtime.Context) ResponseBody {
	return ResponseBody{
		Reason:             "",
		PhysicalResourceID: ctx.LogStreamName,
		StackID:            event.StackID,
		RequestID:          event.RequestID,
		LogicalResourceID:  event.LogicalResourceID,
		Data:               map[string]string{},
	}
}

func Respond(url string, body ResponseBody) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	client := &http.Client{}
	req.Header.Add("Content-Type", "")
	req.Header.Add("Content-Length", string(len(jsonBody)))
	_, err = client.Do(req)
	return err
}

func FireResponse(url string, body ResponseBody) {
	// No return value is sent because if this fails we can't recover.
	err := Respond(url, body)
	if err != nil {
		fmt.Println(err.Error())
	}
}
