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

package gcp

import (
	"context"

	container "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/storage"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

type clients struct {
	gcs     *storage.Client
	compute *compute.Service
	gke     *container.ClusterManagerClient
	disks   *compute.DisksService
}

func (c *Client) GCS() (*storage.Client, error) {
	if c.clients.gcs == nil {
		var clientOption option.ClientOption
		if c.IsAnonymous {
			clientOption = option.WithoutAuthentication()
		} else {
			clientOption = option.WithCredentialsJSON(c.CredentialsJSON)
		}
		gcs, err := storage.NewClient(context.Background(), clientOption)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		c.clients.gcs = gcs
	}
	return c.clients.gcs, nil
}

func (c *Client) Compute() (*compute.Service, error) {
	if c.clients.compute == nil {
		var clientOption option.ClientOption
		if c.IsAnonymous {
			clientOption = option.WithoutAuthentication()
		} else {
			clientOption = option.WithCredentialsJSON(c.CredentialsJSON)
		}
		comp, err := compute.NewService(context.Background(), clientOption)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		c.clients.compute = comp
	}
	return c.clients.compute, nil
}

func (c *Client) GKE() (*container.ClusterManagerClient, error) {
	if c.clients.gke == nil {
		var clientOption option.ClientOption
		if c.IsAnonymous {
			clientOption = option.WithoutAuthentication()
		} else {
			clientOption = option.WithCredentialsJSON(c.CredentialsJSON)
		}
		gke, err := container.NewClusterManagerClient(context.Background(), clientOption)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		c.clients.gke = gke
	}
	return c.clients.gke, nil
}

func (c *Client) Disks() (*compute.DisksService, error) {
	if c.clients.disks == nil {
		comp, err := c.Compute()
		if err != nil {
			return nil, err
		}
		disks := compute.NewDisksService(comp)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		c.clients.disks = disks
	}
	return c.clients.disks, nil
}
