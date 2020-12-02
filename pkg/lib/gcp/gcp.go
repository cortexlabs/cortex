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
	"os"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
)

type Client struct {
	ProjectID    string
	Zone         string
	ClientEmail  string
	ClientId     string
	PrivateKey   string
	PrivateKeyId string
	clients      clients
}

type credentialsFile struct {
	ClientEmail  string `json:"client_email"`
	ClientId     string `json:"client_id"`
	PrivateKeyId string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ProjectId    string `json:"project_id"`
}

// Uses environment variable $GOOGLE_APPLICATION_CREDENTIALS
func NewFromEnv(projectID string, zone string) (*Client, error) {
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

	if credsFile.ProjectId != projectID {
		return nil, ErrorProjectIDMismatch(credsFile.ProjectId, projectID, credsFilePath)
	}

	return &Client{
		ProjectID:    projectID,
		Zone:         zone,
		ClientEmail:  credsFile.ClientEmail,
		ClientId:     credsFile.ClientId,
		PrivateKey:   credsFile.PrivateKey,
		PrivateKeyId: credsFile.PrivateKeyId,
	}, nil
}
