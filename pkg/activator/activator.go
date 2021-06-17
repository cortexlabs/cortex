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
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	"go.uber.org/zap"
	istionetworkingclient "istio.io/client-go/pkg/clientset/versioned/typed/networking/v1beta1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

type ctxValue string

const APINameCtxKey ctxValue = "apiName"

type Activator interface {
	Try(ctx context.Context, fn func() error) error
}

type activator struct {
	apiActivators map[string]*apiActivator
	istioClient   istionetworkingclient.VirtualServiceInterface
	logger        *zap.SugaredLogger
}

func New(istioClient istionetworkingclient.VirtualServiceInterface, virtualServiceInformer cache.SharedIndexInformer, logger *zap.SugaredLogger) Activator {
	act := &activator{
		apiActivators: make(map[string]*apiActivator),
		istioClient:   istioClient,
		logger:        logger,
	}

	virtualServiceInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    act.addAPI,
			UpdateFunc: act.updateAPI,
			DeleteFunc: act.removeAPI,
		},
	)

	return act
}

func (a *activator) Try(ctx context.Context, fn func() error) error {
	apiNameValue := ctx.Value(APINameCtxKey)
	apiName, ok := apiNameValue.(string)
	if !ok || apiName == "" {
		return fmt.Errorf("failed to get the api name from context") // FIXME: proper error here
	}

	act, err := a.getOrCreateAPIActivator(ctx, apiName)
	if err != nil {
		return err
	}

	return act.Try(ctx, fn)
}

func (a *activator) getOrCreateAPIActivator(ctx context.Context, apiName string) (*apiActivator, error) {
	act, ok := a.apiActivators[apiName]
	if ok {
		return act, nil
	}

	vs, err := a.istioClient.Get(ctx, workloads.K8sName(apiName), kmeta.GetOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	maxQueueLength, maxConcurrency, err := concurrencyFromAnnotations(vs.Annotations)
	if err != nil {
		return nil, err
	}

	apiAct := newAPIActivator(apiName, maxQueueLength, maxConcurrency)
	a.apiActivators[apiName] = apiAct

	return apiAct, nil
}

func (a *activator) addAPI(obj interface{}) {
	apiMetadata, err := getAPIMeta(obj)
	if err != nil {
		a.logger.Errorw("error during virtual service informer add callback", zap.Error(err))
		return
	}

	if apiMetadata.apiKind != userconfig.RealtimeAPIKind {
		return
	}

	apiName := apiMetadata.apiName

	a.logger.Debugw("adding new api activator", zap.String("apiName", apiName))
	a.apiActivators[apiName] = newAPIActivator(apiName, apiMetadata.maxQueueLength, apiMetadata.maxConcurrency)
}

func (a *activator) updateAPI(_ interface{}, newObj interface{}) {
	apiMetadata, err := getAPIMeta(newObj)
	if err != nil {
		a.logger.Errorw("error during virtual service informer update callback", zap.Error(err))
		return
	}

	if apiMetadata.apiKind != userconfig.RealtimeAPIKind {
		return
	}

	apiName := apiMetadata.apiName

	if a.apiActivators[apiName].maxConcurrency != apiMetadata.maxConcurrency ||
		a.apiActivators[apiName].maxQueueLength != apiMetadata.maxQueueLength {

		a.logger.Infow("updating api activator", zap.String("apiName", apiName))
		a.apiActivators[apiName] = newAPIActivator(apiName, apiMetadata.maxQueueLength, apiMetadata.maxConcurrency)
	}
}

func (a *activator) removeAPI(obj interface{}) {
	apiMetadata, err := getAPIMeta(obj)
	if err != nil {
		a.logger.Errorw("error during virtual service informer delete callback", zap.Error(err))
		return
	}

	if apiMetadata.apiKind != userconfig.RealtimeAPIKind {
		return
	}

	a.logger.Debugw("deleting api activator", zap.String("apiName", apiMetadata.apiName))
	delete(a.apiActivators, apiMetadata.apiName)
}
