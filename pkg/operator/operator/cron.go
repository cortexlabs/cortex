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

package operator

import (
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/lib/logging"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var operatorLogger = logging.GetOperatorLogger()
var previousListOfEvictedPods = strset.New()

func DeleteEvictedPods() error {
	failedPods, err := config.K8s.ListPods(&kmeta.ListOptions{
		FieldSelector: "status.phase=Failed",
	})
	if err != nil {
		return err
	}

	var errs []error
	currentEvictedPods := strset.New()
	for _, pod := range failedPods {
		if pod.Status.Reason != k8s.ReasonEvicted {
			continue
		}
		if previousListOfEvictedPods.Has(pod.Name) {
			_, err := config.K8s.DeletePod(pod.Name)
			if err != nil {
				errs = append(errs, err)
			}
			continue
		}
		currentEvictedPods.Add(pod.Name)
	}
	previousListOfEvictedPods = currentEvictedPods

	if errors.HasError(errs) {
		return errors.FirstError(errs...)
	}
	return nil
}

type instanceInfo struct {
	InstanceType  string  `json:"instance_type" yaml:"instance_type"`
	IsSpot        bool    `json:"is_spot" yaml:"is_spot"`
	Price         float64 `json:"price" yaml:"price"`
	OnDemandPrice float64 `json:"on_demand_price" yaml:"on_demand_price"`
	Count         int32   `json:"count" yaml:"count"`
	Memory        int64   `json:"memory" yaml:"memory"`
	CPU           float64 `json:"cpu" yaml:"cpu"`
	GPU           int64   `json:"gpu" yaml:"gpu"`
	Inf           int64   `json:"inf" yaml:"inf"`
	GPUType       string  `json:"gpu_type" yaml:"gpu_type"` // currently only used in GCP
}

func InstanceTelemetryAWS() error {
	nodes, err := config.K8s.ListNodes(nil)
	if err != nil {
		return err
	}

	instanceInfos := make(map[string]*instanceInfo)
	var totalInstances int

	for _, node := range nodes {
		if node.Labels["workload"] != "true" {
			continue
		}

		instanceType := node.Labels["beta.kubernetes.io/instance-type"]
		if instanceType == "" {
			instanceType = "unknown"
		}

		isSpot := false
		if strings.Contains(strings.ToLower(node.Labels["lifecycle"]), "spot") {
			isSpot = true
		}

		totalInstances++

		instanceInfosKey := instanceType + "_ondemand"
		if isSpot {
			instanceInfosKey = instanceType + "_spot"
		}

		if info, ok := instanceInfos[instanceInfosKey]; ok {
			info.Count++
			continue
		}

		onDemandPrice := aws.InstanceMetadatas[*config.Cluster.Region][instanceType].Price
		price := onDemandPrice
		if isSpot {
			spotPrice, err := config.AWS.SpotInstancePrice(instanceType)
			if err == nil && spotPrice != 0 {
				price = spotPrice
			}
		}

		gpuQty := node.Status.Capacity["nvidia.com/gpu"]
		infQty := node.Status.Capacity["aws.amazon.com/neuron"]

		// For AWS, use the instance type as the GPU type
		gpuType := ""
		if gpuQty.Value() > 0 {
			gpuType = instanceType
		}

		info := instanceInfo{
			InstanceType:  instanceType,
			IsSpot:        isSpot,
			Price:         price,
			OnDemandPrice: onDemandPrice,
			Count:         1,
			Memory:        node.Status.Capacity.Memory().Value(),
			CPU:           float64(node.Status.Capacity.Cpu().MilliValue()) / 1000,
			GPU:           gpuQty.Value(),
			Inf:           infQty.Value(),
			GPUType:       gpuType,
		}

		instanceInfos[instanceInfosKey] = &info
	}

	apiEBSPrice := aws.EBSMetadatas[*config.Cluster.Region][config.Cluster.InstanceVolumeType.String()].PriceGB * float64(config.Cluster.InstanceVolumeSize) / 30 / 24
	if config.Cluster.InstanceVolumeType.String() == "io1" && config.Cluster.InstanceVolumeIOPS != nil {
		apiEBSPrice += aws.EBSMetadatas[*config.Cluster.Region][config.Cluster.InstanceVolumeType.String()].PriceIOPS * float64(*config.Cluster.InstanceVolumeIOPS) / 30 / 24
	}

	var totalInstancePrice float64
	var totalInstancePriceIfOnDemand float64
	for _, info := range instanceInfos {
		totalInstancePrice += (info.Price + apiEBSPrice) * float64(info.Count)
		totalInstancePriceIfOnDemand += (info.OnDemandPrice + apiEBSPrice) * float64(info.Count)
	}

	fixedPrice := clusterFixedPriceAWS()

	properties := map[string]interface{}{
		"region":                      *config.Cluster.Region,
		"instance_count":              totalInstances,
		"instances":                   instanceInfos,
		"fixed_price":                 fixedPrice,
		"workload_price":              totalInstancePrice,
		"workload_price_if_on_demand": totalInstancePriceIfOnDemand,
		"total_price":                 totalInstancePrice + fixedPrice,
		"total_price_if_on_demand":    totalInstancePriceIfOnDemand + fixedPrice,
	}

	telemetry.Event("operator.cron", properties, config.Cluster.TelemetryEvent())

	return nil
}

func clusterFixedPriceAWS() float64 {
	eksPrice := aws.EKSPrices[*config.Cluster.Region]
	operatorInstancePrice := aws.InstanceMetadatas[*config.Cluster.Region]["t3.medium"].Price
	operatorEBSPrice := aws.EBSMetadatas[*config.Cluster.Region]["gp2"].PriceGB * 20 / 30 / 24
	nlbPrice := aws.NLBMetadatas[*config.Cluster.Region].Price
	natUnitPrice := aws.NATMetadatas[*config.Cluster.Region].Price

	var natTotalPrice float64
	if config.Cluster.NATGateway == clusterconfig.SingleNATGateway {
		natTotalPrice = natUnitPrice
	} else if config.Cluster.NATGateway == clusterconfig.HighlyAvailableNATGateway {
		natTotalPrice = natUnitPrice * float64(len(config.Cluster.AvailabilityZones))
	}

	return eksPrice + operatorInstancePrice + operatorEBSPrice + 2*nlbPrice + natTotalPrice
}

func InstanceTelemetryGCP() error {
	nodes, err := config.K8s.ListNodes(nil)
	if err != nil {
		return err
	}

	instanceInfos := make(map[string]*instanceInfo)
	var totalInstances int

	for _, node := range nodes {
		if node.Labels["workload"] != "true" {
			continue
		}

		instanceType := node.Labels["beta.kubernetes.io/instance-type"]
		if instanceType == "" {
			instanceType = "unknown"
		}

		totalInstances++

		instanceInfosKey := instanceType + "_ondemand"

		if info, ok := instanceInfos[instanceInfosKey]; ok {
			info.Count++
			continue
		}

		gpuQty := node.Status.Capacity["nvidia.com/gpu"]

		info := instanceInfo{
			InstanceType: instanceType,
			IsSpot:       false,
			Count:        1,
			Memory:       node.Status.Capacity.Memory().Value(),
			CPU:          float64(node.Status.Capacity.Cpu().MilliValue()) / 1000,
			GPU:          gpuQty.Value(),
			GPUType:      node.Labels["cloud.google.com/gke-accelerator"],
		}

		instanceInfos[instanceInfosKey] = &info
	}

	properties := map[string]interface{}{
		"zone":           *config.GCPCluster.Zone,
		"instance_count": totalInstances,
		"instances":      instanceInfos,
	}

	telemetry.Event("operator.cron", properties, config.GCPCluster.TelemetryEvent())

	return nil
}

func ErrorHandler(cronName string) func(error) {
	return func(err error) {
		err = errors.Wrap(err, cronName+" cron failed")
		telemetry.Error(err)
		operatorLogger.Error(err)
	}
}
