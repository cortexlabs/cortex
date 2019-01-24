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
	"sync"

	"github.com/cortexlabs/cortex/pkg/api/context"
)

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
	ctxs := make([]*context.Context, len(currentCtxs.m))
	i := 0
	for _, ctx := range currentCtxs.m {
		ctxs[i] = ctx
		i++
	}
	return ctxs
}

func setCurrentContext(ctx *context.Context) {
	currentCtxs.Lock()
	defer currentCtxs.Unlock()
	currentCtxs.m[ctx.App.Name] = ctx
}

func deleteCurrentContext(appName string) {
	currentCtxs.Lock()
	defer currentCtxs.Unlock()
	delete(currentCtxs.m, appName)
}
