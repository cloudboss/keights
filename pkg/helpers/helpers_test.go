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

package helpers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteIfChanged(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "keights")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(tempFile.Name())

	fileInfo, err := tempFile.Stat()
	if err != nil {
		t.Error(err)
	}
	err = tempFile.Close()
	if err != nil {
		t.Error(err)
	}

	err = WriteIfChanged(tempFile.Name(), []byte(""), 0644)
	if err != nil {
		t.Error(err)
	}

	noChangeFileInfo, err := os.Stat(tempFile.Name())
	if !os.SameFile(fileInfo, noChangeFileInfo) {
		t.Error("File was written with no change")
	}

	err = WriteIfChanged(tempFile.Name(), []byte("hello"), 0644)
	if err != nil {
		t.Error(err)
	}

	changeFileInfo, err := os.Stat(tempFile.Name())
	if os.SameFile(fileInfo, changeFileInfo) {
		t.Error("File was not written atomically")
	}
}

func TestAtomicWrite(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "keights")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(tempFile.Name())

	fileInfo, err := tempFile.Stat()
	if err != nil {
		t.Error(err)
	}
	err = tempFile.Close()
	if err != nil {
		t.Error(err)
	}

	err = AtomicWrite(tempFile.Name(), []byte(""), 0644)
	if err != nil {
		t.Error(err)
	}

	newFileInfo, err := os.Stat(tempFile.Name())
	if os.SameFile(fileInfo, newFileInfo) {
		t.Error("File write was not atomic")
	}
}

func TestAppendToFile(t *testing.T) {
	var testCases = []struct {
		name         string
		oldContents  string
		extraContent string
		newContent   string
	}{
		{
			"empty",
			"",
			"",
			"",
		},
		{
			"empty-orig",
			"",
			"hello",
			"hello",
		},
		{
			"nonempty-orig",
			"hello",
			"",
			"hello",
		},
		{
			"nonempty-both",
			"hello",
			"hello",
			"hello\nhello",
		},
	}
	for _, tc := range testCases {
		tempDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Errorf("error creating temp dir: %v", err)
		}
		path := fmt.Sprintf("%s/%s", tempDir, tc.name)
		fmt.Printf(path)
		err = ioutil.WriteFile(path, []byte(tc.oldContents), 0644)
		if err != nil {
			t.Errorf("error writing file: %v", err)
		}
		err = AppendToFile(path, []byte(tc.extraContent), 0644)
		if err != nil {
			t.Errorf("error reading updated file: %v", err)
		}
		newContent, err := ioutil.ReadFile(path)
		expectedNewContent := []byte(tc.newContent)
		if !bytes.Equal(newContent, expectedNewContent) {
			assert.Equal(t, newContent, expectedNewContent)
		}
		err = os.RemoveAll(tempDir)
		if err != nil {
			t.Errorf("error removing temp dir: %v", err)
		}
	}
}
