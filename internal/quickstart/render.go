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

package quickstart

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ModuleSourceForVersion returns the URL for the keights Terraform module
// tarball matching the given keights version (e.g. "v2.0.0"). The version is
// injected at build time via the Makefile's VERSION variable.
func ModuleSourceForVersion(version string) string {
	return fmt.Sprintf("https://github.com/cloudboss/keights/releases/download/%s/"+
		"keights-terraform-%s.tar.gz",
		version, version)
}

//go:embed templates/*.tmpl
var templatesFS embed.FS

// RenderOptions controls non-answer inputs to template rendering.
type RenderOptions struct {
	// ModuleSource is the Terraform module source URL written into the
	// generated main.tf.
	ModuleSource string
}

// templateData is the view passed to text/template.
type templateData struct {
	ModuleSource   string
	ClusterName    string
	Region         string
	VPCID          string
	AMIName        string
	AMIOwnerID     string
	IRSAEnabled    bool
	KMSKeyID       string
	SSHKeyPair     string
	APICIDRs       []string
	NodePortsCIDRs []string
	SSHCIDRs       []string
	ControlPlane   ControlPlane
	NodeGroups     []NodeGroup
}

// Render writes main.tf and vars.tf into outputDir using the answers.
// outputDir is created if it does not exist; existing files are overwritten.
func Render(a Answers, opts RenderOptions) error {
	if err := os.MkdirAll(a.OutputDir, 0o755); err != nil {
		return fmt.Errorf("unable to create output directory: %w", err)
	}

	data := templateData{
		ModuleSource:   opts.ModuleSource,
		ClusterName:    a.ClusterName,
		Region:         a.Region,
		VPCID:          a.VPCID,
		AMIName:        a.AMIName,
		AMIOwnerID:     a.AMIOwnerID,
		IRSAEnabled:    a.IRSAEnabled,
		KMSKeyID:       a.KMSKeyID,
		SSHKeyPair:     a.SSHKeyPair,
		APICIDRs:       []string{a.AccessCIDR},
		NodePortsCIDRs: a.NodePortsCIDRs,
		SSHCIDRs:       a.SSHCIDRs,
		ControlPlane:   a.ControlPlane,
		NodeGroups:     a.NodeGroups,
	}

	files := []string{"main.tf.tmpl", "vars.tf.tmpl"}
	for _, name := range files {
		out, err := renderTemplate(name, data)
		if err != nil {
			return err
		}
		dst := filepath.Join(a.OutputDir, strings.TrimSuffix(name, ".tmpl"))
		if err := os.WriteFile(dst, out, 0o644); err != nil {
			return fmt.Errorf("unable to write %s: %w", dst, err)
		}
	}
	return nil
}

func renderTemplate(name string, data templateData) ([]byte, error) {
	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"quotedList": quotedList,
	}).ParseFS(templatesFS, "templates/"+name)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("unable to render %s: %w", name, err)
	}
	return buf.Bytes(), nil
}

func quotedList(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	parts := make([]string, len(items))
	for i, s := range items {
		parts[i] = fmt.Sprintf("%q", s)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
