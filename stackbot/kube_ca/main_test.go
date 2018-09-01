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

	"github.com/cloudboss/keights/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleCreateSuccess(t *testing.T) {
	var tests = []struct {
		setStoreParameter func(s *mocks.Whisperer)
		err               error
	}{
		{
			func(w *mocks.Whisperer) {
				w.On("StoreParameter", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			nil,
		},
		{
			func(w *mocks.Whisperer) {
				err := fmt.Errorf("oops")
				w.On("StoreParameter", mock.Anything, mock.Anything, mock.Anything).Return(err)
			},
			fmt.Errorf("oops"),
		},
	}
	for _, test := range tests {
		whisp := &mocks.Whisperer{}
		props := resourceProperties{NumInstances: "3"}
		test.setStoreParameter(whisp)
		err := handleCreate(props, whisp)
		assert.Equal(t, err, test.err)
		whisp.AssertExpectations(t)
	}
}
