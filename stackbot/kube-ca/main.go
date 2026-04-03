// Copyright 2026 Joseph Wright <joseph@cloudboss.co>
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
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/cloudboss/keights/internal/environment"
	"github.com/cloudboss/keights/internal/whisperer"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	tokenutil "k8s.io/cluster-bootstrap/token/util"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"
)

const (
	clusterPathTemplate    = "/keights/%s/cluster/%s"
	controllerPathTemplate = "/keights/%s/controller/%s"
	etcdCACertName         = "etcd-ca.crt"
	etcdCAKeyName          = "etcd-ca.key"
	bootstrapTokenName     = "bootstrap-token"
)

var (
	certDecodeError = errors.New("could not decode CA certificate")
	invalidKeyError = errors.New("private key is not in expected format")

	requiredEnvironment = []string{
		"CLUSTER_NAME",
		"ENCRYPTION_ALGORITHM",
		"KMS_KEY_ID",
	}
)

type Properties struct {
	ClusterName         string
	EncryptionAlgorithm kubeadmapi.EncryptionAlgorithmType
	KMSKeyID            string
}

func pathFormatter(template, prefix string) func(string) string {
	return func(suffix string) string {
		return fmt.Sprintf(template, prefix, suffix)
	}
}

func retrieveCA(
	ctx context.Context,
	whisp whisperer.Whisperer,
	certPath, keyPath string,
) (*x509.Certificate, crypto.Signer, error) {
	log.Printf("Retrieving CA via SSM parameters %s and %s\n", certPath, keyPath)

	var caCert *x509.Certificate
	var caKey crypto.Signer

	certString, err := whisp.GetParameter(ctx, certPath)
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
	keyString, err := whisp.GetParameter(ctx, keyPath)
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

func genCA(
	ctx context.Context,
	whisp whisperer.Whisperer,
	certPath, keyPath, kmsKeyID, commonName string,
	encryptionAlgorithm kubeadmapi.EncryptionAlgorithmType,
) (*x509.Certificate, crypto.Signer, error) {
	log.Printf("Generating CA: %s and %s\n", certPath, keyPath)

	caCert, caKey, err := pkiutil.NewCertificateAuthority(&pkiutil.CertConfig{
		EncryptionAlgorithm: encryptionAlgorithm,
		Config:              certutil.Config{CommonName: commonName},
	})
	caCertPEMBytes, err := certutil.EncodeCertificates(caCert)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("Storing %s\n", certPath)
	caCertPEM := string(caCertPEMBytes)
	err = whisp.ForceStoreParameter(ctx, certPath, kmsKeyID, caCertPEM)
	if err != nil {
		return nil, nil, err
	}
	caKeyPEMBytes, err := keyutil.MarshalPrivateKeyToPEM(caKey)
	if err != nil {
		return nil, nil, err
	}
	caKeyPEM := string(caKeyPEMBytes)
	log.Printf("Storing %s\n", keyPath)
	err = whisp.ForceStoreParameter(ctx, keyPath, kmsKeyID, caKeyPEM)
	if err != nil {
		return nil, nil, err
	}
	return caCert, caKey, nil
}

func genServiceAccountArtifacts(
	ctx context.Context,
	whisp whisperer.Whisperer,
	keyPath, pubKeyPath, kmsKeyID string,
	encryptionAlgorithm kubeadmapi.EncryptionAlgorithmType,
) error {
	log.Printf("Generating service account artifacts: %s and %s\n", keyPath, pubKeyPath)

	saSigningKey, err := pkiutil.NewPrivateKey(encryptionAlgorithm)
	if err != nil {
		return err
	}
	saSigningKeyPEMBytes, err := keyutil.MarshalPrivateKeyToPEM(saSigningKey)
	if err != nil {
		return err
	}
	saSigningKeyPEM := string(saSigningKeyPEMBytes)
	log.Printf("Storing %s\n", keyPath)
	err = whisp.ForceStoreParameter(ctx, keyPath, kmsKeyID, saSigningKeyPEM)
	if err != nil {
		return err
	}
	saSigningPubKeyPEMBytes, err := pkiutil.EncodePublicKeyPEM(saSigningKey.Public())
	if err != nil {
		return err
	}
	saSigningPubKeyPEM := string(saSigningPubKeyPEMBytes)
	log.Printf("Storing %s\n", pubKeyPath)
	return whisp.ForceStoreParameter(ctx, pubKeyPath, kmsKeyID, saSigningPubKeyPEM)
}

func genBootstrapToken(
	ctx context.Context,
	whisp whisperer.Whisperer,
	path, kmsKeyID string,
) error {
	log.Printf("Generating bootstrap token\n")

	token, err := tokenutil.GenerateBootstrapToken()
	if err != nil {
		return err
	}
	log.Printf("Storing %s\n", path)
	return whisp.StoreParameter(ctx, path, kmsKeyID, token)
}

func genAPIServerKubeletClientCert(
	ctx context.Context,
	whisp whisperer.Whisperer,
	caCert *x509.Certificate,
	caKey crypto.Signer,
	certPath, keyPath, kmsKeyID string,
	encryptionAlgorithm kubeadmapi.EncryptionAlgorithmType,
) error {
	log.Printf("Generating API client certificate\n")

	config := &pkiutil.CertConfig{
		EncryptionAlgorithm: encryptionAlgorithm,
		// Config defined in func KubeadmCertKubeletClient() from
		// k8s.io/kubernetes/cmd/kubeadm/app/phases/certs/certlist.go.
		Config: certutil.Config{
			CommonName: kubeadmconstants.APIServerKubeletClientCertCommonName,
			Organization: []string{
				kubeadmconstants.ClusterAdminsGroupAndClusterRoleBinding,
			},
			Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
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
	log.Printf("Storing %s\n", certPath)
	err = whisp.ForceStoreParameter(ctx, certPath, kmsKeyID, apiClientCertPEM)
	if err != nil {
		return err
	}
	apiClientKeyPEMBytes, err := keyutil.MarshalPrivateKeyToPEM(apiClientKey)
	if err != nil {
		return err
	}
	apiClientKeyPEM := string(apiClientKeyPEMBytes)
	log.Printf("Storing %s\n", keyPath)
	return whisp.ForceStoreParameter(ctx, keyPath, kmsKeyID, apiClientKeyPEM)
}

func createOrUpdateCerts(ctx context.Context, props Properties, whisp whisperer.Whisperer) error {
	var (
		caCert *x509.Certificate
		caKey  crypto.Signer
		err    error
	)

	clusterScopedPath := pathFormatter(clusterPathTemplate, props.ClusterName)
	controllerScopedPath := pathFormatter(controllerPathTemplate, props.ClusterName)

	bootstrapTokenPath := clusterScopedPath(bootstrapTokenName)
	caCertPath := clusterScopedPath(kubeadmconstants.CACertName)
	caKeyPath := controllerScopedPath(kubeadmconstants.CAKeyName)
	etcdCACertPath := controllerScopedPath(etcdCACertName)
	etcdCAKeyPath := controllerScopedPath(etcdCAKeyName)
	frontProxyCACertPath := controllerScopedPath(kubeadmconstants.FrontProxyCACertName)
	frontProxyCAKeyPath := controllerScopedPath(kubeadmconstants.FrontProxyCAKeyName)
	apiServerKubeletClientCertPath := controllerScopedPath(
		kubeadmconstants.APIServerKubeletClientCertName)
	apiServerKubeletClientKeyPath := controllerScopedPath(
		kubeadmconstants.APIServerKubeletClientKeyName)
	saSigningKeyPath := controllerScopedPath(kubeadmconstants.ServiceAccountPrivateKeyName)
	saSigningPubKeyPath := controllerScopedPath(kubeadmconstants.ServiceAccountPublicKeyName)

	// Root CA.
	err = doUnless(
		func() (bool, error) {
			hasParameters, err := whisp.HasParameters(ctx, caCertPath, caKeyPath)
			if err != nil {
				return false, err
			}
			if hasParameters {
				// Fill caCert and caKey for signing API client cert if needed.
				caCert, caKey, err = retrieveCA(ctx, whisp, caCertPath, caKeyPath)
				if err != nil {
					return false, err
				}
			}
			return hasParameters, nil
		},
		func() error {
			caCert, caKey, err = genCA(ctx, whisp, caCertPath, caKeyPath,
				props.KMSKeyID, "kubernetes", props.EncryptionAlgorithm)
			return err
		},
	)
	if err != nil {
		return err
	}

	// Etcd CA.
	err = doUnless(
		func() (bool, error) {
			return whisp.HasParameters(ctx, etcdCACertPath, etcdCAKeyPath)
		},
		func() error {
			_, _, err = genCA(ctx, whisp, etcdCACertPath, etcdCAKeyPath,
				props.KMSKeyID, "etcd-ca", props.EncryptionAlgorithm)
			return err
		},
	)
	if err != nil {
		return err
	}

	// Front Proxy CA.
	err = doUnless(
		func() (bool, error) {
			return whisp.HasParameters(ctx, frontProxyCACertPath, frontProxyCAKeyPath)
		},
		func() error {
			_, _, err = genCA(ctx, whisp, frontProxyCACertPath, frontProxyCAKeyPath,
				props.KMSKeyID, "front-proxy-ca", props.EncryptionAlgorithm)
			return err
		},
	)
	if err != nil {
		return err
	}

	// API Server Kubelet Client Cert.
	err = doUnless(
		func() (bool, error) {
			return whisp.HasParameters(ctx, apiServerKubeletClientCertPath,
				apiServerKubeletClientKeyPath)
		},
		func() error {
			return genAPIServerKubeletClientCert(ctx, whisp, caCert, caKey,
				apiServerKubeletClientCertPath, apiServerKubeletClientKeyPath,
				props.KMSKeyID, props.EncryptionAlgorithm)
		},
	)
	if err != nil {
		return err
	}

	// Service Account signing keys.
	err = doUnless(
		func() (bool, error) {
			return whisp.HasParameters(ctx, saSigningKeyPath, saSigningPubKeyPath)
		},
		func() error {
			return genServiceAccountArtifacts(ctx, whisp, saSigningKeyPath,
				saSigningPubKeyPath, props.KMSKeyID, props.EncryptionAlgorithm)
		},
	)
	if err != nil {
		return err
	}

	// Bootstrap token.
	return doUnless(
		func() (bool, error) {
			return whisp.HasParameters(ctx, bootstrapTokenPath)
		},
		func() error {
			return genBootstrapToken(ctx, whisp, bootstrapTokenPath, props.KMSKeyID)
		},
	)
}

func handleRequest(ctx context.Context) error {
	env, err := environment.EnsureEnvironment(requiredEnvironment)
	if err != nil {
		return err
	}
	log.Printf("Environment: %+v\n", env)

	clusterName := env["CLUSTER_NAME"]
	encryptionAlgorithm := kubeadmapi.EncryptionAlgorithmType(env["ENCRYPTION_ALGORITHM"])
	kmsKeyID := env["KMS_KEY_ID"]

	props := Properties{
		ClusterName:         clusterName,
		EncryptionAlgorithm: encryptionAlgorithm,
		KMSKeyID:            kmsKeyID,
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("unable to load default AWS config: %w", err)
	}

	whisp := whisperer.NewSSMWhisperer(cfg)

	return createOrUpdateCerts(ctx, props, whisp)
}

func doUnless(isDone func() (bool, error), do func() error) error {
	done, err := isDone()
	if err != nil {
		return err
	}
	if !done {
		return do()
	}
	return nil
}

func main() {
	lambda.Start(handleRequest)
}
