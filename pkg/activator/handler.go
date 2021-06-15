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

package activator

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/proxy"
	"go.uber.org/zap"
)

const (
	CortexAPINameHeader       = "X-Cortex-API-Name"
	CortexTargetServiceHeader = "X-Cortex-Target-Service"
)

type activatorHandler struct {
	activator Activator
	logger    *zap.SugaredLogger
}

func NewHandler(act Activator, logger *zap.SugaredLogger) *activatorHandler {
	return &activatorHandler{
		activator: act,
		logger:    logger,
	}
}

func (h *activatorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	apiName := r.Header.Get(CortexAPINameHeader)

	ctx := r.Context()
	ctx = context.WithValue(ctx, ApiNameCtxKey, apiName)

	if err := h.activator.Try(ctx, func() error {
		return h.proxyRequest(w, r)
	}); err != nil {
		h.logger.Errorw("activator try error", zap.Error(err))

		if stderrors.Is(err, context.DeadlineExceeded) || stderrors.Is(err, proxy.ErrRequestQueueFull) {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func (h *activatorHandler) proxyRequest(w http.ResponseWriter, r *http.Request) error {
	target := r.Header.Get(CortexTargetServiceHeader)
	if target == "" {
		return fmt.Errorf("missing %s header", CortexTargetServiceHeader) // FIXME: proper error
	}

	targetURL, err := url.Parse(target)
	if err != nil {
		return errors.WithStack(err)
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)
	reverseProxy.ServeHTTP(w, r)

	return nil
}
