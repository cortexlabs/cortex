/*
Copyright 2022 Cortex Labs, Inc.

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
	"context"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/logging"
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	v1 "k8s.io/api/core/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	telemetry.Event("operator.cron", properties,
		config.ClusterConfig.CoreConfig.TelemetryEvent(),
	)

	return nil
}

func clusterTelemetryProperties() (map[string]interface{}, error) {
	ctx := context.Background()
	var nodeList v1.NodeList

	err := config.K8s.List(ctx, &nodeList)
	if err != nil {
		return nil, err
	}
	nodes := nodeList.Items

	instanceInfos := make(map[string]*instanceInfo)

	var numOperatorInstances int
	var totalInstances int
	var totalInstancePrice float64
	var totalInstancePriceIfOnDemand float64

	spotPriceCache := make(map[string]float64) // instance type -> spot price

	for _, node := range nodes {
		if node.Labels["workload"] != "true" {
			if node.Labels["alpha.eksctl.io/nodegroup-name"] == "cx-operator" {
				numOperatorInstances++
			}
			continue
		}

		instanceType := node.Labels["node.kubernetes.io/instance-type"]
		if instanceType == "" {
			instanceType = "unknown"
		}

		isSpot := false
		if node.Labels["node-lifecycle"] == "spot" {
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

		onDemandPrice := aws.InstanceMetadatas[config.ClusterConfig.Region][instanceType].Price
		price := onDemandPrice
		if isSpot {
			if spotPrice, ok := spotPriceCache[instanceType]; ok {
				price = spotPrice
			} else {
				spotPrice, err := config.AWS.SpotInstancePrice(instanceType)
				if err == nil && spotPrice != 0 {
					price = spotPrice
					spotPriceCache[instanceType] = spotPrice
				} else {
					spotPriceCache[instanceType] = price // the request failed, so no need to try again in the future
				}
			}
		}

		ngName := node.Labels["alpha.eksctl.io/nodegroup-name"]
		ebsPricePerVolume := getEBSPriceForNodeGroupInstance(config.ClusterConfig.NodeGroups, ngName)
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

	fixedPrice := cortexSystemPrice(numOperatorInstances, 1)

	return map[string]interface{}{
		"region":                      config.ClusterConfig.Region,
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
			ebsPrice = aws.EBSMetadatas[config.ClusterConfig.Region][ng.InstanceVolumeType.String()].PriceGB * float64(ng.InstanceVolumeSize) / 30 / 24
			if ng.InstanceVolumeType == clusterconfig.IO1VolumeType && ng.InstanceVolumeIOPS != nil {
				ebsPrice += aws.EBSMetadatas[config.ClusterConfig.Region][ng.InstanceVolumeType.String()].PriceIOPS * float64(*ng.InstanceVolumeIOPS) / 30 / 24
			}
			if ng.InstanceVolumeType == clusterconfig.GP3VolumeType && ng.InstanceVolumeIOPS != nil && ng.InstanceVolumeThroughput != nil {
				ebsPrice += libmath.MaxFloat64(0, (aws.EBSMetadatas[config.ClusterConfig.Region][ng.InstanceVolumeType.String()].PriceIOPS-3000)*float64(*ng.InstanceVolumeIOPS)/30/24)
				ebsPrice += libmath.MaxFloat64(0, (aws.EBSMetadatas[config.ClusterConfig.Region][ng.InstanceVolumeType.String()].PriceThroughput-125)*float64(*ng.InstanceVolumeThroughput)/30/24)
			}
			break
		}
	}

	if ebsPrice == 0 && (ngName == "cx-operator" || ngName == "cx-prometheus") {
		return aws.EBSMetadatas[config.ClusterConfig.Region]["gp3"].PriceGB * 20 / 30 / 24
	}

	return ebsPrice
}

func cortexSystemPrice(numOperatorInstances, numPrometheusInstances int) float64 {
	eksPrice := aws.EKSPrices[config.ClusterConfig.Region]
	metricsEBSPrice := aws.EBSMetadatas[config.ClusterConfig.Region]["gp2"].PriceGB * (40 + 2) / 30 / 24
	nlbPrice := aws.NLBMetadatas[config.ClusterConfig.Region].Price
	natUnitPrice := aws.NATMetadatas[config.ClusterConfig.Region].Price
	var natTotalPrice float64

	if config.ClusterConfig.NATGateway == clusterconfig.SingleNATGateway {
		natTotalPrice = natUnitPrice
	} else if config.ClusterConfig.NATGateway == clusterconfig.HighlyAvailableNATGateway {
		natTotalPrice = natUnitPrice * float64(len(config.ClusterConfig.AvailabilityZones))
	}

	operatorInstancePrice := aws.InstanceMetadatas[config.ClusterConfig.Region]["t3.medium"].Price
	operatorEBSPrice := aws.EBSMetadatas[config.ClusterConfig.Region]["gp3"].PriceGB * 20 / 30 / 24

	prometheusInstancePrice := aws.InstanceMetadatas[config.ClusterConfig.Region][config.ClusterConfig.PrometheusInstanceType].Price
	prometheusEBSPrice := aws.EBSMetadatas[config.ClusterConfig.Region]["gp3"].PriceGB * 20 / 30 / 24

	fixedCosts := eksPrice + metricsEBSPrice + 2*nlbPrice + natTotalPrice +
		float64(numOperatorInstances)*(operatorInstancePrice+operatorEBSPrice) +
		float64(numPrometheusInstances)*(prometheusInstancePrice+prometheusEBSPrice)

	return fixedCosts
}

var clusterGauge = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "cortex_cluster_cost",
		Help: "The cost breakdown of the cortex cluster",
	}, []string{"api", "kind", "component"},
)

func CostBreakdown() error {
	ctx := context.Background()

	var nodeList v1.NodeList
	err := config.K8s.List(ctx, &nodeList)
	if err != nil {
		return err
	}
	nodes := nodeList.Items

	var podList v1.PodList
	err = config.K8s.List(ctx, &podList,
		client.InNamespace(consts.DefaultNamespace),
		client.HasLabels{"apiName", "apiKind"},
	)
	if err != nil {
		return err
	}
	pods := podList.Items

	spotPriceCache := make(map[string]float64) // instance type -> spot price

	// Total cluster costs = cortex system + cortex workloads
	var totalClusterCosts float64 = cortexSystemPrice(0, 0)
	// Total cortex system costs = operator + prometheus node groups + workload daemonsets
	var totalCortexSystemCosts float64 = cortexSystemPrice(0, 0)
	// Total workload compute costs by api name
	totalWorkloadComputeCostsByAPIName := make(map[string]float64, 0)
	// Total workload compute costs by kind
	totalWorkloadComputeCostsByAPIKind := make(map[string]float64, 0)

	for _, node := range nodes {
		instanceType := node.Labels["node.kubernetes.io/instance-type"]
		if instanceType == "" {
			continue
		}

		workloadNode := false
		if node.Labels["workload"] == "true" {
			workloadNode = true
		}

		isSpot := false
		if node.Labels["node-lifecycle"] == "spot" {
			isSpot = true
		}

		var instanceComputePrice float64
		if isSpot {
			if spotPrice, ok := spotPriceCache[instanceType]; ok {
				instanceComputePrice = spotPrice
			} else {
				spotPrice, err := config.AWS.SpotInstancePrice(instanceType)
				if err == nil && spotPrice != 0 {
					instanceComputePrice = spotPrice
					spotPriceCache[instanceType] = spotPrice
				} else {
					instanceComputePrice = aws.InstanceMetadatas[config.ClusterConfig.Region][instanceType].Price // the request failed, so no need to try again in the future
					spotPriceCache[instanceType] = instanceComputePrice
				}
			}
		} else {
			instanceComputePrice = aws.InstanceMetadatas[config.ClusterConfig.Region][instanceType].Price
		}

		ngName := node.Labels["alpha.eksctl.io/nodegroup-name"]
		instanceEBSPrice := getEBSPriceForNodeGroupInstance(config.ClusterConfig.NodeGroups, ngName)

		instancePrice := instanceComputePrice + instanceEBSPrice

		totalClusterCosts += instancePrice
		if !workloadNode {
			totalCortexSystemCosts += instancePrice
			continue
		}

		for _, pod := range pods {
			if pod.Spec.NodeName != node.Name {
				continue
			}
			maxPods := k8s.HowManyPodsFitOnNode(pod.Spec, node, consts.CortexCPUPodReserved, consts.CortexMemPodReserved)
			costPerPod := instancePrice / float64(maxPods)

			if apiName, ok := pod.Labels["apiName"]; ok {
				if _, okMap := totalWorkloadComputeCostsByAPIName[apiName]; okMap {
					totalWorkloadComputeCostsByAPIName[apiName] += costPerPod
				} else {
					totalWorkloadComputeCostsByAPIName[apiName] = costPerPod
				}
			}

			if apiKind, ok := pod.Labels["apiKind"]; ok {
				if _, okMap := totalWorkloadComputeCostsByAPIName[apiKind]; okMap {
					totalWorkloadComputeCostsByAPIKind[apiKind] += costPerPod
				} else {
					totalWorkloadComputeCostsByAPIKind[apiKind] = costPerPod
				}
			}
		}
	}

	totalWorkloadComputeCosts := totalClusterCosts - totalCortexSystemCosts
	var totalWorkloadComputeCostsFromAPIName float64
	var totalWorkloadComputeCostsFromAPIKind float64
	apiNameRenormalizationRatio := float64(1)
	apiKindRenormalizationRatio := float64(1)

	for apiName := range totalWorkloadComputeCostsByAPIName {
		totalWorkloadComputeCostsFromAPIName += totalWorkloadComputeCostsByAPIName[apiName]
	}
	if totalWorkloadComputeCostsFromAPIName > 0 {
		apiNameRenormalizationRatio = totalWorkloadComputeCosts / totalWorkloadComputeCostsFromAPIName
	}
	for apiKind := range totalWorkloadComputeCostsByAPIKind {
		totalWorkloadComputeCostsFromAPIKind += totalWorkloadComputeCostsByAPIKind[apiKind]
	}
	if totalWorkloadComputeCostsFromAPIKind > 0 {
		apiKindRenormalizationRatio = totalWorkloadComputeCosts / totalWorkloadComputeCostsFromAPIKind
	}

	clusterGauge.Reset()
	clusterGauge.WithLabelValues("false", "false", "cluster-costs").Set(totalClusterCosts)
	clusterGauge.WithLabelValues("false", "false", "cortex-system-costs").Set(totalCortexSystemCosts)
	for apiName := range totalWorkloadComputeCostsByAPIName {
		clusterGauge.WithLabelValues("true", "false", apiName).Set(totalWorkloadComputeCostsByAPIName[apiName] * apiNameRenormalizationRatio)
	}
	for apiKind := range totalWorkloadComputeCostsByAPIKind {
		clusterGauge.WithLabelValues("false", "true", apiKind).Set(totalWorkloadComputeCostsByAPIKind[apiKind] * apiKindRenormalizationRatio)
	}

	return nil
}

func ErrorHandler(cronName string) func(error) {
	return func(err error) {
		err = errors.Wrap(err, cronName+" cron failed")
		telemetry.Error(err)
		operatorLogger.Error(err)
	}
}
