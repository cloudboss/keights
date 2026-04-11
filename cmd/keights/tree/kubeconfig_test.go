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

package tree

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildKubeconfig(t *testing.T) {
	caPEM := []byte("-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n")
	kc := buildKubeconfig("test-cluster", "test.example.com", caPEM, "")

	assert.Equal(t, "test-cluster", kc.CurrentContext)

	cluster, ok := kc.Clusters["test-cluster"]
	require.True(t, ok)
	assert.Equal(t, "https://test.example.com", cluster.Server)
	assert.Equal(t, caPEM, cluster.CertificateAuthorityData)

	ctx, ok := kc.Contexts["test-cluster"]
	require.True(t, ok)
	assert.Equal(t, "test-cluster", ctx.Cluster)
	assert.Equal(t, "test-cluster", ctx.AuthInfo)

	auth, ok := kc.AuthInfos["test-cluster"]
	require.True(t, ok)
	require.NotNil(t, auth.Exec)
	assert.Equal(t, "keights", auth.Exec.Command)
	assert.Equal(t, []string{"token", "--cluster-name", "test-cluster"}, auth.Exec.Args)
}

func TestBuildKubeconfigWithRegion(t *testing.T) {
	caPEM := []byte("-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n")
	kc := buildKubeconfig("test-cluster", "test.example.com", caPEM, "us-east-1")

	auth := kc.AuthInfos["test-cluster"]
	assert.Equal(t,
		[]string{"token", "--cluster-name", "test-cluster", "--region", "us-east-1"},
		auth.Exec.Args,
	)
}
