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

package main

import (
	"fmt"
	"testing"

	"github.com/cloudboss/stackhand/mocks"
	"github.com/stretchr/testify/mock"
)

func TestHandleCreateSuccess(t *testing.T) {
	var tests = []struct {
		setResponse       func(r *mocks.Responder)
		setStoreParameter func(s *mocks.Whisperer)
	}{
		{
			func(r *mocks.Responder) {
				r.On("FireSuccess").Return(nil)
			},
			func(w *mocks.Whisperer) {
				w.On("StoreParameter", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			func(r *mocks.Responder) {
				err := fmt.Errorf("oops")
				r.On("FireFailed", mock.Anything).Return(err)
			},
			func(w *mocks.Whisperer) {
				err := fmt.Errorf("oops")
				w.On("StoreParameter", mock.Anything, mock.Anything, mock.Anything).Return(err)
			},
		},
	}
	for _, test := range tests {
		responder := &mocks.Responder{}
		whisp := &mocks.Whisperer{}
		props := resourceProperties{}
		test.setResponse(responder)
		test.setStoreParameter(whisp)
		_ = handleCreate(props, whisp, responder)
		whisp.AssertExpectations(t)
		responder.AssertExpectations(t)
	}
}
