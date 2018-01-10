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
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/cloudboss/stackhand/response"
	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
	"github.com/eawsy/aws-lambda-go-event/service/lambda/runtime/event/cloudformationevt"
	certutil "k8s.io/client-go/util/cert"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/certs"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/certs/pkiutil"
)

const (
	clusterSecretPath    = "/%s/cluster/%s"
	controllerSecretPath = "/%s/controller/%s"
	etcd                 = "etcd"
	etcdCertName         = "etcd.crt"
	etcdKeyName          = "etcd.key"
	etcdPeer             = "etcd-peer"
	etcdPeerCertName     = "etcd-peer.crt"
	etcdPeerKeyName      = "etcd-peer.key"
)

type resourceProperties struct {
	ServiceToken     string
	ClusterName      string
	LoadBalancerName string
	NumInstances     int    `json:"NumInstances,string"`
	KMSKeyID         string `json:"KmsKeyId"`
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

func newCerts(ssmClient ssmiface.SSMAPI, clusterName, kmsKeyID, loadBalancerName string, controllerNames []string) error {
	caCert, caKey, err := pkiutil.NewCertificateAuthority()
	if err != nil {
		return err
	}

	apiDNSNames := append(controllerNames, loadBalancerName)
	apiCert, apiKey, err := newServerKeyPair(caCert, caKey, kubeadmconstants.APIServerCertCommonName, apiDNSNames)
	if err != nil {
		return err
	}

	etcdCert, etcdKey, err := newServerKeyPair(caCert, caKey, etcd, controllerNames)
	if err != nil {
		return err
	}

	etcdPeerCert, etcdPeerKey, err := newServerKeyPair(caCert, caKey, etcdPeer, controllerNames)
	if err != nil {
		return err
	}

	apiClientCert, apiClientKey, err := newClientKeyPair(caCert, caKey)
	if err != nil {
		return err
	}

	caCertPEM := string(certutil.EncodeCertPEM(caCert))
	caCertPath := fmt.Sprintf(clusterSecretPath, clusterName, kubeadmconstants.CACertName)
	err = storeParameter(ssmClient, caCertPath, kmsKeyID, &caCertPEM)
	if err != nil {
		return err
	}
	caKeyPEM := string(certutil.EncodePrivateKeyPEM(caKey))
	caKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.CAKeyName)
	err = storeParameter(ssmClient, caKeyPath, kmsKeyID, &caKeyPEM)
	if err != nil {
		return err
	}

	apiCertPEM := string(certutil.EncodeCertPEM(apiCert))
	apiCertPath := fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerCertName)
	err = storeParameter(ssmClient, apiCertPath, kmsKeyID, &apiCertPEM)
	if err != nil {
		return err
	}
	apiKeyPEM := string(certutil.EncodePrivateKeyPEM(apiKey))
	apiKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerKeyName)
	err = storeParameter(ssmClient, apiKeyPath, kmsKeyID, &apiKeyPEM)
	if err != nil {
		return err
	}

	etcdCertPEM := string(certutil.EncodeCertPEM(etcdCert))
	etcdCertPath := fmt.Sprintf(controllerSecretPath, clusterName, etcdCertName)
	err = storeParameter(ssmClient, etcdCertPath, kmsKeyID, &etcdCertPEM)
	if err != nil {
		return err
	}
	etcdKeyPEM := string(certutil.EncodePrivateKeyPEM(etcdKey))
	etcdKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, etcdKeyName)
	err = storeParameter(ssmClient, etcdKeyPath, kmsKeyID, &etcdKeyPEM)
	if err != nil {
		return err
	}

	etcdPeerCertPEM := string(certutil.EncodeCertPEM(etcdPeerCert))
	etcdPeerCertPath := fmt.Sprintf(controllerSecretPath, clusterName, etcdPeerCertName)
	err = storeParameter(ssmClient, etcdPeerCertPath, kmsKeyID, &etcdPeerCertPEM)
	if err != nil {
		return err
	}
	etcdPeerKeyPEM := string(certutil.EncodePrivateKeyPEM(etcdPeerKey))
	etcdPeerKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, etcdPeerKeyName)
	err = storeParameter(ssmClient, etcdPeerKeyPath, kmsKeyID, &etcdPeerKeyPEM)
	if err != nil {
		return err
	}

	apiClientCertPEM := string(certutil.EncodeCertPEM(apiClientCert))
	apiClientCertPath := fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerKubeletClientCertName)
	err = storeParameter(ssmClient, apiClientCertPath, kmsKeyID, &apiClientCertPEM)
	if err != nil {
		return err
	}
	apiClientKeyPEM := string(certutil.EncodePrivateKeyPEM(apiClientKey))
	apiClientKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerKubeletClientKeyName)
	err = storeParameter(ssmClient, apiClientKeyPath, kmsKeyID, &apiClientKeyPEM)
	if err != nil {
		return err
	}

	err = newServiceAccountArtifacts(ssmClient, clusterName, kmsKeyID)
	if err != nil {
		return err
	}

	return nil
}

