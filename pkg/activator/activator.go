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
	"sync"

	"github.com/cortexlabs/cortex/pkg/autoscaler"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	"go.uber.org/zap"
	istionetworkingclient "istio.io/client-go/pkg/clientset/versioned/typed/networking/v1beta1"
	kapps "k8s.io/api/apps/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

type ctxValue string

const APINameCtxKey ctxValue = "apiName"

type StatsReporter interface {
	AddAPI(apiName string)
	RemoveAPI(apiName string)
}

type Activator interface {
	Try(ctx context.Context, fn func() error) error
}

type activator struct {
	activatorsMux     sync.Mutex
	trackersMux       sync.Mutex
	autoscalerClient  autoscaler.Client
	apiActivators     map[string]*apiActivator
	readinessTrackers map[string]*readinessTracker
	istioClient       istionetworkingclient.VirtualServiceInterface
	reporter          StatsReporter
	logger            *zap.SugaredLogger
}

func New(
	istioClient istionetworkingclient.VirtualServiceInterface,
	deploymentInformer cache.SharedIndexInformer,
	virtualServiceInformer cache.SharedIndexInformer,
	autoscalerClient autoscaler.Client,
	reporter StatsReporter,
	logger *zap.SugaredLogger,
) Activator {
	log := logger.With(zap.String("apiKind", userconfig.RealtimeAPIKind.String()))

	act := &activator{
		apiActivators:     make(map[string]*apiActivator),
		readinessTrackers: make(map[string]*readinessTracker),
		istioClient:       istioClient,
		logger:            log,
		autoscalerClient:  autoscalerClient,
		reporter:          reporter,
	}

	virtualServiceInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    act.addAPI,
			UpdateFunc: act.updateAPI,
			DeleteFunc: act.removeAPI,
		},
	)

	deploymentInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: act.updateReadinessTracker,
			UpdateFunc: func(_, newObj interface{}) {
				act.updateReadinessTracker(newObj)
			},
			DeleteFunc: act.removeReadinessTracker,
		},
	)

	return act
}

func (a *activator) Try(ctx context.Context, fn func() error) error {
	apiNameValue := ctx.Value(APINameCtxKey)
	apiName, ok := apiNameValue.(string)
	if !ok || apiName == "" {
		return errors.ErrorUnexpected("failed to get the api name from context")
	}

	act, err := a.getOrCreateAPIActivator(ctx, apiName)
	if err != nil {
		return err
	}

	tracker := a.getOrCreateReadinessTracker(apiName)

	if act.inFlight() == 0 {
		go a.awakenAPI(apiName)
	}

	return act.try(ctx, fn, tracker)
}

