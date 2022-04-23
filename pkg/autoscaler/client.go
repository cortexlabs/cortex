/*
Copyright 2022 Cortex Labs, Inc.

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

package autoscaler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

type Client interface {
	Awaken(api userconfig.Resource) error
}

type client struct {
	httpClient *http.Client
	endpoint   string
}

func NewClient(endpoint string) Client {
	return &client{
		httpClient: &http.Client{},
		endpoint:   endpoint,
	}
}

func (c *client) Awaken(api userconfig.Resource) error {
	payload, err := json.Marshal(api)
	if err != nil {
		return err
	}

	response, err := c.httpClient.Post(
		urls.Join(c.endpoint, "/awaken"), "application/json", bytes.NewBuffer(payload),
	)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(response.Body)
		errMsg := fmt.Sprintf("failed to awake api (status code %d)", response.StatusCode)
		if bodyBytes != nil {
			errMsg = errMsg + fmt.Sprintf(": %s", string(bodyBytes))
		}

		return fmt.Errorf(errMsg)
	}

	return nil
}