func newServiceAccountArtifacts(ssmClient ssmiface.SSMAPI, clusterName, kmsKeyID string) error {
	saSigningKey, err := certs.NewServiceAccountSigningKey()
	if err != nil {
		return err
	}

	saSigningKeyPEM := string(certutil.EncodePrivateKeyPEM(saSigningKey))
	saSigningKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.ServiceAccountPrivateKeyName)
	err = storeParameter(ssmClient, saSigningKeyPath, kmsKeyID, &saSigningKeyPEM)
	if err != nil {
		return err
	}

	saSigningPubKeyPEM, err := certutil.EncodePublicKeyPEM(&saSigningKey.PublicKey)
	if err != nil {
		return err
	}
	saSigningPubKeyPEMStr := string(saSigningPubKeyPEM)
	saSigningPubKeyPath := fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.ServiceAccountPublicKeyName)
	return storeParameter(ssmClient, saSigningPubKeyPath, kmsKeyID, &saSigningPubKeyPEMStr)
}

func storeParameter(ssmClient ssmiface.SSMAPI, path, kmsKeyID string, content *string) error {
	input := &ssm.PutParameterInput{
		Name:  aws.String(path),
		Type:  aws.String("SecureString"),
		Value: content,
	}
	if kmsKeyID != "" {
		input.KeyId = aws.String(kmsKeyID)
	}
	_, err := ssmClient.PutParameter(input)
	return err
}

func deleteParameter(ssmClient ssmiface.SSMAPI, path *string) error {
	_, err := ssmClient.DeleteParameter(&ssm.DeleteParameterInput{
		Name: path,
	})
	return err
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

func handleCreate(props resourceProperties, ssmClient ssmiface.SSMAPI, responder response.Responder) error {
	names := controllerNames(props.ClusterName, props.NumInstances)
	err := newCerts(ssmClient, props.ClusterName, props.KMSKeyID, props.LoadBalancerName, names)
	if err != nil {
		return responder.FireFailed(err.Error())
	}
	return responder.FireSuccess()
}

func handleDelete(props resourceProperties, ssmClient ssmiface.SSMAPI, responder response.Responder) error {
	clusterName := props.ClusterName
	parameters := []string{
		fmt.Sprintf(clusterSecretPath, clusterName, kubeadmconstants.CACertName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.CAKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerCertName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, etcdCertName),
		fmt.Sprintf(controllerSecretPath, clusterName, etcdKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, etcdPeerCertName),
		fmt.Sprintf(controllerSecretPath, clusterName, etcdPeerKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerKubeletClientCertName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.APIServerKubeletClientKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.ServiceAccountPrivateKeyName),
		fmt.Sprintf(controllerSecretPath, clusterName, kubeadmconstants.ServiceAccountPublicKeyName),
	}
	for _, parameter := range parameters {
		if err := deleteParameter(ssmClient, &parameter); err != nil {
			return responder.FireFailed(err.Error())
		}
	}
	return responder.FireSuccess()
}

func Handle(event *cloudformationevt.Event, ctx *runtime.Context) (interface{}, error) {
	responder := response.NewResponder(event, ctx)

	if event.RequestType == "Update" {
		return nil, responder.FireSuccess()
	}

	var props resourceProperties
	err := json.Unmarshal(event.ResourceProperties, &props)
	if err != nil {
		return nil, responder.FireFailed(err.Error())
	}

	sess, err := session.NewSession()
	if err != nil {
		return nil, responder.FireFailed(err.Error())
	}
	ssmClient := ssm.New(sess)

	if event.RequestType == "Create" {
		return nil, handleCreate(props, ssmClient, responder)
	}

	if event.RequestType == "Delete" {
		return nil, handleDelete(props, ssmClient, responder)
	}

	return nil, nil
}
