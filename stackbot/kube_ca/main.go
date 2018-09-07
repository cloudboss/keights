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
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"strconv"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cloudboss/keights/stackbot/whisperer"
	"github.com/mitchellh/mapstructure"
	certutil "k8s.io/client-go/util/cert"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/certs"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/certs/pkiutil"
	tokenutil "k8s.io/kubernetes/cmd/kubeadm/app/util/token"
)

const (
	clusterSecretPath    = "/%s/cluster/%s"
	controllerSecretPath = "/%s/controller/%s"
	etcd                 = "etcd"
	etcdCaCertName       = "etcd-ca.crt"
	etcdCaKeyName        = "etcd-ca.key"
	etcdCertName         = "etcd.crt"
	etcdKeyName          = "etcd.key"
	etcdPeer             = "etcd-peer"
	etcdPeerCertName     = "etcd-peer.crt"
	etcdPeerKeyName      = "etcd-peer.key"
	etcdClientCertName   = "etcd-client.crt"
	etcdClientKeyName    = "etcd-client.key"
	bootstrapTokenName   = "bootstrap-token"
)

type resourceProperties struct {
	ServiceToken     string
	ClusterName      string
	LoadBalancerName string
	NumInstances     string
	KMSKeyID         string `mapstructure:"KmsKeyId"`
}

func newServerKeyPair(caCert *x509.Certificate, caKey *rsa.PrivateKey, cn string, dnsNames []string) (*x509.Certificate, *rsa.PrivateKey, error) {
	config := certutil.Config{
		CommonName: cn,
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		AltNames:   certutil.AltNames{DNSNames: dnsNames},
	}
	return pkiutil.NewCertAndKey(caCert, caKey, config)
}

func newClientKeyPair(caCert *x509.Certificate, caKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey, error) {
	config := certutil.Config{
		CommonName:   kubeadmconstants.APIServerKubeletClientCertCommonName,
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Organization: []string{kubeadmconstants.MastersGroup},
	}
	return pkiutil.NewCertAndKey(caCert, caKey, config)
}

func newClientServerKeyPair(caCert *x509.Certificate, caKey *rsa.PrivateKey, cn string, dnsNames []string) (*x509.Certificate, *rsa.PrivateKey, error) {
	config := certutil.Config{
		CommonName: cn,
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		AltNames:   certutil.AltNames{DNSNames: dnsNames},
	}
	return pkiutil.NewCertAndKey(caCert, caKey, config)
}

