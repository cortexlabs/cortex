/*
Copyright 2020 Cortex Labs, Inc.

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
	"github.com/cortexlabs/cortex/pkg/operator/schema"
)

func gcpLogsQueryBuilder(apiName string) schema.GCPLogsResponse {
	gcpLogsResponse := schema.GCPLogsResponse{}
	gcpLogsResponse.Query = make(map[string]schema.QueryParam)

	gcpLogsResponse.Query["resource.type"] = schema.QueryParam{Param: "k8s_container"}
	gcpLogsResponse.Query["resource.labels.namespace_name"] = schema.QueryParam{Param: "default"}
	gcpLogsResponse.Query["resource.labels.project_id"] = schema.QueryParam{Param: *config.GCPCluster.Project}
	gcpLogsResponse.Query["resource.labels.location"] = schema.QueryParam{Param: *config.GCPCluster.Zone}
	gcpLogsResponse.Query["resource.labels.cluster_name"] = schema.QueryParam{Param: config.GCPCluster.ClusterName}
	gcpLogsResponse.Query["labels.k8s-pod/apiName"] = schema.QueryParam{Param: apiName}

	return gcpLogsResponse
}
