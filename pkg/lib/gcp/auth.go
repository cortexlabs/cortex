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
	"net/http"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"k8s.io/client-go/rest"
)

type googleAuthProvider struct {
	tokenSource oauth2.TokenSource
}

func (g *googleAuthProvider) WrapTransport(rt http.RoundTripper) http.RoundTripper {
	return &oauth2.Transport{
		Base:   rt,
		Source: g.tokenSource,
	}
}
func (g *googleAuthProvider) Login() error { return nil }

func NewGoogleAuthProvider(addr string, config map[string]string, persister rest.AuthProviderConfigPersister) (rest.AuthProvider, error) {
	ts, err := google.DefaultTokenSource(
		context.Background(),
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/userinfo.email",
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &googleAuthProvider{tokenSource: ts}, nil
}
