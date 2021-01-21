/*
Copyright 2021 Cortex Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package k8s

import (
	"context"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

var _secretTypeMeta = kmeta.TypeMeta{
	APIVersion: "v1",
	Kind:       "Secret",
}

type SecretSpec struct {
	Name        string
	Data        map[string][]byte
	Labels      map[string]string
	Annotations map[string]string
}

func Secret(spec *SecretSpec) *kcore.Secret {
	secret := &kcore.Secret{
		TypeMeta: _secretTypeMeta,
		ObjectMeta: kmeta.ObjectMeta{
			Name:        spec.Name,
			Labels:      spec.Labels,
			Annotations: spec.Annotations,
		},
		Data: spec.Data,
	}
	return secret
}

func (c *Client) CreateSecret(secret *kcore.Secret) (*kcore.Secret, error) {
	secret.TypeMeta = _secretTypeMeta
	secret, err := c.secretClient.Create(context.Background(), secret, kmeta.CreateOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return secret, nil
}

func (c *Client) UpdateSecret(secret *kcore.Secret) (*kcore.Secret, error) {
	secret.TypeMeta = _secretTypeMeta
	secret, err := c.secretClient.Update(context.Background(), secret, kmeta.UpdateOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return secret, nil
}

func (c *Client) ApplySecret(secret *kcore.Secret) (*kcore.Secret, error) {
	existing, err := c.GetSecret(secret.Name)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.CreateSecret(secret)
	}
	return c.UpdateSecret(secret)
}

func (c *Client) GetSecret(name string) (*kcore.Secret, error) {
	secret, err := c.secretClient.Get(context.Background(), name, kmeta.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}
	secret.TypeMeta = _secretTypeMeta
	return secret, nil
}

func (c *Client) GetSecretData(name string) (map[string][]byte, error) {
	secret, err := c.GetSecret(name)
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, nil
	}
	return secret.Data, nil
}

func (c *Client) DeleteSecret(name string) (bool, error) {
	err := c.secretClient.Delete(context.Background(), name, _deleteOpts)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) ListSecrets(opts *kmeta.ListOptions) ([]kcore.Secret, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	secretList, err := c.secretClient.List(context.Background(), *opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range secretList.Items {
		secretList.Items[i].TypeMeta = _secretTypeMeta
	}
	return secretList.Items, nil
}

func (c *Client) ListSecretsByLabels(labels map[string]string) ([]kcore.Secret, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(labels).String(),
	}
	return c.ListSecrets(opts)
}

func (c *Client) ListSecretsByLabel(labelKey string, labelValue string) ([]kcore.Secret, error) {
	return c.ListSecretsByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) ListSecretsWithLabelKeys(labelKeys ...string) ([]kcore.Secret, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelExistsSelector(labelKeys...),
	}
	return c.ListSecrets(opts)
}

func SecretMap(secrets []kcore.Secret) map[string]kcore.Secret {
	secretMap := map[string]kcore.Secret{}
	for _, secret := range secrets {
		secretMap[secret.Name] = secret
	}
	return secretMap
}
