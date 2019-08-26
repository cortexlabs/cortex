package k8s

import (
	"github.com/cortexlabs/cortex/pkg/lib/errors"

	kcore "k8s.io/api/core/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) ListNodes(opts *kmeta.ListOptions) ([]kcore.Node, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	podList, err := c.nodeClient.List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range podList.Items {
		podList.Items[i].TypeMeta = podTypeMeta
	}
	return podList.Items, nil
}
