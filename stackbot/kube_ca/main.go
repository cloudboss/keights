// Copyright 2022 Joseph Wright <joseph@cloudboss.co>
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
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cloudboss/keights/pkg/helpers"
	"github.com/cloudboss/keights/stackbot/whisperer"
	"github.com/mitchellh/mapstructure"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	tokenutil "k8s.io/cluster-bootstrap/token/util"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"
)

const (
	clusterPathTemplate    = "/%s/cluster/%s"
	controllerPathTemplate = "/%s/controller/%s"
	etcdCACertName         = "etcd-ca.crt"
	etcdCAKeyName          = "etcd-ca.key"
	bootstrapTokenName     = "bootstrap-token"
)

var (
	certDecodeError = errors.New("could not decode CA certificate")
	invalidKeyError = errors.New("private key is not in expected format")
)

type resourceProperties struct {
	ServiceToken   string
	ClusterName    string
	KMSKeyID       string `mapstructure:"KmsKeyId"`
	KeightsVersion string
}

func pathFormatter(template, prefix string) func(string) *string {
	return func(suffix string) *string {
		s := fmt.Sprintf(template, prefix, suffix)
		return &s
	}
}

func retrieveCA(whisp whisperer.Whisperer, certPath, keyPath *string) (*x509.Certificate, crypto.Signer, error) {
	fmt.Printf("Retrieving CA via SSM parameters %s and %s\n", *certPath, *keyPath)

	var caCert *x509.Certificate
	var caKey crypto.Signer

	certString, err := whisp.GetParameter(certPath)
	if err != nil {
		return nil, nil, err
	}
	certBlock, _ := pem.Decode([]byte(*certString))
	if certBlock == nil {
		return nil, nil, certDecodeError
	}
	caCert, err = x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	keyString, err := whisp.GetParameter(keyPath)
	if err != nil {
		return nil, nil, err
	}
	parsedKey, err := keyutil.ParsePrivateKeyPEM([]byte(*keyString))
	if err != nil {
		return nil, nil, err
	}
	caKey, ok := parsedKey.(crypto.Signer)
	if !ok {
		return nil, nil, invalidKeyError
	}
	return caCert, caKey, err
}

func genCA(whisp whisperer.Whisperer, certPath, keyPath, kmsKeyID *string) (*x509.Certificate, crypto.Signer, error) {
	fmt.Printf("Generating CA: %s and %s\n", *certPath, *keyPath)

	caCert, caKey, err := pkiutil.NewCertificateAuthority(&pkiutil.CertConfig{
		// Config defined in func KubeadmCertRootCA() from
		// k8s.io/kubernetes@v1.23.4/cmd/kubeadm/app/phases/certs/certlist.go.
		Config: certutil.Config{
			CommonName: "kubernetes",
		},
	})
	caCertPEMBytes, err := certutil.EncodeCertificates(caCert)
	if err != nil {
		return nil, nil, err
	}
	fmt.Printf("Storing %s\n", *certPath)
	caCertPEM := string(caCertPEMBytes)
	err = whisp.ForceStoreParameter(certPath, kmsKeyID, &caCertPEM)
	if err != nil {
		return nil, nil, err
	}
	caKeyPEMBytes, err := keyutil.MarshalPrivateKeyToPEM(caKey)
	if err != nil {
		return nil, nil, err
	}
	caKeyPEM := string(caKeyPEMBytes)
	fmt.Printf("Storing %s\n", *keyPath)
	err = whisp.ForceStoreParameter(keyPath, kmsKeyID, &caKeyPEM)
	if err != nil {
		return nil, nil, err
	}
	return caCert, caKey, nil
}

