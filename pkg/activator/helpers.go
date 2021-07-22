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
	"strconv"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"k8s.io/apimachinery/pkg/api/meta"
)

type apiMeta struct {
	apiName        string
	apiKind        userconfig.Kind
	labels         map[string]string
	annotations    map[string]string
	maxConcurrency int
	maxQueueLength int
}

func getAPIMeta(obj interface{}) (apiMeta, error) {
	resource, err := meta.Accessor(obj)
	if err != nil {
		return apiMeta{}, err
	}

	labels := resource.GetLabels()
	apiKind, ok := labels["apiKind"]
	if !ok {
		return apiMeta{}, err
	}

	apiName, ok := labels["apiName"]
	if !ok {
		return apiMeta{}, errors.ErrorUnexpected("got a virtual service without apiName label")
	}

	annotations := resource.GetAnnotations()
	maxQueueLength, maxConcurrency, err := concurrencyFromAnnotations(annotations)
	if err != nil {
		return apiMeta{}, err
	}

	return apiMeta{
		apiName:        apiName,
		apiKind:        userconfig.KindFromString(apiKind),
		labels:         labels,
		annotations:    annotations,
		maxConcurrency: maxConcurrency,
		maxQueueLength: maxQueueLength,
	}, nil
}

// TODO move this out of here
func concurrencyFromAnnotations(annotations map[string]string) (int, int, error) {
	maxQueueLength, err := strconv.Atoi(annotations[userconfig.MaxQueueLengthAnnotationKey])
	if err != nil {
		return 0, 0, errors.ErrorUnexpected("failed to parse annotation", userconfig.MaxQueueLengthAnnotationKey)
	}

	maxConcurrency, err := strconv.Atoi(annotations[userconfig.MaxConcurrencyAnnotationKey])
	if err != nil {
		return 0, 0, errors.ErrorUnexpected("failed to parse annotation", userconfig.MaxConcurrencyAnnotationKey)
	}

	return maxQueueLength, maxConcurrency, err
}
