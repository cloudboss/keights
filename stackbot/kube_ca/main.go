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
	"errors"
	"fmt"

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
	etcdCaCertName       = "etcd-ca.crt"
	etcdCaKeyName        = "etcd-ca.key"
	bootstrapTokenName   = "bootstrap-token"
)

type resourceProperties struct {
	ServiceToken string
	ClusterName  string
	KMSKeyID     string `mapstructure:"KmsKeyId"`
}

func newClientKeyPair(caCert *x509.Certificate, caKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey, error) {
	config := certutil.Config{
		CommonName:   kubeadmconstants.APIServerKubeletClientCertCommonName,
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Organization: []string{kubeadmconstants.MastersGroup},
	}
	return pkiutil.NewCertAndKey(caCert, caKey, config)
}

func handleCreate(props resourceProperties, whisp whisperer.Whisperer) error {
	caCert, caKey, err := pkiutil.NewCertificateAuthority()
	if err != nil {
		return err
	}

	etcdCaCert, etcdCaKey, err := pkiutil.NewCertificateAuthority()
	if err != nil {
		return err
	}

	apiClientCert, apiClientKey, err := newClientKeyPair(caCert, caKey)
	if err != nil {
		return err
	}

	caCertPEM := string(certutil.EncodeCertPEM(caCert))
	caCertPath := fmt.Sprintf(clusterSecretPath, props.ClusterName, kubeadmconstants.CACertName)
	err = whisp.StoreParameter(caCertPath, props.KMSKeyID, &caCertPEM)
	if err != nil {
		return err
	}
	caKeyPEM := string(certutil.EncodePrivateKeyPEM(caKey))
	caKeyPath := fmt.Sprintf(controllerSecretPath, props.ClusterName, kubeadmconstants.CAKeyName)
	err = whisp.StoreParameter(caKeyPath, props.KMSKeyID, &caKeyPEM)
	if err != nil {
		return err
	}

	etcdCaCertPEM := string(certutil.EncodeCertPEM(etcdCaCert))
	etcdCaCertPath := fmt.Sprintf(controllerSecretPath, props.ClusterName, etcdCaCertName)
	err = whisp.StoreParameter(etcdCaCertPath, props.KMSKeyID, &etcdCaCertPEM)
	if err != nil {
		return err
	}
	etcdCaKeyPEM := string(certutil.EncodePrivateKeyPEM(etcdCaKey))
	etcdCaKeyPath := fmt.Sprintf(controllerSecretPath, props.ClusterName, etcdCaKeyName)
	err = whisp.StoreParameter(etcdCaKeyPath, props.KMSKeyID, &etcdCaKeyPEM)
	if err != nil {
		return err
	}

	apiClientCertPEM := string(certutil.EncodeCertPEM(apiClientCert))
	apiClientCertPath := fmt.Sprintf(controllerSecretPath, props.ClusterName, kubeadmconstants.APIServerKubeletClientCertName)
	err = whisp.StoreParameter(apiClientCertPath, props.KMSKeyID, &apiClientCertPEM)
	if err != nil {
		return err
	}
	apiClientKeyPEM := string(certutil.EncodePrivateKeyPEM(apiClientKey))
	apiClientKeyPath := fmt.Sprintf(controllerSecretPath, props.ClusterName, kubeadmconstants.APIServerKubeletClientKeyName)
	err = whisp.StoreParameter(apiClientKeyPath, props.KMSKeyID, &apiClientKeyPEM)
	if err != nil {
		return err
	}

	err = newServiceAccountArtifacts(whisp, props.ClusterName, props.KMSKeyID)
	if err != nil {
		return err
	}

	return newBootstrapToken(whisp, props.ClusterName, props.KMSKeyID)
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

func handleDelete(props resourceProperties, whisp whisperer.Whisperer) error {
	clusterName := props.ClusterName
	parameters := []string{
		fmt.Sprintf(clusterSecretPath, clusterName, kubeadmconstants.CACertName),
		fmt.Sprintf(clusterSecretPath, clusterName, bootstrapTokenName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.CAKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, etcdCaCertName),
		fmt.Sprintf(controllerSecretPath, clusterName, etcdCaKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerKubeletClientCertName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerKubeletClientKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.ServiceAccountPrivateKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.ServiceAccountPublicKeyName),
	}
	errs := []error{}
	for _, parameter := range parameters {
		if err := whisp.DeleteParameter(&parameter); err != nil {
			errs = append(errs, err)
		}
	}
	return concatErrors(errs)
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