func genServiceAccountArtifacts(whisp whisperer.Whisperer, keyPath, pubKeyPath, kmsKeyID *string) error {
	fmt.Printf("Generating service account artifacts: %s and %s\n", *keyPath, *pubKeyPath)

	saSigningKey, err := pkiutil.NewPrivateKey(x509.RSA)
	if err != nil {
		return err
	}
	saSigningKeyPEMBytes, err := keyutil.MarshalPrivateKeyToPEM(saSigningKey)
	if err != nil {
		return err
	}
	saSigningKeyPEM := string(saSigningKeyPEMBytes)
	fmt.Printf("Storing %s\n", *keyPath)
	err = whisp.ForceStoreParameter(keyPath, kmsKeyID, &saSigningKeyPEM)
	if err != nil {
		return err
	}
	saSigningPubKeyPEMBytes, err := pkiutil.EncodePublicKeyPEM(saSigningKey.Public())
	if err != nil {
		return err
	}
	saSigningPubKeyPEM := string(saSigningPubKeyPEMBytes)
	fmt.Printf("Storing %s\n", *pubKeyPath)
	return whisp.ForceStoreParameter(pubKeyPath, kmsKeyID, &saSigningPubKeyPEM)
}

func genBootstrapToken(whisp whisperer.Whisperer, path, kmsKeyID *string) error {
	fmt.Printf("Generating bootstrap token\n")

	token, err := tokenutil.GenerateBootstrapToken()
	if err != nil {
		return err
	}
	fmt.Printf("Storing %s\n", *path)
	return whisp.StoreParameter(path, kmsKeyID, &token)
}

func genAPIClientCert(whisp whisperer.Whisperer, caCert *x509.Certificate, caKey crypto.Signer,
	certPath, keyPath, kmsKeyID *string) error {
	fmt.Printf("Generating API client certificate\n")

	config := &pkiutil.CertConfig{
		// Config defined in func KubeadmCertKubeletClient() from
		// k8s.io/kubernetes@v1.23.4/cmd/kubeadm/app/phases/certs/certlist.go.
		Config: certutil.Config{
			CommonName:   kubeadmconstants.APIServerKubeletClientCertCommonName,
			Organization: []string{kubeadmconstants.SystemPrivilegedGroup},
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
	}
	apiClientCert, apiClientKey, err := pkiutil.NewCertAndKey(caCert, caKey, config)
	if err != nil {
		return err
	}
	apiClientCertPEMBytes, err := certutil.EncodeCertificates(apiClientCert)
	if err != nil {
		return err
	}
	apiClientCertPEM := string(apiClientCertPEMBytes)
	fmt.Printf("Storing %s\n", *certPath)
	err = whisp.ForceStoreParameter(certPath, kmsKeyID, &apiClientCertPEM)
	if err != nil {
		return err
	}
	apiClientKeyPEMBytes, err := keyutil.MarshalPrivateKeyToPEM(apiClientKey)
	if err != nil {
		return err
	}
	apiClientKeyPEM := string(apiClientKeyPEMBytes)
	fmt.Printf("Storing %s\n", *keyPath)
	return whisp.ForceStoreParameter(keyPath, kmsKeyID, &apiClientKeyPEM)
}

func concatErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	errStr := ""
	for _, err := range errs {
		if errStr == "" {
			errStr = err.Error()
		} else {
			errStr += fmt.Sprintf("; %s", err.Error())
		}
	}
	return errors.New(errStr)
}

