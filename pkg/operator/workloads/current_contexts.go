/*
Copyright 2019 Cortex Labs, Inc.

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

package workloads

import (
	"fmt"
	"sync"

	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	ocontext "github.com/cortexlabs/cortex/pkg/operator/context"
)

const configMapName = "cortex-current-contexts"

// appName -> currently deployed context
var currentCtxs = struct {
	m map[string]*context.Context
	sync.RWMutex
}{m: make(map[string]*context.Context)}

func CurrentContext(appName string) *context.Context {
	currentCtxs.RLock()
	defer currentCtxs.RUnlock()
	return currentCtxs.m[appName]
}

func CurrentContexts() []*context.Context {
	currentCtxs.RLock()
	defer currentCtxs.RUnlock()
	ctxs := make([]*context.Context, 0, len(currentCtxs.m))
	for _, ctx := range currentCtxs.m {
		ctxs = append(ctxs, ctx)
	}
	return ctxs
}

func setCurrentContext(ctx *context.Context) error {
	currentCtxs.Lock()
	defer currentCtxs.Unlock()

	currentCtxs.m[ctx.App.Name] = ctx

	err := updateContextConfigMap()
	if err != nil {
		return err
	}

	return nil
}

func deleteCurrentContext(appName string) error {
	currentCtxs.Lock()
	defer currentCtxs.Unlock()

	delete(currentCtxs.m, appName)

	err := updateContextConfigMap()
	if err != nil {
		return err
	}

	return nil
}

func updateContextConfigMap() error {
	configMapData := make(map[string]string, len(currentCtxs.m))
	for appName, ctx := range currentCtxs.m {
		configMapData[appName] = ctx.ID
	}

	configMap := k8s.ConfigMap(&k8s.ConfigMapSpec{
		Name:      configMapName,
		Namespace: config.Cortex.Namespace,
		Data:      configMapData,
	})

	_, err := config.Kubernetes.ApplyConfigMap(configMap)
	if err != nil {
		return err
	}

	return nil
}

func reloadCurrentContexts() error {
	currentCtxs.Lock()
	defer currentCtxs.Unlock()

	configMap, err := config.Kubernetes.GetConfigMap(configMapName)
	if err != nil {
		return err
	}
	if configMap == nil {
		return nil
	}

	for appName, ctxID := range configMap.Data {
		ctx, err := ocontext.DownloadContext(ctxID, appName)
		if err != nil {
			fmt.Printf("Deleting stale workflow: %s", appName)
			DeleteApp(appName, true)
		} else if ctx != nil {
			currentCtxs.m[appName] = ctx
		}
	}

	return nil
}
