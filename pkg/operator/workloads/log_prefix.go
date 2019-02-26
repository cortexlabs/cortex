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
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
)

func logPreifixKey(workloadID string, appName string) string {
	return filepath.Join(
		consts.AppsDir,
		appName,
		consts.LogPrefixesDir,
		workloadID+".txt",
	)
}

func uploadLogPrefix(logPrefix string, workloadID string, appName string) error {
	if isLogPrefixCached(logPrefix, workloadID, appName) {
		return nil
	}
	key := logPreifixKey(workloadID, appName)
	err := aws.UploadStringToS3(logPrefix, key)
	if err != nil {
		return errors.Wrap(err, "upload log prefix", appName, workloadID)
	}
	cacheLogPrefix(logPrefix, workloadID, appName)
	return nil
}

type LogPrefixInfo struct {
	LogPrefix  string
	WorkloadID string
	AppName    string
}

func uploadLogPrefixes(logPrefixInfos []*LogPrefixInfo) error {
	fns := make([]func() error, len(logPrefixInfos))
	for i, logPrefixInfo := range logPrefixInfos {
		fns[i] = uploadLogPrefixFunc(logPrefixInfo)
	}
	return parallel.RunFirstErr(fns...)
}

func uploadLogPrefixFunc(logPrefixInfo *LogPrefixInfo) func() error {
	return func() error {
		return uploadLogPrefix(logPrefixInfo.LogPrefix, logPrefixInfo.WorkloadID, logPrefixInfo.AppName)
	}
}

func getSavedLogPrefix(workloadID string, appName string, allowNil bool) (string, error) {
	logPrefix := getCachedLogPrefix(workloadID, appName)
	if logPrefix != "" {
		return logPrefix, nil
	}
	key := logPreifixKey(workloadID, appName)
	logPrefix, err := aws.ReadStringFromS3(key)
	if err != nil {
		if aws.IsNoSuchKeyErr(err) && allowNil {
			return "", nil
		}
		return "", errors.Wrap(err, "download log prefix", appName, workloadID)
	}
	cacheLogPrefix(logPrefix, workloadID, appName)
	return logPrefix, nil
}

func UploadLogPrefixesFromAPIPods(pods []corev1.Pod) error {
	logPrefixInfos := []*LogPrefixInfo{}
	currentWorkloadIDs := make(map[string]strset.Set)
	for _, pod := range pods {
		if pod.Labels["workloadType"] != WorkloadTypeAPI {
			continue
		}

		workloadID := pod.Labels["workloadID"]
		appName := pod.Labels["appName"]
		if _, ok := currentWorkloadIDs[appName]; !ok {
			currentWorkloadIDs[appName] = strset.New()
		}

		if currentWorkloadIDs[appName].Has(workloadID) {
			continue
		}

		suffixIndex := strings.LastIndex(pod.Name, "-")
		logPrefixInfos = append(logPrefixInfos, &LogPrefixInfo{
			LogPrefix:  pod.Name[:suffixIndex+1],
			AppName:    appName,
			WorkloadID: workloadID,
		})

		currentWorkloadIDs[appName].Add(workloadID)
	}

	uncacheLogPrefixes(currentWorkloadIDs)
	return uploadLogPrefixes(logPrefixInfos)
}
