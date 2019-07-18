/*
Copyright 2019 Cortex Labs, Inc.

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
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

var configMapTypeMeta = kmeta.TypeMeta{
	APIVersion: "v1",
	Kind:       "ConfigMap",
}

type ConfigMapSpec struct {
	Name      string
	Namespace string
	Data      map[string]string
	Labels    map[string]string
}

func ConfigMap(spec *ConfigMapSpec) *kcore.ConfigMap {
	if spec.Namespace == "" {
		spec.Namespace = "default"
	}
	configMap := &kcore.ConfigMap{
		TypeMeta: configMapTypeMeta,
		ObjectMeta: kmeta.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
			Labels:    spec.Labels,
		},
		Data: spec.Data,
	}
	return configMap
}

func (c *Client) CreateConfigMap(configMap *kcore.ConfigMap) (*kcore.ConfigMap, error) {
	configMap.TypeMeta = configMapTypeMeta
	configMap, err := c.configMapClient.Create(configMap)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return configMap, nil
}

func (c *Client) updateConfigMap(configMap *kcore.ConfigMap) (*kcore.ConfigMap, error) {
	configMap.TypeMeta = configMapTypeMeta
	configMap, err := c.configMapClient.Update(configMap)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return configMap, nil
}

func (c *Client) ApplyConfigMap(configMap *kcore.ConfigMap) (*kcore.ConfigMap, error) {
	existing, err := c.GetConfigMap(configMap.Name)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.CreateConfigMap(configMap)
	}
	return c.updateConfigMap(configMap)
}

func (c *Client) GetConfigMap(name string) (*kcore.ConfigMap, error) {
	configMap, err := c.configMapClient.Get(name, kmeta.GetOptions{})
	if kerrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	configMap.TypeMeta = configMapTypeMeta
	return configMap, nil
}

func (c *Client) GetConfigMapData(name string) (map[string]string, error) {
	configMap, err := c.GetConfigMap(name)
	if err != nil {
		return nil, err
	}
	if configMap == nil {
		return nil, nil
	}
	return configMap.Data, nil
}

func (c *Client) DeleteConfigMap(name string) (bool, error) {
	err := c.configMapClient.Delete(name, deleteOpts)
	if kerrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) ConfigMapExists(name string) (bool, error) {
	configMap, err := c.GetConfigMap(name)
	if err != nil {
		return false, err
	}
	return configMap != nil, nil
}

func (c *Client) ListConfigMaps(opts *kmeta.ListOptions) ([]kcore.ConfigMap, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	configMapList, err := c.configMapClient.List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range configMapList.Items {
		configMapList.Items[i].TypeMeta = configMapTypeMeta
	}
	return configMapList.Items, nil
}

func (c *Client) ListConfigMapsByLabels(labels map[string]string) ([]kcore.ConfigMap, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelSelector(labels),
	}
	return c.ListConfigMaps(opts)
}

func (c *Client) ListConfigMapsByLabel(labelKey string, labelValue string) ([]kcore.ConfigMap, error) {
	return c.ListConfigMapsByLabels(map[string]string{labelKey: labelValue})
}

func ConfigMapMap(configMaps []kcore.ConfigMap) map[string]kcore.ConfigMap {
	configMapMap := map[string]kcore.ConfigMap{}
	for _, configMap := range configMaps {
		configMapMap[configMap.Name] = configMap
	}
	return configMapMap
}
