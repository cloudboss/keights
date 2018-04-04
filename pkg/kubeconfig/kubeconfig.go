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

package kubeconfig

import (
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func newConfig(serverURL, clusterName, userName, certDir string) *api.Config {
	contextName := fmt.Sprintf("%s@%s", userName, clusterName)
	caCert := fmt.Sprintf("%s/ca.crt", certDir)

	return &api.Config{
		Clusters: map[string]*api.Cluster{
			clusterName: {
				Server:               serverURL,
				CertificateAuthority: caCert,
			},
		},
		Contexts: map[string]*api.Context{
			contextName: {
				Cluster:  clusterName,
				AuthInfo: userName,
			},
		},
		AuthInfos:      map[string]*api.AuthInfo{},
		CurrentContext: contextName,
	}
}

func newConfigWithCert(serverURL string, clusterName string, userName, certDir string) *api.Config {
	config := newConfig(serverURL, clusterName, userName, certDir)
	config.AuthInfos[userName] = &api.AuthInfo{
		ClientKey:         fmt.Sprintf("%s/%s.key", certDir, userName),
		ClientCertificate: fmt.Sprintf("%s/%s.crt", certDir, userName),
	}
	return config
}

func newConfigWithToken(serverURL string, clusterName string, userName, certDir, token string) *api.Config {
	config := newConfig(serverURL, clusterName, userName, certDir)
	config.AuthInfos[userName] = &api.AuthInfo{
		TokenFile: token,
	}
	return config
}

func DoIt(certDir, serverURL, clusterName, bootstrapToken, bootstrapPath, kubeRouterPath string) error {
	kubeletBootstrapConfig := newConfigWithToken(serverURL, clusterName, "kubelet-bootstrap", certDir, bootstrapToken)
	err := clientcmd.WriteToFile(*kubeletBootstrapConfig, bootstrapPath)
	if err != nil {
		return err
	}
	kubeRouterConfig := newConfigWithCert(serverURL, clusterName, "kube-router", certDir)
	return clientcmd.WriteToFile(*kubeRouterConfig, kubeRouterPath)
}
