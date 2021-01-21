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
	"os"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/json"
)

type Client struct {
	ProjectID       string
	HashedProjectID string
	ClientEmail     string
	ClientID        string
	PrivateKey      string
	PrivateKeyID    string
	IsAnonymous     bool
	CredentialsJSON []byte
	clients         clients
}

type credentialsFile struct {
	ClientEmail  string `json:"client_email"`
	ClientID     string `json:"client_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ProjectID    string `json:"project_id"`
}

// Uses environment variable $GOOGLE_APPLICATION_CREDENTIALS
func NewFromEnvCheckProjectID(projectID string) (*Client, error) {
	client, err := NewFromEnv()
	if err != nil {
		return nil, err
	}

	if client.ProjectID != projectID {
		return nil, ErrorProjectIDMismatch(client.ProjectID, projectID, os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	}

	return client, nil
}

// Uses environment variable $GOOGLE_APPLICATION_CREDENTIALS
func NewFromEnv() (*Client, error) {
	credsFilePath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsFilePath == "" {
		return nil, ErrorCredentialsFileEnvVarNotSet()
	}

	credsBytes, err := files.ReadFileBytes(credsFilePath)
	if err != nil {
		return nil, err
	}

	credsFile := credentialsFile{}
	err = json.Unmarshal(credsBytes, &credsFile)
	if err != nil {
		return nil, errors.Wrap(err, credsFilePath)
	}

	return &Client{
		ProjectID:       credsFile.ProjectID,
		HashedProjectID: hash.String(credsFile.ProjectID),
		ClientEmail:     credsFile.ClientEmail,
		ClientID:        credsFile.ClientID,
		PrivateKey:      credsFile.PrivateKey,
		PrivateKeyID:    credsFile.PrivateKeyID,
		CredentialsJSON: credsBytes,
	}, nil
}

func NewAnonymousClient() *Client {
	return &Client{IsAnonymous: true}
}
