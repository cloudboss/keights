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

package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDir_NonexistentDir(t *testing.T) {
	err := validateDir("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestValidateDir_NotADirectory(t *testing.T) {
	f, err := os.CreateTemp("", "not-a-dir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Close()

	err = validateDir(f.Name())
	if err == nil {
		t.Fatal("expected error for non-directory path")
	}
}

func TestValidateDir_NoTerraformFiles(t *testing.T) {
	dir := t.TempDir()
	err := validateDir(dir)
	if err == nil {
		t.Fatal("expected error for directory with no .tf files")
	}
}

func TestValidateDir_WithTerraformFiles(t *testing.T) {
	dir := t.TempDir()
	tfFile := filepath.Join(dir, "main.tf")
	if err := os.WriteFile(tfFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateDir(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
