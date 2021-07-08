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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterHealth represents the healthiness of each component of a cluster
type ClusterHealth struct {
	Operator                 bool `json:"operator"`
	ControllerManager        bool `json:"controller_manager"`
	Prometheus               bool `json:"prometheus"`
	Autoscaler               bool `json:"autoscaler"`
	Activator                bool `json:"activator"`
	Grafana                  bool `json:"grafana"`
	OperatorGateway          bool `json:"operator_gateway"`
	APIsGateway              bool `json:"apis_gateway"`
	ClusterAutoscaler        bool `json:"cluster_autoscaler"`
	OperatorLoadBalancer     bool `json:"operator_load_balancer"`
	APIsLoadBalancer         bool `json:"apis_load_balancer"`
	FluentBit                bool `json:"fluent_bit"`
	NodeExporter             bool `json:"node_exporter"`
	DCGMExporter             bool `json:"dcgm_exporter"`
	StatsDExporter           bool `json:"statsd_exporter"`
	EventExporter            bool `json:"event_exporter"`
	KubeStateMetricsExporter bool `json:"kube_state_metrics_exporter"`
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
		fluentBitHealth            bool
		nodeExporterHealth         bool
		dcgmExporterHealth         bool
		statsdExporterHealth       bool
		eventExporterHealth        bool
		kubeStateMetricsHealth     bool
	)

	if err := parallel.RunFirstErr(
		func() error {
			var err error
			operatorHealth, err = getDeploymentReadiness(k8sClient, "operator", "default")
			return err
		},
		func() error {
			var err error
			controllerManagerHealth, err = getDeploymentReadiness(k8sClient, "operator-controller-manager", "default")
			return err
		},
		func() error {
			var err error
			prometheusHealth, err = getStatefulSetReadiness(k8sClient, "prometheus-prometheus", "default")
			return err
		},
		func() error {
			var err error
			autoscalerHealth, err = getDeploymentReadiness(k8sClient, "autoscaler", "default")
			return err
		},
		func() error {
			var err error
			activatorHealth, err = getDeploymentReadiness(k8sClient, "activator", "default")
			return err
		},
		func() error {
			var err error
			grafanaHealth, err = getStatefulSetReadiness(k8sClient, "grafana", "default")
			return err
		},
		func() error {
			var err error
			operatorGatewayHealth, err = getDeploymentReadiness(k8sClient, "ingressgateway-operator", "istio-system")
			return err
		},
		func() error {
			var err error
			apisGatewayHealth, err = getDeploymentReadiness(k8sClient, "ingressgateway-apis", "istio-system")
			return err
		},
		func() error {
			var err error
			clusterAutoscalerHealth, err = getDeploymentReadiness(k8sClient, "cluster-autoscaler", "kube-system")
			return err
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
		func() error {
			var err error
			fluentBitHealth, err = getDaemonSetReadiness(k8sClient, "fluent-bit", "default")
			return err
		},
		func() error {
			var err error
			dcgmExporterHealth, err = getDaemonSetReadiness(k8sClient, "dcgm-exporter", "default")
			return err
		},
		func() error {
			var err error
			nodeExporterHealth, err = getDaemonSetReadiness(k8sClient, "node-exporter", "default")
			return err
		},
		func() error {
			var err error
			statsdExporterHealth, err = getDeploymentReadiness(k8sClient, "prometheus-statsd-exporter", "default")
			return err
		},
		func() error {
			var err error
			eventExporterHealth, err = getDeploymentReadiness(k8sClient, "event-exporter", "default")
			return err
		},
		func() error {
			var err error
			kubeStateMetricsHealth, err = getDeploymentReadiness(k8sClient, "kube-state-metrics", "default")
			return err
		},
	); err != nil {
		return ClusterHealth{}, err
	}

	return ClusterHealth{
		Operator:                 operatorHealth,
		ControllerManager:        controllerManagerHealth,
		Prometheus:               prometheusHealth,
		Autoscaler:               autoscalerHealth,
		Activator:                activatorHealth,
		Grafana:                  grafanaHealth,
		OperatorGateway:          operatorGatewayHealth,
		APIsGateway:              apisGatewayHealth,
		ClusterAutoscaler:        clusterAutoscalerHealth,
		OperatorLoadBalancer:     operatorLoadBalancerHealth,
		APIsLoadBalancer:         apisLoadBalancerHealth,
		FluentBit:                fluentBitHealth,
		NodeExporter:             nodeExporterHealth,
		DCGMExporter:             dcgmExporterHealth,
		StatsDExporter:           statsdExporterHealth,
		EventExporter:            eventExporterHealth,
		KubeStateMetricsExporter: kubeStateMetricsHealth,
	}, nil
}

func getDeploymentReadiness(k8sClient *k8s.Client, name, namespace string) (bool, error) {
	ctx := context.Background()
	var deployment kapps.Deployment
	if err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &deployment); err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return deployment.Status.ReadyReplicas > 0, nil
}

func getStatefulSetReadiness(k8sClient *k8s.Client, name, namespace string) (bool, error) {
	ctx := context.Background()
	var statefulSet kapps.StatefulSet
	if err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &statefulSet); err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return statefulSet.Status.ReadyReplicas > 0, nil
}

func getDaemonSetReadiness(k8sClient *k8s.Client, name, namespace string) (bool, error) {
	ctx := context.Background()
	var daemonSet kapps.DaemonSet
	if err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &daemonSet); err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return daemonSet.Status.NumberReady == daemonSet.Status.CurrentNumberScheduled, nil
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
