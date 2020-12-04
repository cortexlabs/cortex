/*
Copyright 2020 Cortex Labs, Inc.

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

	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func (c *Client) GetCluster(clusterName string) (*containerpb.Cluster, error) {
	gke, err := c.GKE()
	if err != nil {
		return nil, err
	}
	cluster, err := gke.GetCluster(context.Background(), &containerpb.GetClusterRequest{
		Name: clusterName,
	})
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (c *Client) CreateK8SConfigFromCluster(cluster *containerpb.Cluster) (*rest.Config, error) {
	googleAuthPlugin := "google"
	var _ rest.AuthProvider = &googleAuthProvider{}

	if err := rest.RegisterAuthProviderPlugin(googleAuthPlugin, newGoogleAuthProvider); err != nil {
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
