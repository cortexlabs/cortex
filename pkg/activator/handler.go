/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Modifications Copyright 2022 Cortex Labs, Inc.
*/

package activator

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/proxy"
	"go.uber.org/zap"
)

type Handler struct {
	activator Activator
	logger    *zap.SugaredLogger
}

func NewHandler(act Activator, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		activator: act,
		logger:    logger,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if hasCortexProbeHeader(r) {
		w.WriteHeader(http.StatusOK)
		return
	}

	apiName := r.Header.Get(consts.CortexAPINameHeader)
	if apiName == "" {
		http.Error(w, fmt.Sprintf("missing %s", consts.CortexAPINameHeader), http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, APINameCtxKey, apiName)

	if err := h.activator.Try(ctx, func() error {
		return h.proxyRequest(w, r)
	}); err != nil {
		h.logger.Errorw("activator try error", zap.Error(err))

		if stderrors.Is(err, context.DeadlineExceeded) || stderrors.Is(err, proxy.ErrRequestQueueFull) {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			telemetry.Error(err)
		}
	}
}

func (h *Handler) proxyRequest(w http.ResponseWriter, r *http.Request) error {
	target := r.Header.Get(consts.CortexTargetServiceHeader)
	if target == "" {
		return errors.ErrorUnexpected("missing header", consts.CortexTargetServiceHeader)
	}

	targetURL, err := url.Parse(target)
	if err != nil {
		return errors.WithStack(err)
	}

	// delete activator specific headers
	r.Header.Del(consts.CortexAPINameHeader)
	r.Header.Del(consts.CortexTargetServiceHeader)

	reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)
	reverseProxy.ServeHTTP(w, r)

	return nil
}

func hasCortexProbeHeader(r *http.Request) bool {
	return r.Header.Get(consts.CortexProbeHeader) != ""
}
