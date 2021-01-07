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

package endpoints

import (
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

func gcpLogsQueryParams(apiName string) map[string]string {
	queryParams := make(map[string]string)

	queryParams["resource.type"] = "k8s_container"
	queryParams["resource.labels.namespace_name"] = "default"
	queryParams["resource.labels.project_id"] = *config.GCPCluster.Project
	queryParams["resource.labels.location"] = *config.GCPCluster.Zone
	queryParams["resource.labels.cluster_name"] = config.GCPCluster.ClusterName
	queryParams["labels.k8s-pod/apiName"] = apiName

	return queryParams
}
