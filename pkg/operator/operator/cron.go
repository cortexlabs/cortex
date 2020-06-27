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

package operator

import (
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DeleteEvictedPods() error {
	failedPods, err := config.K8s.ListPods(&kmeta.ListOptions{
		FieldSelector: "status.phase=Failed",
	})
	if err != nil {
		return err
	}

	var errs []error
	for _, pod := range failedPods {
		if pod.Status.Reason == k8s.ReasonEvicted {
			_, err := config.K8s.DeletePod(pod.Name)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

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
}

func InstanceTelemetry() error {
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
			spotPrice, err := config.AWS.SpotInstancePrice(*config.Cluster.Region, instanceType)
			if err == nil && spotPrice != 0 {
				price = spotPrice
			}
		}

		info := instanceInfo{
			InstanceType:  instanceType,
			IsSpot:        isSpot,
			Price:         price,
			OnDemandPrice: onDemandPrice,
			Count:         1,
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

	fixedPrice := clusterFixedPrice()

	properties := map[string]interface{}{
		"region":                   *config.Cluster.Region,
		"instance_count":           totalInstances,
		"instances":                instanceInfos,
		"fixed_price":              fixedPrice,
		"total_price":              totalInstancePrice + fixedPrice,
		"total_price_if_on_demand": totalInstancePriceIfOnDemand + fixedPrice,
	}

	telemetry.Event("operator.cron", properties)

	return nil
}

func clusterFixedPrice() float64 {
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

func ErrorHandler(cronName string) func(error) {
	return func(err error) {
		err = errors.Wrap(err, cronName+" cron failed")
		telemetry.Error(err)
		errors.PrintError(err)
	}
}