func (a *activator) getOrCreateAPIActivator(ctx context.Context, apiName string) (*apiActivator, error) {
	a.activatorsMux.Lock()
	defer a.activatorsMux.Unlock()

	act, ok := a.apiActivators[apiName]
	if ok {
		return act, nil
	}

	vs, err := a.istioClient.Get(ctx, workloads.K8sName(apiName), kmeta.GetOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	maxQueueLength, maxConcurrency, err := userconfig.ConcurrencyFromAnnotations(vs)
	if err != nil {
		return nil, err
	}

	apiAct := newAPIActivator(maxQueueLength, maxConcurrency)

	a.apiActivators[apiName] = apiAct

	return apiAct, nil
}

func (a *activator) getOrCreateReadinessTracker(apiName string) *readinessTracker {
	a.trackersMux.Lock()
	defer a.trackersMux.Unlock()
	tracker, ok := a.readinessTrackers[apiName]
	if ok {
		return tracker
	}

	tracker = newReadinessTracker()
	a.readinessTrackers[apiName] = tracker

	return tracker
}

func (a *activator) addAPI(obj interface{}) {
	apiMetadata, err := getAPIMeta(obj)
	if err != nil {
		a.logger.Errorw("error during virtual service informer add callback", zap.Error(err))
		telemetry.Error(err)
		return
	}

	if apiMetadata.apiKind != userconfig.RealtimeAPIKind {
		return
	}

	apiName := apiMetadata.apiName

	a.activatorsMux.Lock()
	if a.apiActivators[apiName] == nil {
		a.logger.Debugw("adding new api activator", zap.String("apiName", apiName))
		a.apiActivators[apiName] = newAPIActivator(apiMetadata.maxQueueLength, apiMetadata.maxConcurrency)
	}
	a.activatorsMux.Unlock()

	a.reporter.AddAPI(apiName)
}

func (a *activator) updateAPI(oldObj interface{}, newObj interface{}) {
	apiMetadata, err := getAPIMeta(newObj)
	if err != nil {
		a.logger.Errorw("error during virtual service informer update callback", zap.Error(err))
		telemetry.Error(err)
		return
	}

	if apiMetadata.apiKind != userconfig.RealtimeAPIKind {
		return
	}

	apiName := apiMetadata.apiName

	oldAPIMetatada, err := getAPIMeta(oldObj)
	if err != nil {
		a.logger.Errorw("error during virtual service informer update callback", zap.Error(err))
		telemetry.Error(err)
		return
	}

	if oldAPIMetatada.maxConcurrency != apiMetadata.maxConcurrency || oldAPIMetatada.maxQueueLength != apiMetadata.maxQueueLength {
		a.logger.Debugw("updating api activator", zap.String("apiName", apiName))

		a.activatorsMux.Lock()
		a.apiActivators[apiName].updateQueueParams(apiMetadata.maxQueueLength, apiMetadata.maxConcurrency)
		a.activatorsMux.Unlock()
	}
}

func (a *activator) removeAPI(obj interface{}) {
	apiMetadata, err := getAPIMeta(obj)
	if err != nil {
		a.logger.Errorw("error during virtual service informer delete callback", zap.Error(err))
		telemetry.Error(err)
		return
	}

	if apiMetadata.apiKind != userconfig.RealtimeAPIKind {
		return
	}

	a.logger.Debugw("deleting api activator", zap.String("apiName", apiMetadata.apiName))

	a.activatorsMux.Lock()
	delete(a.apiActivators, apiMetadata.apiName)
	a.activatorsMux.Unlock()

	a.reporter.RemoveAPI(apiMetadata.apiName)
}

func (a *activator) awakenAPI(apiName string) {
	err := a.autoscalerClient.Awaken(
		userconfig.Resource{
			Name: apiName,
			Kind: userconfig.RealtimeAPIKind, // only realtime apis are supported as of now, so we can assume the kind
		},
	)
	if err != nil {
		a.logger.Errorw("failed to awake api", zap.Error(err), zap.String("apiName", apiName))
	}
}

func (a *activator) updateReadinessTracker(obj interface{}) {
	deployment, ok := obj.(*kapps.Deployment)
	if !ok {
		return
	}

	api, err := getAPIMeta(obj)
	if err != nil {
		a.logger.Errorw("error during deployment informer callback", zap.Error(err))
		telemetry.Error(err)
		return
	}

	if api.apiKind != userconfig.RealtimeAPIKind {
		return
	}

	tracker := a.getOrCreateReadinessTracker(api.apiName)
	tracker.Update(deployment)

	a.logger.Debugw("updated readiness tracker",
		zap.Bool("ready", deployment.Status.ReadyReplicas > 0),
		zap.String("apiName", api.apiName),
	)
}

func (a *activator) removeReadinessTracker(obj interface{}) {
	api, err := getAPIMeta(obj)
	if err != nil {
		a.logger.Errorw("error during deployment informer callback", zap.Error(err))
		telemetry.Error(err)
		return
	}

	if api.apiKind != userconfig.RealtimeAPIKind {
		return
	}

	a.trackersMux.Lock()
	defer a.trackersMux.Unlock()
	delete(a.readinessTrackers, api.apiName)
}
