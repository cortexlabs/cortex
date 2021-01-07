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
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

var _nodeTypeMeta = kmeta.TypeMeta{
	APIVersion: "v1",
	Kind:       "Node",
}

func (c *Client) ListNodes(opts *kmeta.ListOptions) ([]kcore.Node, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	nodeList, err := c.nodeClient.List(context.Background(), *opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range nodeList.Items {
		nodeList.Items[i].TypeMeta = _nodeTypeMeta
	}
	return nodeList.Items, nil
}

func (c *Client) ListNodesByLabels(labels map[string]string) ([]kcore.Node, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(labels).String(),
	}
	return c.ListNodes(opts)
}

func (c *Client) ListNodesByLabel(labelKey string, labelValue string) ([]kcore.Node, error) {
	return c.ListNodesByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) ListNodesWithLabelKeys(labelKeys ...string) ([]kcore.Node, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelExistsSelector(labelKeys...),
	}
	return c.ListNodes(opts)
}
