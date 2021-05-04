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
	"github.com/cortexlabs/cortex/pkg/lib/logging"
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var operatorLogger = logging.GetLogger()
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
}

func ClusterTelemetry() error {
	properties, err := clusterTelemetryProperties()
	if err != nil {
		return err
	}
	telemetry.Event("operator.cron", properties, config.CoreConfig.TelemetryEvent(), config.ManagedConfig.TelemetryEvent())

	return nil
}

func clusterTelemetryProperties() (map[string]interface{}, error) {
	nodes, err := config.K8s.ListNodes(nil)
	if err != nil {
		return nil, err
	}

	instanceInfos := make(map[string]*instanceInfo)

	var totalInstances int
	var totalInstancePrice float64
	var totalInstancePriceIfOnDemand float64

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

		onDemandPrice := aws.InstanceMetadatas[config.CoreConfig.Region][instanceType].Price
		price := onDemandPrice
		if isSpot {
			spotPrice, err := config.AWS.SpotInstancePrice(instanceType)
			if err == nil && spotPrice != 0 {
				price = spotPrice
			}
		}

		ngName := node.Labels["alpha.eksctl.io/nodegroup-name"]
		ebsPricePerVolume := getEBSPriceForNodeGroupInstance(config.ManagedConfig.NodeGroups, ngName)
		onDemandPrice += ebsPricePerVolume
		price += ebsPricePerVolume

		gpuQty := node.Status.Capacity["nvidia.com/gpu"]
		infQty := node.Status.Capacity["aws.amazon.com/neuron"]

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
		}

		instanceInfos[instanceInfosKey] = &info
		totalInstancePrice += info.Price
		totalInstancePriceIfOnDemand += info.OnDemandPrice
	}

	fixedPrice := clusterFixedPrice()

	return map[string]interface{}{
		"region":                      config.CoreConfig.Region,
		"instance_count":              totalInstances,
		"instances":                   instanceInfos,
		"fixed_price":                 fixedPrice,
		"workload_price":              totalInstancePrice,
		"workload_price_if_on_demand": totalInstancePriceIfOnDemand,
		"total_price":                 totalInstancePrice + fixedPrice,
		"total_price_if_on_demand":    totalInstancePriceIfOnDemand + fixedPrice,
	}, nil
}

func getEBSPriceForNodeGroupInstance(ngs []*clusterconfig.NodeGroup, ngName string) float64 {
	var ebsPrice float64
	for _, ng := range ngs {
		var ngNamePrefix string
		if ng.Spot {
			ngNamePrefix = "cx-ws-"
		} else {
			ngNamePrefix = "cx-wd-"
		}
		if ng.Name == ngNamePrefix+ngName {
			ebsPrice = aws.EBSMetadatas[config.CoreConfig.Region][ng.InstanceVolumeType.String()].PriceGB * float64(ng.InstanceVolumeSize) / 30 / 24
			if ng.InstanceVolumeType == clusterconfig.IO1VolumeType && ng.InstanceVolumeIOPS != nil {
				ebsPrice += aws.EBSMetadatas[config.CoreConfig.Region][ng.InstanceVolumeType.String()].PriceIOPS * float64(*ng.InstanceVolumeIOPS) / 30 / 24
			}
			if ng.InstanceVolumeType == clusterconfig.GP3VolumeType && ng.InstanceVolumeIOPS != nil && ng.InstanceVolumeThroughput != nil {
				ebsPrice += libmath.MaxFloat64(0, (aws.EBSMetadatas[config.CoreConfig.Region][ng.InstanceVolumeType.String()].PriceIOPS-3000)*float64(*ng.InstanceVolumeIOPS)/30/24)
				ebsPrice += libmath.MaxFloat64(0, (aws.EBSMetadatas[config.CoreConfig.Region][ng.InstanceVolumeType.String()].PriceThroughput-125)*float64(*ng.InstanceVolumeThroughput)/30/24)
			}
			break
		}
	}
	return ebsPrice
}

func clusterFixedPrice() float64 {
	eksPrice := aws.EKSPrices[config.CoreConfig.Region]
	operatorInstancePrice := aws.InstanceMetadatas[config.CoreConfig.Region]["t3.medium"].Price
	operatorEBSPrice := aws.EBSMetadatas[config.CoreConfig.Region]["gp3"].PriceGB * 20 / 30 / 24
	metricsEBSPrice := aws.EBSMetadatas[config.CoreConfig.Region]["gp2"].PriceGB * (40 + 2) / 30 / 24
	nlbPrice := aws.NLBMetadatas[config.CoreConfig.Region].Price
	natUnitPrice := aws.NATMetadatas[config.CoreConfig.Region].Price
	var natTotalPrice float64

	if config.ManagedConfig.NATGateway == clusterconfig.SingleNATGateway {
		natTotalPrice = natUnitPrice
	} else if config.ManagedConfig.NATGateway == clusterconfig.HighlyAvailableNATGateway {
		natTotalPrice = natUnitPrice * float64(len(config.ManagedConfig.AvailabilityZones))
	}

	return eksPrice + 2*(operatorInstancePrice+operatorEBSPrice) + metricsEBSPrice + 2*nlbPrice + natTotalPrice
}

func ErrorHandler(cronName string) func(error) {
	return func(err error) {
		err = errors.Wrap(err, cronName+" cron failed")
		telemetry.Error(err)
		operatorLogger.Error(err)
	}
}
