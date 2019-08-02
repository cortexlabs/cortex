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
	"github.com/cortexlabs/cortex/pkg/lib/errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kunstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kschema "k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	ruleTypeMeta = kmeta.TypeMeta{
		APIVersion: "v1alpha2",
		Kind:       "Rule",
	}

	ruleGVR = kschema.GroupVersionResource{
		Group:    "config.istio.io",
		Version:  "v1alpha2",
		Resource: "Rule",
	}

	ruleGVK = kschema.GroupVersionKind{
		Group:   "rule.istio.io",
		Version: "v1alpha2",
		Kind:    "Rule",
	}
)

type RuleSpec struct {
	Name             string
	Namespace        string
	Match            string
	Path             string
	handlerInstances map[string][]string
}

func Rule(spec *RuleSpec) *kunstructured.Unstructured {
	RuleConfig := &kunstructured.Unstructured{}
	RuleConfig.SetGroupVersionKind(ruleGVK)
	RuleConfig.SetName(spec.Name)
	RuleConfig.SetNamespace(spec.Namespace)
	RuleConfig.Object["metadata"] = map[string]interface{}{
		"name":      spec.Name,
		"namespace": spec.Namespace,
	}

	actions := []map[string]interface{}{}
	for handler, instances := range spec.handlerInstances {
		actions = append(actions, map[string]interface{}{
			handler: instances,
		})
	}

	RuleConfig.Object["spec"] = map[string]interface{}{
		"actions": actions,
	}

	return RuleConfig
}

func (c *Client) CreateRule(spec *kunstructured.Unstructured) (*kunstructured.Unstructured, error) {
	rule, err := c.dynamicClient.
		Resource(ruleGVR).
		Namespace(spec.GetNamespace()).
		Create(spec, kmeta.CreateOptions{
			TypeMeta: ruleTypeMeta,
		})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return rule, nil
}

func (c *Client) updateRule(spec *kunstructured.Unstructured) (*kunstructured.Unstructured, error) {
	rule, err := c.dynamicClient.
		Resource(ruleGVR).
		Namespace(spec.GetNamespace()).
		Update(spec, kmeta.UpdateOptions{
			TypeMeta: ruleTypeMeta,
		})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return rule, nil
}

func (c *Client) ApplyRule(spec *kunstructured.Unstructured) (*kunstructured.Unstructured, error) {
	existing, err := c.GetRule(spec.GetName(), spec.GetNamespace())
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.CreateRule(spec)
	}
	spec.SetResourceVersion(existing.GetResourceVersion())
	return c.updateVirtualService(spec)
}

func (c *Client) GetRule(name, namespace string) (*kunstructured.Unstructured, error) {
	rule, err := c.dynamicClient.Resource(ruleGVR).Namespace(namespace).Get(name, kmeta.GetOptions{
		TypeMeta: ruleTypeMeta,
	})

	if kerrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return rule, nil
}

func (c *Client) RuleExists(name, namespace string) (bool, error) {
	rule, err := c.GetRule(name, namespace)
	if err != nil {
		return false, err
	}
	return rule != nil, nil
}

func (c *Client) DeleteRule(name, namespace string) (bool, error) {
	err := c.dynamicClient.Resource(ruleGVR).Namespace(namespace).Delete(name, &kmeta.DeleteOptions{
		TypeMeta: ruleTypeMeta,
	})
	if kerrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) ListRules(namespace string, opts *kmeta.ListOptions) ([]kunstructured.Unstructured, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}

	vsList, err := c.dynamicClient.Resource(ruleGVR).Namespace(namespace).List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range vsList.Items {
		vsList.Items[i].SetGroupVersionKind(ruleGVK)
	}
	return vsList.Items, nil
}

func (c *Client) ListRulesByLabels(namespace string, labels map[string]string) ([]kunstructured.Unstructured, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelSelector(labels),
	}
	return c.ListRules(namespace, opts)
}

func (c *Client) ListRulesByLabel(namespace string, labelKey string, labelValue string) ([]kunstructured.Unstructured, error) {
	return c.ListRulesByLabels(namespace, map[string]string{labelKey: labelValue})
}