func newCerts(whisp whisperer.Whisperer, clusterName, kmsKeyID, loadBalancerName string, controllerNames []string) error {
	caCert, caKey, err := pkiutil.NewCertificateAuthority()
	if err != nil {
		return err
	}

	etcdCaCert, etcdCaKey, err := pkiutil.NewCertificateAuthority()
	if err != nil {
		return err
	}

	apiDNSNames := append(controllerNames, loadBalancerName)
	apiCert, apiKey, err := newServerKeyPair(caCert, caKey, kubeadmconstants.APIServerCertCommonName, apiDNSNames)
	if err != nil {
		return err
	}

	etcdCert, etcdKey, err := newServerKeyPair(etcdCaCert, etcdCaKey, etcd, controllerNames)
	if err != nil {
		return err
	}

	etcdPeerCert, etcdPeerKey, err := newClientServerKeyPair(etcdCaCert, etcdCaKey, etcdPeer, controllerNames)
	if err != nil {
		return err
	}

	apiClientCert, apiClientKey, err := newClientKeyPair(caCert, caKey)
	if err != nil {
		return err
	}

	etcdClientCert, etcdClientKey, err := newClientKeyPair(etcdCaCert, etcdCaKey)
	if err != nil {
		return err
	}

	caCertPEM := string(certutil.EncodeCertPEM(caCert))
	caCertPath := fmt.Sprintf(clusterSecretPath, clusterName, kubeadmconstants.CACertName)
	err = whisp.StoreParameter(caCertPath, kmsKeyID, &caCertPEM)
	if err != nil {
		return err
	}
	caKeyPEM := string(certutil.EncodePrivateKeyPEM(caKey))
	caKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.CAKeyName)
	err = whisp.StoreParameter(caKeyPath, kmsKeyID, &caKeyPEM)
	if err != nil {
		return err
	}

	etcdCaCertPEM := string(certutil.EncodeCertPEM(etcdCaCert))
	etcdCaCertPath := fmt.Sprintf(controllerSecretPath, clusterName, etcdCaCertName)
	err = whisp.StoreParameter(etcdCaCertPath, kmsKeyID, &etcdCaCertPEM)
	if err != nil {
		return err
	}
	etcdCaKeyPEM := string(certutil.EncodePrivateKeyPEM(etcdCaKey))
	etcdCaKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, etcdCaKeyName)
	err = whisp.StoreParameter(etcdCaKeyPath, kmsKeyID, &etcdCaKeyPEM)
	if err != nil {
		return err
	}

	apiCertPEM := string(certutil.EncodeCertPEM(apiCert))
	apiCertPath := fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerCertName)
	err = whisp.StoreParameter(apiCertPath, kmsKeyID, &apiCertPEM)
	if err != nil {
		return err
	}
	apiKeyPEM := string(certutil.EncodePrivateKeyPEM(apiKey))
	apiKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerKeyName)
	err = whisp.StoreParameter(apiKeyPath, kmsKeyID, &apiKeyPEM)
	if err != nil {
		return err
	}

	etcdCertPEM := string(certutil.EncodeCertPEM(etcdCert))
	etcdCertPath := fmt.Sprintf(controllerSecretPath, clusterName, etcdCertName)
	err = whisp.StoreParameter(etcdCertPath, kmsKeyID, &etcdCertPEM)
	if err != nil {
		return err
	}
	etcdKeyPEM := string(certutil.EncodePrivateKeyPEM(etcdKey))
	etcdKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, etcdKeyName)
	err = whisp.StoreParameter(etcdKeyPath, kmsKeyID, &etcdKeyPEM)
	if err != nil {
		return err
	}

	etcdPeerCertPEM := string(certutil.EncodeCertPEM(etcdPeerCert))
	etcdPeerCertPath := fmt.Sprintf(controllerSecretPath, clusterName, etcdPeerCertName)
	err = whisp.StoreParameter(etcdPeerCertPath, kmsKeyID, &etcdPeerCertPEM)
	if err != nil {
		return err
	}
	etcdPeerKeyPEM := string(certutil.EncodePrivateKeyPEM(etcdPeerKey))
	etcdPeerKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, etcdPeerKeyName)
	err = whisp.StoreParameter(etcdPeerKeyPath, kmsKeyID, &etcdPeerKeyPEM)
	if err != nil {
		return err
	}

	apiClientCertPEM := string(certutil.EncodeCertPEM(apiClientCert))
	apiClientCertPath := fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerKubeletClientCertName)
	err = whisp.StoreParameter(apiClientCertPath, kmsKeyID, &apiClientCertPEM)
	if err != nil {
		return err
	}
	apiClientKeyPEM := string(certutil.EncodePrivateKeyPEM(apiClientKey))
	apiClientKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerKubeletClientKeyName)
	err = whisp.StoreParameter(apiClientKeyPath, kmsKeyID, &apiClientKeyPEM)
	if err != nil {
		return err
	}

	etcdClientCertPEM := string(certutil.EncodeCertPEM(etcdClientCert))
	etcdClientCertPath := fmt.Sprintf(controllerSecretPath, clusterName, etcdClientCertName)
	err = whisp.StoreParameter(etcdClientCertPath, kmsKeyID, &etcdClientCertPEM)
	if err != nil {
		return err
	}
	etcdClientKeyPEM := string(certutil.EncodePrivateKeyPEM(etcdClientKey))
	etcdClientKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, etcdClientKeyName)
	err = whisp.StoreParameter(etcdClientKeyPath, kmsKeyID, &etcdClientKeyPEM)
	if err != nil {
		return err
	}

	err = newServiceAccountArtifacts(whisp, clusterName, kmsKeyID)
	if err != nil {
		return err
	}

	return newBootstrapToken(whisp, clusterName, kmsKeyID)
}

