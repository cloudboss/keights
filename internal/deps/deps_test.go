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

package deps

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheHit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)

	dep := Dependency{
		Name:    "test-tool",
		Version: "1.0.0",
	}

	binDir := filepath.Join(dir, "keights", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))

	binPath := filepath.Join(binDir, "test-tool-1.0.0")
	require.NoError(t, os.WriteFile(binPath, []byte("binary"), 0o755))

	path, err := Ensure(dep)
	require.NoError(t, err)
	assert.Equal(t, binPath, path)
}

func TestUnsupportedPlatform(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)

	dep := Dependency{
		Name:    "test-tool",
		Version: "1.0.0",
		URLs:    map[Platform]string{},
		SHA256:  map[Platform]string{},
	}

	_, err := Ensure(dep)
	assert.ErrorContains(t, err, "unsupported platform")
}

func TestChecksumMismatch(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "testfile")
	require.NoError(t, os.WriteFile(tmpFile, []byte("data"), 0o644))

	err := verifyChecksum(tmpFile, "0000000000000000000000000000000000000000000000000000000000000000")
	assert.ErrorContains(t, err, "checksum mismatch")
}

func TestChecksumMatch(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "testfile")
	require.NoError(t, os.WriteFile(tmpFile, []byte("data"), 0o644))

	// SHA256 of "data"
	err := verifyChecksum(
		tmpFile,
		"3a6eb0790f39ac87c94f3856b2dd2c5d110e6811602261a9a923d3bb23adc8b7",
	)
	assert.NoError(t, err)
}

func TestURLSelectionForAllDependencies(t *testing.T) {
	platform := Platform{runtime.GOOS, runtime.GOARCH}
	for _, dep := range All {
		_, ok := dep.URLs[platform]
		assert.True(t, ok, "%s missing URL for %s/%s",
			dep.Name, runtime.GOOS, runtime.GOARCH)
		_, ok = dep.SHA256[platform]
		assert.True(t, ok, "%s missing SHA256 for %s/%s",
			dep.Name, runtime.GOOS, runtime.GOARCH)
	}
}

func TestBinaryName(t *testing.T) {
	dep := Dependency{Name: "kubectl", Version: "1.34.5"}
	assert.Equal(t, "kubectl-1.34.5", dep.BinaryName())
}