func handleCreateOrUpdate(props resourceProperties, whisp whisperer.Whisperer) error {
	var caCert *x509.Certificate
	var caKey crypto.Signer
	var err error

	clusterScopedPath := pathFormatter(clusterPathTemplate, props.ClusterName)
	controllerScopedPath := pathFormatter(controllerPathTemplate, props.ClusterName)

	bootstrapTokenPath := clusterScopedPath(bootstrapTokenName)
	caCertPath := clusterScopedPath(kubeadmconstants.CACertName)
	caKeyPath := controllerScopedPath(kubeadmconstants.CAKeyName)
	etcdCACertPath := controllerScopedPath(etcdCACertName)
	etcdCAKeyPath := controllerScopedPath(etcdCAKeyName)
	frontProxyCACertPath := controllerScopedPath(kubeadmconstants.FrontProxyCACertName)
	frontProxyCAKeyPath := controllerScopedPath(kubeadmconstants.FrontProxyCAKeyName)
	apiClientCertPath := controllerScopedPath(kubeadmconstants.APIServerKubeletClientCertName)
	apiClientKeyPath := controllerScopedPath(kubeadmconstants.APIServerKubeletClientKeyName)
	saSigningKeyPath := controllerScopedPath(kubeadmconstants.ServiceAccountPrivateKeyName)
	saSigningPubKeyPath := controllerScopedPath(kubeadmconstants.ServiceAccountPublicKeyName)

	err = helpers.IdempotentDo(
		func() (bool, error) {
			hasParameters, err := whisp.HasParameters(caCertPath, caKeyPath)
			if err != nil {
				return false, err
			}
			if hasParameters {
				// Fill caCert and caKey to be used for signing API client cert if needed.
				caCert, caKey, err = retrieveCA(whisp, caCertPath, caKeyPath)
				if err != nil {
					return false, err
				}
			}
			return hasParameters, nil
		},
		func() error {
			caCert, caKey, err = genCA(whisp, caCertPath, caKeyPath, &props.KMSKeyID)
			return err
		},
	)
	if err != nil {
		return err
	}

	err = helpers.IdempotentDo(
		func() (bool, error) {
			return whisp.HasParameters(etcdCACertPath, etcdCAKeyPath)
		},
		func() error {
			_, _, err = genCA(whisp, etcdCACertPath, etcdCAKeyPath, &props.KMSKeyID)
			return err
		},
	)
	if err != nil {
		return err
	}

	err = helpers.IdempotentDo(
		func() (bool, error) {
			return whisp.HasParameters(frontProxyCACertPath, frontProxyCAKeyPath)
		},
		func() error {
			_, _, err = genCA(whisp, frontProxyCACertPath, frontProxyCAKeyPath, &props.KMSKeyID)
			return err
		},
	)
	if err != nil {
		return err
	}

	err = helpers.IdempotentDo(
		func() (bool, error) {
			return whisp.HasParameters(apiClientCertPath, apiClientKeyPath)
		},
		func() error {
			return genAPIClientCert(whisp, caCert, caKey, apiClientCertPath, apiClientKeyPath, &props.KMSKeyID)
		},
	)
	if err != nil {
		return err
	}

	err = helpers.IdempotentDo(
		func() (bool, error) {
			return whisp.HasParameters(saSigningKeyPath, saSigningPubKeyPath)
		},
		func() error {
			return genServiceAccountArtifacts(whisp, saSigningKeyPath, saSigningPubKeyPath, &props.KMSKeyID)
		},
	)
	if err != nil {
		return err
	}

	return helpers.IdempotentDo(
		func() (bool, error) {
			return whisp.HasParameters(bootstrapTokenPath)
		},
		func() error {
			return genBootstrapToken(whisp, bootstrapTokenPath, &props.KMSKeyID)
		},
	)
}

func Handle(_ctx context.Context, event cfn.Event) (string, map[string]interface{}, error) {
	fmt.Printf("Event: %+v\n", event)

	emptyResourceID := ""
	emptyResponse := make(map[string]interface{})

	if event.RequestType != cfn.RequestCreate && event.RequestType != cfn.RequestUpdate {
		fmt.Printf("Doing nothing for %s request type\n", event.RequestType)
		return emptyResourceID, emptyResponse, nil
	}

	var props resourceProperties
	err := mapstructure.Decode(event.ResourceProperties, &props)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return emptyResourceID, emptyResponse, err
	}

	sess, err := session.NewSession()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return emptyResourceID, emptyResponse, err
	}
	whisp := whisperer.NewSSMWhisperer(sess)

	err = handleCreateOrUpdate(props, whisp)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	return emptyResourceID, emptyResponse, err
}

func main() {
	lambda.Start(cfn.LambdaWrap(Handle))
}
