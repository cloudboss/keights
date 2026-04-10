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
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

type Format int

const (
	Binary Format = iota
	Zip
)

type Platform struct {
	OS   string
	Arch string
}

type Dependency struct {
	Name    string
	Version string
	Format  Format
	URLs    map[Platform]string
	SHA256  map[Platform]string
}

func cacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to determine cache directory: %w", err)
		}
		dir = filepath.Join(home, ".cache")
	}
	return filepath.Join(dir, "keights", "bin"), nil
}

func (d Dependency) BinaryName() string {
	return fmt.Sprintf("%s-%s", d.Name, d.Version)
}

func Ensure(dep Dependency) (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}

	binPath := filepath.Join(dir, dep.BinaryName())
	if _, err := os.Stat(binPath); err == nil {
		return binPath, nil
	}

	platform := Platform{runtime.GOOS, runtime.GOARCH}

	url, ok := dep.URLs[platform]
	if !ok {
		return "", fmt.Errorf("unsupported platform %s/%s for %s",
			runtime.GOOS, runtime.GOARCH, dep.Name)
	}

	expectedHash, ok := dep.SHA256[platform]
	if !ok {
		return "", fmt.Errorf("no checksum for %s on %s/%s",
			dep.Name, runtime.GOOS, runtime.GOARCH)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("unable to create cache directory: %w", err)
	}

	tmpFile, err := download(url, dir)
	if err != nil {
		return "", fmt.Errorf("unable to download %s: %w", dep.Name, err)
	}
	defer os.Remove(tmpFile)

	if err := verifyChecksum(tmpFile, expectedHash); err != nil {
		return "", fmt.Errorf("unable to verify %s: %w", dep.Name, err)
	}

	if err := install(dep.Format, tmpFile, binPath, dep.Name); err != nil {
		return "", fmt.Errorf("unable to install %s: %w", dep.Name, err)
	}

	if err := os.Chmod(binPath, 0o755); err != nil {
		return "", fmt.Errorf("unable to set permissions on %s: %w", dep.Name, err)
	}

	return binPath, nil
}

func EnsureAll() (map[string]string, error) {
	paths := make(map[string]string, len(All))
	for _, dep := range All {
		path, err := Ensure(dep)
		if err != nil {
			return nil, err
		}
		paths[dep.Name] = path
	}
	return paths, nil
}

func download(url, dir string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received http %d from %s", resp.StatusCode, url)
	}

	tmp, err := os.CreateTemp(dir, "keights-dep-*")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}

	return tmp.Name(), nil
}

func verifyChecksum(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch: got %s, wanted %s", actual, expected)
	}
	return nil
}

func install(format Format, srcPath, destPath, name string) error {
	switch format {
	case Binary:
		return os.Rename(srcPath, destPath)
	case Zip:
		return extractZip(srcPath, destPath, name)
	default:
		return fmt.Errorf("unsupported format: %d", format)
	}
}

func extractZip(zipPath, destPath, binaryName string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name != binaryName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		out, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, rc)
		return err
	}

	return fmt.Errorf("%s not found in zip archive", binaryName)
}
