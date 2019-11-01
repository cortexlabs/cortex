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

package clusterconfig

import (
	"github.com/cortexlabs/cortex/pkg/lib/table"
)

func (cc *InternalClusterConfig) String() string {
	defaultCC, _ := GetFileDefaults()

	var items []table.KV

	items = append(items, table.KV{K: "cluster version", V: cc.APIVersion})
	items = append(items, table.KV{K: "instance type", V: *cc.InstanceType})
	items = append(items, table.KV{K: "min instances", V: *cc.MinInstances})
	items = append(items, table.KV{K: "max instances", V: *cc.MaxInstances})
	items = append(items, table.KV{K: "cluster name", V: cc.ClusterName})
	items = append(items, table.KV{K: "region", V: cc.Region})
	items = append(items, table.KV{K: "bucket", V: cc.Bucket})
	items = append(items, table.KV{K: "log group", V: cc.LogGroup})
	items = append(items, table.KV{K: "telemetry", V: cc.Telemetry})

	if cc.ImagePredictorServe != defaultCC.ImagePredictorServe {
		items = append(items, table.KV{K: "image_predictor_serve", V: cc.ImagePredictorServe})
	}
	if cc.ImagePredictorServeGPU != defaultCC.ImagePredictorServeGPU {
		items = append(items, table.KV{K: "image_predictor_serve_gpu", V: cc.ImagePredictorServeGPU})
	}
	if cc.ImageTFServe != defaultCC.ImageTFServe {
		items = append(items, table.KV{K: "image_tf_serve", V: cc.ImageTFServe})
	}
	if cc.ImageTFServeGPU != defaultCC.ImageTFServeGPU {
		items = append(items, table.KV{K: "image_tf_serve_gpu", V: cc.ImageTFServeGPU})
	}
	if cc.ImageTFAPI != defaultCC.ImageTFAPI {
		items = append(items, table.KV{K: "image_tf_api", V: cc.ImageTFAPI})
	}
	if cc.ImageONNXServe != defaultCC.ImageONNXServe {
		items = append(items, table.KV{K: "image_onnx_serve", V: cc.ImageONNXServe})
	}
	if cc.ImageONNXServeGPU != defaultCC.ImageONNXServeGPU {
		items = append(items, table.KV{K: "image_onnx_serve_gpu", V: cc.ImageONNXServeGPU})
	}
	if cc.ImageOperator != defaultCC.ImageOperator {
		items = append(items, table.KV{K: "image_operator", V: cc.ImageOperator})
	}
	if cc.ImageManager != defaultCC.ImageManager {
		items = append(items, table.KV{K: "image_manager", V: cc.ImageManager})
	}
	if cc.ImageDownloader != defaultCC.ImageDownloader {
		items = append(items, table.KV{K: "image_downloader", V: cc.ImageDownloader})
	}
	if cc.ImageClusterAutoscaler != defaultCC.ImageClusterAutoscaler {
		items = append(items, table.KV{K: "image_cluster_autoscaler", V: cc.ImageClusterAutoscaler})
	}
	if cc.ImageMetricsServer != defaultCC.ImageMetricsServer {
		items = append(items, table.KV{K: "image_metrics_server", V: cc.ImageMetricsServer})
	}
	if cc.ImageNvidia != defaultCC.ImageNvidia {
		items = append(items, table.KV{K: "image_nvidia", V: cc.ImageNvidia})
	}
	if cc.ImageFluentd != defaultCC.ImageFluentd {
		items = append(items, table.KV{K: "image_fluentd", V: cc.ImageFluentd})
	}
	if cc.ImageStatsd != defaultCC.ImageStatsd {
		items = append(items, table.KV{K: "image_statsd", V: cc.ImageStatsd})
	}
	if cc.ImageIstioProxy != defaultCC.ImageIstioProxy {
		items = append(items, table.KV{K: "image_istio_proxy", V: cc.ImageIstioProxy})
	}
	if cc.ImageIstioPilot != defaultCC.ImageIstioPilot {
		items = append(items, table.KV{K: "image_istio_pilot", V: cc.ImageIstioPilot})
	}
	if cc.ImageIstioCitadel != defaultCC.ImageIstioCitadel {
		items = append(items, table.KV{K: "image_istio_citadel", V: cc.ImageIstioCitadel})
	}
	if cc.ImageIstioGalley != defaultCC.ImageIstioGalley {
		items = append(items, table.KV{K: "image_istio_galley", V: cc.ImageIstioGalley})
	}

	return table.AlignKeyValue(items, ":", 1)
}
