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

package spark

import (
	sparkop "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1alpha1"
	sparkopclientset "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/client/clientset/versioned"
	sparkopclientapi "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/client/clientset/versioned/typed/sparkoperator.k8s.io/v1alpha1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclientrest "k8s.io/client-go/rest"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
)

type Client struct {
	sparkClientset sparkopclientset.Interface
	sparkClient    sparkopclientapi.SparkApplicationInterface
}

func New(restConfig *kclientrest.Config, namespace string) (*Client, error) {
	var err error
	client := &Client{}
	client.sparkClientset, err = sparkopclientset.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "spark", "kubeconfig")
	}

	client.sparkClient = client.sparkClientset.SparkoperatorV1alpha1().SparkApplications(namespace)
	return client, nil
}

var sparkAppTypeMeta = kmeta.TypeMeta{
	APIVersion: "sparkoperator.k8s.io/v1alpha1",
	Kind:       "SparkApplication",
}

type Spec struct {
	Name      string
	Namespace string
	Spec      sparkop.SparkApplicationSpec
	Labels    map[string]string
}

func App(spec *Spec) *sparkop.SparkApplication {
	if spec.Namespace == "" {
		spec.Namespace = "default"
	}
	return &sparkop.SparkApplication{
		TypeMeta: sparkAppTypeMeta,
		ObjectMeta: kmeta.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
			Labels:    spec.Labels,
		},
		Spec: spec.Spec,
	}
}

func (c *Client) Create(sparkApp *sparkop.SparkApplication) (*sparkop.SparkApplication, error) {
	sparkApp.TypeMeta = sparkAppTypeMeta
	sparkApp, err := c.sparkClient.Create(sparkApp)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return sparkApp, nil
}

func (c *Client) update(sparkApp *sparkop.SparkApplication) (*sparkop.SparkApplication, error) {
	sparkApp.TypeMeta = sparkAppTypeMeta
	sparkApp, err := c.sparkClient.Update(sparkApp)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return sparkApp, nil
}

func (c *Client) Apply(sparkApp *sparkop.SparkApplication) (*sparkop.SparkApplication, error) {
	existing, err := c.Get(sparkApp.Name)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.Create(sparkApp)
	}
	return c.update(sparkApp)
}

func (c *Client) Get(name string) (*sparkop.SparkApplication, error) {
	sparkApp, err := c.sparkClient.Get(name, kmeta.GetOptions{})
	if kerrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	sparkApp.TypeMeta = sparkAppTypeMeta
	return sparkApp, nil
}

func (c *Client) Delete(appName string) (bool, error) {
	err := c.sparkClient.Delete(appName, &kmeta.DeleteOptions{})
	if kerrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) Exists(name string) (bool, error) {
	sparkApp, err := c.Get(name)
	if err != nil {
		return false, err
	}
	return sparkApp != nil, nil
}

func (c *Client) List(opts *kmeta.ListOptions) ([]sparkop.SparkApplication, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	sparkAppList, err := c.sparkClient.List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range sparkAppList.Items {
		sparkAppList.Items[i].TypeMeta = sparkAppTypeMeta
	}
	return sparkAppList.Items, nil
}

func (c *Client) ListByLabels(labels map[string]string) ([]sparkop.SparkApplication, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: k8s.LabelSelector(labels),
	}
	return c.List(opts)
}

func (c *Client) ListByLabel(labelKey string, labelValue string) ([]sparkop.SparkApplication, error) {
	return c.ListByLabels(map[string]string{labelKey: labelValue})
}

func Map(services []sparkop.SparkApplication) map[string]sparkop.SparkApplication {
	sparkAppMap := map[string]sparkop.SparkApplication{}
	for _, sparkApp := range services {
		sparkAppMap[sparkApp.Name] = sparkApp
	}
	return sparkAppMap
}

func (c *Client) IsRunning(name string) (bool, error) {
	sparkApp, err := c.Get(name)
	if err != nil {
		return false, err
	}
	if sparkApp == nil {
		return false, nil
	}
	return sparkApp.Status.CompletionTime.IsZero(), nil
}
