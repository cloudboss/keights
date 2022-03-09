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
	"errors"
	"testing"

	"github.com/cloudboss/keights/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

var (
	clusterName = "cb"

	clusterScopedPath    = pathFormatter(clusterPathTemplate, clusterName)
	controllerScopedPath = pathFormatter(controllerPathTemplate, clusterName)

	bootstrapTokenPath   = clusterScopedPath(bootstrapTokenName)
	caCertPath           = clusterScopedPath(kubeadmconstants.CACertName)
	caKeyPath            = controllerScopedPath(kubeadmconstants.CAKeyName)
	etcdCACertPath       = controllerScopedPath(etcdCACertName)
	etcdCAKeyPath        = controllerScopedPath(etcdCAKeyName)
	frontProxyCACertPath = controllerScopedPath(kubeadmconstants.FrontProxyCACertName)
	frontProxyCAKeyPath  = controllerScopedPath(kubeadmconstants.FrontProxyCAKeyName)
	apiClientCertPath    = controllerScopedPath(kubeadmconstants.APIServerKubeletClientCertName)
	apiClientKeyPath     = controllerScopedPath(kubeadmconstants.APIServerKubeletClientKeyName)
	saSigningKeyPath     = controllerScopedPath(kubeadmconstants.ServiceAccountPrivateKeyName)
	saSigningPubKeyPath  = controllerScopedPath(kubeadmconstants.ServiceAccountPublicKeyName)

	caCertMaterial = `-----BEGIN CERTIFICATE-----
MIICyDCCAbCgAwIBAgIBADANBgkqhkiG9w0BAQsFADAVMRMwEQYDVQQDEwprdWJl
cm5ldGVzMB4XDTE5MDQwMzAyMDAyM1oXDTI5MDMzMTAyMDAyM1owFTETMBEGA1UE
AxMKa3ViZXJuZXRlczCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAKTk
xM9kd8phwpYY0ic96l6w9vqdUso0bARF07hsbx2dlF5WKm9nkaTtnyGZfEMwi2nc
oTMvFfrLp8KjXp3TlS84bKkMqzDF7s9JCVGKOe+HSIz154IyuAPFUKOu67gQ1D1q
F2Kl/3z3qElbRsVDrfAuQP1YkAR4P8/BiUFexyVH+NL0Kx3JtkvqCFlJVn3I2d8L
Jh6u7ANPerHtwS1uHOdIuEu9B8ak7Wt6bY3e9ccqzOPIw9CLhYiIbv6YQWo3rIG0
kI+CLnTVvl9gjiQmlSRCg9QgSt5STm9hRT/fiLGl6oNW384JdsWVlydEKQO7N90m
Iv840ZhIP8kAUag75s0CAwEAAaMjMCEwDgYDVR0PAQH/BAQDAgKkMA8GA1UdEwEB
/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAADfelyhElG0Pxk47+FfK1JZcYqM
cs5azh5vPXdU68fPFG615mblYoK2Ulm7b8eqyn16MVaEiXFVfef891N6c7zkqXre
2Po+SL9VHxRwoEjQHpI8StL3Es+L6fD7iVH0aeAi2KgqkZonMW19cv1Hnd7sHlbq
lx2XMLsQnCHLnAVO/WV4E4UMTidb+jjVDVkvIsvaF+yQYzQWBobwjvKdcM2vV+K0
x9fc/KqBmq4+V8MAunQqZGzbdnfEmwcd8+Hrz6dNoTvwU40bIhcnvjGl3JLhix6A
I0+U7V8GWLinFP1dl0Idzg9VHAyMjk1qSbAHua3OiONsF0Q+JMC06vlvCgk=
-----END CERTIFICATE-----`

	caKeyMaterial = `-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEApOTEz2R3ymHClhjSJz3qXrD2+p1SyjRsBEXTuGxvHZ2UXlYq
b2eRpO2fIZl8QzCLadyhMy8V+sunwqNendOVLzhsqQyrMMXuz0kJUYo574dIjPXn
gjK4A8VQo67ruBDUPWoXYqX/fPeoSVtGxUOt8C5A/ViQBHg/z8GJQV7HJUf40vQr
Hcm2S+oIWUlWfcjZ3wsmHq7sA096se3BLW4c50i4S70HxqTta3ptjd71xyrM48jD
0IuFiIhu/phBajesgbSQj4IudNW+X2COJCaVJEKD1CBK3lJOb2FFP9+IsaXqg1bf
zgl2xZWXJ0QpA7s33SYi/zjRmEg/yQBRqDvmzQIDAQABAoIBADKnyMJBhf9ZOvLr
WxwNDEPcr3LcA8P0iL5jSSBdx2DcuOimJdElivuUuA8VXLQzZJC345mavHDYQYgs
sfNgPXNNLSxdpPWNyMhLEp7HDPdFowcSv/UiaZ9W7WfrY6SfHuRjBB4dCri0SDGI
5dvR58xiGTr7Cvskic3kEatQV3NfA1r4SFnhNHQ7HBlJ1lC7bMmjiu+RGTwfEddn
dVM/QT8hER7+xEHgZoHkN9F51gsU1mF+7klnpSlW+QzRRcvKlTMOIvmndCht9OKQ
euKMmFpwkcCZ+JPWCfJEbqRBMWPHzWZbas68J8VstLEpOlqUQG/ix3Bn4JdAOjEz
Ix7XRq0CgYEA2oZqqq3vbNlIHMiuoDGur31tTaqG7sNttRNy7pubfKjxVQk36mij
GcIh+4HMaMT0qKXGxZpKN+EY3KUgnJpbatv8OlQy9TnlbW5W/gbFQSsmlOUN1gzu
FGj6fEW6fNF8tj74nQ2SDpAl5A7s9aK1vVn6rX/dOhV0hZxPAoNMeksCgYEAwSvZ
rmNys+IaA2n5w9t/X7WFhrruD2rHqukdmKrQXnlwUcz+GVQHpUWYqYQMC+9radRg
a7AIIKQ+F0eFknZ3S8K+wfwovH5D+tNxohTRDBEaqMsXRfBE/NJaNe1RIqY6Gcdd
ceLS86oRqttvRvaV3/rYQfjgD7VfjacCz6JUdEcCgYAwKWfg7izSpKDMFz7Fd620
Z8RrVaYfgVrwibTO+eSu+N0XjMySETXBO5QZxmWywZXahY7lhjfNUQMVvh8N5Mc5
KfrRMDV67qOuFp99pShcUJJURpdiEb93KBvsv8F2OQVvdTl+A7upEgQH23JGQPIl
JWumSYQMhSYFPIn9V8rHOQKBgBkLh2idyjRaX0cMCW8EWWpeTZafS9hB3utg2A6A
Lw3grthcPKGqDGe4M0ffL/SoMQQCnhG4PAWHZel8w2uu4l63PCZIfDucH1I48eWy
zzvCR/OUiUrvEPK6jymowDk+1g+bkpj+cJ1Y8nt1geLwe5QToNBE5UAEIwRpn+qt
wEdnAoGAEpTvq9CclNGGJaV9/+y3fJGrnKfSMGmiBrmk6q8j5G36956QvvMYwI+/
kY7Ey0WQlaCrUj7Wut5F3ORXcJ+UOVXiz5IMhpWhdm8OY4bKVNXFRX6otbs8DesX
d/7GD2s85mcbJCAGYBT8XBawWfObWOPvZ3VQLalBfa92GQlmcgg=
-----END RSA PRIVATE KEY-----`

	invalidCAMaterial = "xyz"
)