func newServiceAccountArtifacts(whisp whisperer.Whisperer, clusterName, kmsKeyID string) error {
	saSigningKey, err := certs.NewServiceAccountSigningKey()
	if err != nil {
		return err
	}

	saSigningKeyPEM := string(certutil.EncodePrivateKeyPEM(saSigningKey))
	saSigningKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.ServiceAccountPrivateKeyName)
	err = whisp.StoreParameter(saSigningKeyPath, kmsKeyID, &saSigningKeyPEM)
	if err != nil {
		return err
	}

	saSigningPubKeyPEM, err := certutil.EncodePublicKeyPEM(&saSigningKey.PublicKey)
	if err != nil {
		return err
	}
	saSigningPubKeyPEMStr := string(saSigningPubKeyPEM)
	saSigningPubKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.ServiceAccountPublicKeyName)
	return whisp.StoreParameter(saSigningPubKeyPath, kmsKeyID, &saSigningPubKeyPEMStr)
}

func newBootstrapToken(whisp whisperer.Whisperer, clusterName, kmsKeyID string) error {
	token, err := tokenutil.GenerateToken()
	if err != nil {
		return err
	}
	path := fmt.Sprintf(clusterSecretPath, clusterName, bootstrapTokenName)
	return whisp.StoreParameter(path, kmsKeyID, &token)
}

func controllerNames(clusterName string, numInstances int) []string {
	names := []string{
		fmt.Sprintf("api-%s.k8s.local", clusterName),
	}
	for i := 1; i <= numInstances; i++ {
		name := fmt.Sprintf("%s-%d.k8s.local", clusterName, i)
		names = append(names, name)
	}
	return names
}

func handleCreate(props resourceProperties, whisp whisperer.Whisperer) error {
	numInstances, err := strconv.Atoi(props.NumInstances)
	if err != nil {
		return err
	}
	names := controllerNames(props.ClusterName, numInstances)
	return newCerts(whisp, props.ClusterName, props.KMSKeyID, props.LoadBalancerName, names)
}

func handleDelete(props resourceProperties, whisp whisperer.Whisperer) error {
	clusterName := props.ClusterName
	parameters := []string{
		fmt.Sprintf(clusterSecretPath, clusterName, kubeadmconstants.CACertName),
		fmt.Sprintf(clusterSecretPath, clusterName, bootstrapTokenName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.CAKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerCertName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, etcdCaCertName),
		fmt.Sprintf(controllerSecretPath, clusterName, etcdCaKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, etcdCertName),
		fmt.Sprintf(controllerSecretPath, clusterName, etcdKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, etcdPeerCertName),
		fmt.Sprintf(controllerSecretPath, clusterName, etcdPeerKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, etcdClientCertName),
		fmt.Sprintf(controllerSecretPath, clusterName, etcdClientKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerKubeletClientCertName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerKubeletClientKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.ServiceAccountPrivateKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.ServiceAccountPublicKeyName),
	}
	for _, parameter := range parameters {
		if err := whisp.DeleteParameter(&parameter); err != nil {
			return err
		}
	}
	return nil
}

func Handle(_ctx context.Context, event cfn.Event) (string, map[string]interface{}, error) {
	physicalResourceID := ""
	data := make(map[string]interface{})

	if event.RequestType == "Update" {
		return physicalResourceID, data, nil
	}

	var props resourceProperties
	err := mapstructure.Decode(event.ResourceProperties, &props)
	if err != nil {
		return physicalResourceID, data, err
	}

	sess, err := session.NewSession()
	if err != nil {
		return physicalResourceID, data, err
	}
	whisp := whisperer.NewSSMWhisperer(sess)

	if event.RequestType == "Create" {
		err = handleCreate(props, whisp)
		if err != nil {
			return physicalResourceID, data, err
		}
	}

	if event.RequestType == "Delete" {
		err = handleDelete(props, whisp)
		if err != nil {
			return physicalResourceID, data, err
		}
	}
	return physicalResourceID, data, nil
}

func main() {
	lambda.Start(cfn.LambdaWrap(Handle))
}
