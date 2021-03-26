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
	"encoding/base64"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	grpcCodes "google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"
)

func (c *Client) GetCluster(fullyQualifiedClusterName string) (*containerpb.Cluster, error) {
	gke, err := c.GKE()
	if err != nil {
		return nil, err
	}
	cluster, err := gke.GetCluster(context.Background(), &containerpb.GetClusterRequest{
		Name: fullyQualifiedClusterName,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return cluster, nil
}

func (c *Client) ClusterExists(clusterName string) (bool, error) {
	if cluster, err := c.GetCluster(clusterName); err != nil {
		if errors.Cause(err) == nil {
			return false, err
		}
		if grpcError, ok := grpcStatus.FromError(errors.Cause(err)); ok {
			if grpcError.Code() == grpcCodes.NotFound {
				return false, nil
			}
		}
		return false, err
	} else if cluster != nil {
		return true, nil
	}
	return false, nil
}

func (c *Client) DeleteCluster(clusterName string) (*containerpb.Operation, error) {
	gke, err := c.GKE()
	if err != nil {
		return nil, err
	}
	resp, err := gke.DeleteCluster(context.Background(), &containerpb.DeleteClusterRequest{
		Name: clusterName,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return resp, nil
}

func (c *Client) CreateCluster(req *containerpb.CreateClusterRequest) (*containerpb.Operation, error) {
	gke, err := c.GKE()
	if err != nil {
		return nil, err
	}
	resp, err := gke.CreateCluster(context.Background(), req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return resp, nil
}

func (c *Client) CreateK8SConfigFromCluster(cluster *containerpb.Cluster) (*rest.Config, error) {
	googleAuthPlugin := "google"
	var _ rest.AuthProvider = &googleAuthProvider{}

	if err := rest.RegisterAuthProviderPlugin(googleAuthPlugin, NewGoogleAuthProvider); err != nil {
		return nil, err
	}

	decodedClientCertificate, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClientCertificate)
	if err != nil {
		return nil, err
	}
	decodedClientKey, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClientKey)
	if err != nil {
		return nil, err
	}
	decodedClusterCaCertificate, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return nil, err
	}

	return &rest.Config{
		AuthProvider: &clientcmdapi.AuthProviderConfig{Name: googleAuthPlugin},
		Host:         "https://" + cluster.Endpoint,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: false,
			CertData: decodedClientCertificate,
			KeyData:  decodedClientKey,
			CAData:   decodedClusterCaCertificate,
		},
	}, nil
}