func TestHandleCreateOrUpdate(t *testing.T) {
	var tests = []struct {
		setupMock func(s *mocks.Whisperer)
		err       error
	}{
		{
			func(w *mocks.Whisperer) {
				w.On("HasParameters", mock.Anything, mock.Anything).Return(true, nil)
				w.On("GetParameter", caCertPath).Return(&invalidCAMaterial, nil)
			},
			certDecodeError,
		},
		{
			func(w *mocks.Whisperer) {
				w.On("HasParameters", mock.Anything, mock.Anything).Return(true, nil)
				w.On("GetParameter", caCertPath).Return(&caCertMaterial, nil)
				w.On("GetParameter", caKeyPath).Return(&invalidCAMaterial, nil)
			},
			errors.New("data does not contain a valid RSA or ECDSA private key"),
		},
		{
			func(w *mocks.Whisperer) {
				w.On("HasParameters", mock.Anything, mock.Anything).Return(true, nil)
				w.On("GetParameter", caCertPath).Return(&caCertMaterial, nil)
				w.On("GetParameter", caKeyPath).Return(&caKeyMaterial, nil)
			},
			nil,
		},
		{
			func(w *mocks.Whisperer) {
				w.On("HasParameters", caCertPath, caKeyPath).Return(false, nil)
				w.On("HasParameters", mock.Anything, mock.Anything).Return(true, nil)
				w.On("ForceStoreParameter", caCertPath, mock.Anything, mock.Anything).Return(nil)
				w.On("ForceStoreParameter", caKeyPath, mock.Anything, mock.Anything).Return(nil)
			},
			nil,
		},
		{
			func(w *mocks.Whisperer) {
				w.On("HasParameters", bootstrapTokenPath).Return(true, nil)
				w.On("HasParameters", caCertPath, caKeyPath).Return(false, nil)
				w.On("HasParameters", mock.Anything, mock.Anything).Return(true, nil)
				w.On("ForceStoreParameter", caCertPath, mock.Anything, mock.Anything).Return(nil)
				w.On("ForceStoreParameter", caKeyPath, mock.Anything, mock.Anything).Return(nil)
			},
			nil,
		},
		{
			func(w *mocks.Whisperer) {
				w.On("HasParameters", bootstrapTokenPath).Return(true, nil)
				w.On("HasParameters", caCertPath, caKeyPath).Return(true, nil)
				w.On("HasParameters", etcdCACertPath, etcdCAKeyPath).Return(false, nil)
				w.On("HasParameters", mock.Anything, mock.Anything).Return(true, nil)
				w.On("GetParameter", caCertPath).Return(&caCertMaterial, nil)
				w.On("GetParameter", caKeyPath).Return(&caKeyMaterial, nil)
				w.On("ForceStoreParameter", etcdCACertPath, mock.Anything, mock.Anything).Return(nil)
				w.On("ForceStoreParameter", etcdCAKeyPath, mock.Anything, mock.Anything).Return(nil)
			},
			nil,
		},
		{
			func(w *mocks.Whisperer) {
				w.On("HasParameters", bootstrapTokenPath).Return(false, nil)
				w.On("HasParameters", mock.Anything, mock.Anything).Return(true, nil)
				w.On("GetParameter", caCertPath).Return(&caCertMaterial, nil)
				w.On("GetParameter", caKeyPath).Return(&caKeyMaterial, nil)
				w.On("StoreParameter", bootstrapTokenPath, mock.Anything, mock.Anything).Return(nil)
			},
			nil,
		},
		{
			func(w *mocks.Whisperer) {
				w.On("HasParameters", frontProxyCACertPath, frontProxyCAKeyPath).Return(false, nil)
				w.On("HasParameters", mock.Anything, mock.Anything).Return(true, nil)
				w.On("GetParameter", caCertPath).Return(&caCertMaterial, nil)
				w.On("GetParameter", caKeyPath).Return(&caKeyMaterial, nil)
				w.On("ForceStoreParameter", frontProxyCACertPath, mock.Anything, mock.Anything).Return(nil)
				w.On("ForceStoreParameter", frontProxyCAKeyPath, mock.Anything, mock.Anything).Return(nil)
			},
			nil,
		},
		{
			func(w *mocks.Whisperer) {
				w.On("HasParameters", apiClientCertPath, apiClientKeyPath).Return(false, nil)
				w.On("HasParameters", mock.Anything, mock.Anything).Return(true, nil)
				w.On("GetParameter", caCertPath).Return(&caCertMaterial, nil)
				w.On("GetParameter", caKeyPath).Return(&caKeyMaterial, nil)
				w.On("ForceStoreParameter", apiClientCertPath, mock.Anything, mock.Anything).Return(nil)
				w.On("ForceStoreParameter", apiClientKeyPath, mock.Anything, mock.Anything).Return(nil)
			},
			nil,
		},
		{
			func(w *mocks.Whisperer) {
				w.On("HasParameters", saSigningKeyPath, saSigningPubKeyPath).Return(false, nil)
				w.On("HasParameters", mock.Anything, mock.Anything).Return(true, nil)
				w.On("GetParameter", caCertPath).Return(&caCertMaterial, nil)
				w.On("GetParameter", caKeyPath).Return(&caKeyMaterial, nil)
				w.On("ForceStoreParameter", saSigningKeyPath, mock.Anything, mock.Anything).Return(nil)
				w.On("ForceStoreParameter", saSigningPubKeyPath, mock.Anything, mock.Anything).Return(nil)
			},
			nil,
		},
		{
			func(w *mocks.Whisperer) {
				w.On("HasParameters", bootstrapTokenPath).Return(false, nil)
				w.On("HasParameters", caCertPath, caKeyPath).Return(false, nil)
				w.On("HasParameters", etcdCACertPath, etcdCAKeyPath).Return(false, nil)
				w.On("HasParameters", frontProxyCACertPath, frontProxyCAKeyPath).Return(false, nil)
				w.On("HasParameters", apiClientCertPath, apiClientKeyPath).Return(false, nil)
				w.On("HasParameters", saSigningKeyPath, saSigningPubKeyPath).Return(false, nil)
				w.On("HasParameters", mock.Anything, mock.Anything).Return(true, nil)
				w.On("ForceStoreParameter", caCertPath, mock.Anything, mock.Anything).Return(nil)
				w.On("ForceStoreParameter", etcdCACertPath, mock.Anything, mock.Anything).Return(nil)
				w.On("ForceStoreParameter", etcdCAKeyPath, mock.Anything, mock.Anything).Return(nil)
				w.On("ForceStoreParameter", caKeyPath, mock.Anything, mock.Anything).Return(nil)
				w.On("ForceStoreParameter", frontProxyCACertPath, mock.Anything, mock.Anything).Return(nil)
				w.On("ForceStoreParameter", frontProxyCAKeyPath, mock.Anything, mock.Anything).Return(nil)
				w.On("ForceStoreParameter", apiClientCertPath, mock.Anything, mock.Anything).Return(nil)
				w.On("ForceStoreParameter", apiClientKeyPath, mock.Anything, mock.Anything).Return(nil)
				w.On("ForceStoreParameter", saSigningKeyPath, mock.Anything, mock.Anything).Return(nil)
				w.On("ForceStoreParameter", saSigningPubKeyPath, mock.Anything, mock.Anything).Return(nil)
				w.On("StoreParameter", bootstrapTokenPath, mock.Anything, mock.Anything).Return(nil)
			},
			nil,
		},
	}
	for _, test := range tests {
		whisp := &mocks.Whisperer{}
		props := resourceProperties{
			ServiceToken: "arn:aws:lambda:us-east-1:123456789012:function:kube-ca",
			ClusterName:  clusterName,
			KMSKeyID:     "alias/cloudboss",
		}
		test.setupMock(whisp)
		err := handleCreateOrUpdate(props, whisp)
		assert.Equal(t, test.err, err)
		whisp.AssertExpectations(t)
	}
}
