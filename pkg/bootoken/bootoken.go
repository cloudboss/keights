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

package bootoken

import (
	"fmt"

	apicorev1 "k8s.io/api/core/v1"
	apirbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"

	"github.com/cloudboss/keights/pkg/helpers"
)

const (
	secretIDTemplate            = "bootstrap-token-%s"
	kubeSystemNamespace         = "kube-system"
	kubeletBootstrapBinding     = "keights:kubelet-bootstrap"
	autoApproveBootstrapBinding = "keights:auto-approve-bootstrap"
	nodeBootstrapperRole        = "system:node-bootstrapper"
	csrRole                     = "system:certificates.k8s.io:certificatesigningrequests:nodeclient"
	bootstrappersGroup          = "system:bootstrappers"
)

type api struct {
	core *clientv1.CoreV1Client
	rbac *rbacv1.RbacV1Client
}

func newAPI(host string) (*api, error) {
	core, err := clientv1.NewForConfig(&rest.Config{Host: host})
	if err != nil {
		return nil, err
	}
	rbac, err := rbacv1.NewForConfig(&rest.Config{Host: host})
	if err != nil {
		return nil, err
	}
	return &api{
		core: core,
		rbac: rbac,
	}, nil
}

func finder(_item interface{}, err error) func() (bool, error) {
	return func() (bool, error) {
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
}

func (a *api) createBootstrapToken(namespace, name, tokenID, tokenSecret string) error {
	secrets := a.core.Secrets(namespace)
	return helpers.IdempotentDo(
		finder(secrets.Get(name, metav1.GetOptions{})),
		func() error {
			secret := &apicorev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Type: "bootstrap.kubernetes.io/token",
				StringData: map[string]string{
					"usage-bootstrap-authentication": "true",
					"usage-bootstrap-signing":        "true",
					"token-id":                       tokenID,
					"token-secret":                   tokenSecret,
				},
			}
			_, err := secrets.Create(secret)
			return err
		},
	)
}

func (a *api) createClusterRoleBinding(name, role, group string) error {
	bindings := a.rbac.ClusterRoleBindings()
	return helpers.IdempotentDo(
		finder(bindings.Get(name, metav1.GetOptions{})),
		func() error {
			clusterRoleBinding := &apirbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				RoleRef: apirbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     role,
				},
				Subjects: []apirbacv1.Subject{
					apirbacv1.Subject{
						Name: group,
						Kind: "Group",
					},
				},
			}
			_, err := bindings.Create(clusterRoleBinding)
			return err
		},
	)
}

func DoIt(apiserver, tokenID, tokenSecret string) error {
	api, err := newAPI(apiserver)
	if err != nil {
		return err
	}
	secretID := fmt.Sprintf(secretIDTemplate, tokenID)
	err = api.createBootstrapToken(kubeSystemNamespace, secretID, tokenID, tokenSecret)
	if err != nil {
		return err
	}
	err = api.createClusterRoleBinding(kubeletBootstrapBinding, nodeBootstrapperRole, bootstrappersGroup)
	if err != nil {
		return err
	}
	return api.createClusterRoleBinding(autoApproveBootstrapBinding, csrRole, bootstrappersGroup)
}
