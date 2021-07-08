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

package health

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/service/elbv2"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	kapps "k8s.io/api/apps/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterHealth represents the healthiness of each component of a cluster
type ClusterHealth struct {
	Operator             bool `json:"operator"`
	ControllerManager    bool `json:"controller_manager"`
	Prometheus           bool `json:"prometheus"`
	Autoscaler           bool `json:"autoscaler"`
	Activator            bool `json:"activator"`
	Grafana              bool `json:"grafana"`
	OperatorGateway      bool `json:"operator_gateway"`
	APIsGateway          bool `json:"apis_gateway"`
	ClusterAutoscaler    bool `json:"cluster_autoscaler"`
	OperatorLoadBalancer bool `json:"operator_load_balancer"`
	APIsLoadBalancer     bool `json:"apis_load_balancer"`
}

func (c ClusterHealth) String() string {
	bytes, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}

	return string(bytes)
}

// Check checks for the health of the different components of a cluster
func Check(awsClient *awslib.Client, k8sClient *k8s.Client, clusterName string) (ClusterHealth, error) {
	var (
		operatorHealth             bool
		controllerManagerHealth    bool
		prometheusHealth           bool
		autoscalerHealth           bool
		activatorHealth            bool
		grafanaHealth              bool
		operatorGatewayHealth      bool
		apisGatewayHealth          bool
		clusterAutoscalerHealth    bool
		operatorLoadBalancerHealth bool
		apisLoadBalancerHealth     bool
	)

	ctx := context.Background()
	if err := parallel.RunFirstErr(
		func() error {
			var deployment kapps.Deployment
			if err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
				Namespace: "default",
				Name:      "operator",
			}, &deployment); err != nil {
				return err
			}
			operatorHealth = deployment.Status.ReadyReplicas > 0
			return nil
		},
		func() error {
			var deployment kapps.Deployment
			if err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
				Namespace: "default",
				Name:      "operator-controller-manager",
			}, &deployment); err != nil {
				return err
			}
			controllerManagerHealth = deployment.Status.ReadyReplicas > 0
			return nil
		},
		func() error {
			var statefulSet kapps.StatefulSet
			if err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
				Namespace: "default",
				Name:      "prometheus-prometheus",
			}, &statefulSet); err != nil {
				return err
			}
			prometheusHealth = statefulSet.Status.ReadyReplicas > 0
			return nil
		},
		func() error {
			var deployment kapps.Deployment
			if err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
				Namespace: "default",
				Name:      "autoscaler",
			}, &deployment); err != nil {
				return err
			}
			autoscalerHealth = deployment.Status.ReadyReplicas > 0
			return nil
		},
		func() error {
			var deployment kapps.Deployment
			if err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
				Namespace: "default",
				Name:      "activator",
			}, &deployment); err != nil {
				return err
			}
			activatorHealth = deployment.Status.ReadyReplicas > 0
			return nil
		},
		func() error {
			var statefulSet kapps.StatefulSet
			if err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
				Namespace: "default",
				Name:      "grafana",
			}, &statefulSet); err != nil {
				return err
			}
			grafanaHealth = statefulSet.Status.ReadyReplicas > 0
			return nil
		},
		func() error {
			var deployment kapps.Deployment
			if err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
				Namespace: "istio-system",
				Name:      "ingressgateway-operator",
			}, &deployment); err != nil {
				return err
			}
			operatorGatewayHealth = deployment.Status.ReadyReplicas > 0
			return nil
		},
		func() error {
			var deployment kapps.Deployment
			if err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
				Namespace: "istio-system",
				Name:      "ingressgateway-apis",
			}, &deployment); err != nil {
				return err
			}
			apisGatewayHealth = deployment.Status.ReadyReplicas > 0
			return nil
		},
		func() error {
			var deployment kapps.Deployment
			if err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
				Namespace: "kube-system",
				Name:      "cluster-autoscaler",
			}, &deployment); err != nil {
				return err
			}
			clusterAutoscalerHealth = deployment.Status.ReadyReplicas > 0
			return nil
		},
		func() error {
			var err error
			operatorLoadBalancerHealth, err = getLoadBalancerHealth(awsClient, clusterName, "operator")
			return err
		},
		func() error {
			var err error
			apisLoadBalancerHealth, err = getLoadBalancerHealth(awsClient, clusterName, "api")
			return err
		},
	); err != nil {
		return ClusterHealth{}, err
	}

	return ClusterHealth{
		Operator:             operatorHealth,
		ControllerManager:    controllerManagerHealth,
		Prometheus:           prometheusHealth,
		Autoscaler:           autoscalerHealth,
		Activator:            activatorHealth,
		Grafana:              grafanaHealth,
		OperatorGateway:      operatorGatewayHealth,
		APIsGateway:          apisGatewayHealth,
		ClusterAutoscaler:    clusterAutoscalerHealth,
		OperatorLoadBalancer: operatorLoadBalancerHealth,
		APIsLoadBalancer:     apisLoadBalancerHealth,
	}, nil
}

func getLoadBalancerHealth(awsClient *awslib.Client, clusterName string, loadBalancerName string) (bool, error) {
	loadBalancer, err := awsClient.FindLoadBalancer(map[string]string{
		clusterconfig.ClusterNameTag: clusterName,
		"cortex.dev/load-balancer":   loadBalancerName,
	})
	if err != nil {
		return false, errors.Wrap(err, fmt.Sprintf("unable to locate %s load balancer", loadBalancerName))
	}

	if loadBalancer.State == nil || loadBalancer.State.Code == nil {
		return false, nil
	}

	return *loadBalancer.State.Code == elbv2.LoadBalancerStateEnumActive, nil
}
