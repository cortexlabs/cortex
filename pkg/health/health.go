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

package health

import (
	"context"
	"fmt"
	"reflect"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kresource "k8s.io/apimachinery/pkg/api/resource"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmetrics "k8s.io/metrics/pkg/client/clientset/versioned"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterHealth represents the healthiness of each component of a cluster
type ClusterHealth struct {
	Operator             bool `json:"operator"`
	ControllerManager    bool `json:"controller_manager"`
	Prometheus           bool `json:"prometheus"`
	Autoscaler           bool `json:"autoscaler"`
	Activator            bool `json:"activator"`
	AsyncGateway         bool `json:"async_gateway"`
	Grafana              bool `json:"grafana"`
	OperatorGateway      bool `json:"operator_gateway"`
	APIsGateway          bool `json:"apis_gateway"`
	ClusterAutoscaler    bool `json:"cluster_autoscaler"`
	OperatorLoadBalancer bool `json:"operator_load_balancer"`
	APIsLoadBalancer     bool `json:"apis_load_balancer"`
	FluentBit            bool `json:"fluent_bit"`
	NodeExporter         bool `json:"node_exporter"`
	DCGMExporter         bool `json:"dcgm_exporter"`
	StatsDExporter       bool `json:"statsd_exporter"`
	EventExporter        bool `json:"event_exporter"`
	KubeStateMetrics     bool `json:"kube_state_metrics"`
}

func (c ClusterHealth) String() string {
	bytes, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}

	return string(bytes)
}

type ClusterWarnings struct {
	Prometheus string
}

// HasWarnings checks if ClusterWarnings has any warnings in its' fields
func (w ClusterWarnings) HasWarnings() bool {
	v := reflect.ValueOf(w)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).String() != "" {
			return true
		}
	}
	return false
}

// Check checks for the health of the different components of a cluster
func Check(awsClient *awslib.Client, k8sClient *k8s.Client, clusterName string) (ClusterHealth, error) {
	var (
		operatorHealth             bool
		controllerManagerHealth    bool
		prometheusHealth           bool
		autoscalerHealth           bool
		activatorHealth            bool
		asyncGatewayHealth         bool
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
			operatorHealth, err = getDeploymentReadiness(k8sClient, "operator", consts.DefaultNamespace)
			return err
		},
		func() error {
			var err error
			controllerManagerHealth, err = getDeploymentReadiness(k8sClient, "operator-controller-manager", consts.DefaultNamespace)
			return err
		},
		func() error {
			var err error
			prometheusHealth, err = getStatefulSetReadiness(k8sClient, "prometheus-prometheus", consts.PrometheusNamespace)
			return err
		},
		func() error {
			var err error
			autoscalerHealth, err = getDeploymentReadiness(k8sClient, "autoscaler", consts.DefaultNamespace)
			return err
		},
		func() error {
			var err error
			activatorHealth, err = getDeploymentReadiness(k8sClient, "activator", consts.DefaultNamespace)
			return err
		},
		func() error {
			var err error
			asyncGatewayHealth, err = getDeploymentReadiness(k8sClient, "async-gateway", consts.DefaultNamespace)
			return err
		},
		func() error {
			var err error
			grafanaHealth, err = getStatefulSetReadiness(k8sClient, "grafana", consts.DefaultNamespace)
			return err
		},
		func() error {
			var err error
			operatorGatewayHealth, err = getDeploymentReadiness(k8sClient, "ingressgateway-operator", consts.IstioNamespace)
			return err
		},
		func() error {
			var err error
			apisGatewayHealth, err = getDeploymentReadiness(k8sClient, "ingressgateway-apis", consts.IstioNamespace)
			return err
		},
		func() error {
			var err error
			clusterAutoscalerHealth, err = getDeploymentReadiness(k8sClient, "cluster-autoscaler", consts.KubeSystemNamespace)
			return err
		},
		func() error {
			var err error
			operatorLoadBalancerHealth, err = getLoadBalancerHealth(awsClient, clusterName, "operator", false)
			return err
		},
		func() error {
			var err error
			apisLoadBalancerHealth, err = getLoadBalancerHealth(awsClient, clusterName, "api", true)
			return err
		},
		func() error {
			var err error
			fluentBitHealth, err = getDaemonSetReadiness(k8sClient, "fluent-bit", consts.LoggingNamespace)
			return err
		},
		func() error {
			var err error
			dcgmExporterHealth, err = getDaemonSetReadiness(k8sClient, "dcgm-exporter", consts.PrometheusNamespace)
			return err
		},
		func() error {
			var err error
			nodeExporterHealth, err = getDaemonSetReadiness(k8sClient, "node-exporter", consts.PrometheusNamespace)
			return err
		},
		func() error {
			var err error
			statsdExporterHealth, err = getDeploymentReadiness(k8sClient, "prometheus-statsd-exporter", consts.PrometheusNamespace)
			return err
		},
		func() error {
			var err error
			eventExporterHealth, err = getDeploymentReadiness(k8sClient, "event-exporter", consts.LoggingNamespace)
			return err
		},
		func() error {
			var err error
			kubeStateMetricsHealth, err = getDeploymentReadiness(k8sClient, "kube-state-metrics", consts.PrometheusNamespace)
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
		AsyncGateway:         asyncGatewayHealth,
		Grafana:              grafanaHealth,
		OperatorGateway:      operatorGatewayHealth,
		APIsGateway:          apisGatewayHealth,
		ClusterAutoscaler:    clusterAutoscalerHealth,
		OperatorLoadBalancer: operatorLoadBalancerHealth,
		APIsLoadBalancer:     apisLoadBalancerHealth,
		FluentBit:            fluentBitHealth,
		NodeExporter:         nodeExporterHealth,
		DCGMExporter:         dcgmExporterHealth,
		StatsDExporter:       statsdExporterHealth,
		EventExporter:        eventExporterHealth,
		KubeStateMetrics:     kubeStateMetricsHealth,
	}, nil
}

func GetWarnings(k8sClient *k8s.Client) (ClusterWarnings, error) {
	var prometheusMemorySaturationWarn string

	saturation, err := getPodMemorySaturation(k8sClient, "prometheus-prometheus-0", consts.PrometheusNamespace)
	if err != nil {
		return ClusterWarnings{}, err
	}

	if saturation >= 0.7 {
		prometheusMemorySaturationWarn = fmt.Sprintf("memory usage is critically high (%.1f%%)", saturation*100)
	}

	return ClusterWarnings{
		Prometheus: prometheusMemorySaturationWarn,
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

func getLoadBalancerHealth(awsClient *awslib.Client, clusterName string, loadBalancerName string, testClassicLB bool) (bool, error) {
	loadBalancerV2, err := awsClient.FindLoadBalancerV2(map[string]string{
		clusterconfig.ClusterNameTag: clusterName,
		"cortex.dev/load-balancer":   loadBalancerName,
	})
	loadBalancerV2Exists := err == nil && loadBalancerV2 != nil

	if loadBalancerV2Exists {
		return aws.IsLoadBalancerV2Healthy(*loadBalancerV2), nil
	}

	if !testClassicLB {
		if err == nil {
			return false, errors.ErrorUnexpected(fmt.Sprintf("unable to locate %s nlb load balancer", loadBalancerName))
		}
		return false, errors.Wrap(err, fmt.Sprintf("unable to locate %s nlb load balancer", loadBalancerName))
	}

	loadBalancer, err := awsClient.FindLoadBalancer(map[string]string{
		clusterconfig.ClusterNameTag: clusterName,
		"cortex.dev/load-balancer":   loadBalancerName,
	})
	loadBalancerExists := err == nil && loadBalancer != nil
	if !loadBalancerExists {
		if err == nil {
			return false, errors.ErrorUnexpected(fmt.Sprintf("unable to locate %s elb load balancer", loadBalancerName))
		}
		return false, errors.Wrap(err, fmt.Sprintf("unable to locate %s elb load balancer", loadBalancerName))
	}
	healthy, err := awsClient.IsLoadBalancerHealthy(*loadBalancer.LoadBalancerName)
	if err != nil {
		return false, errors.Wrap(err, fmt.Sprintf("unable to check %s elb load balancer", loadBalancerName))
	}
	return healthy, nil
}

func getPodMemorySaturation(k8sClient *k8s.Client, podName, namespace string) (float64, error) {
	ctx := context.Background()
	var pod kcore.Pod
	if err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
		Namespace: namespace,
		Name:      podName,
	}, &pod); err != nil {
		return 0, err
	}

	metricsClient, err := kmetrics.NewForConfig(k8sClient.RestConfig)
	if err != nil {
		return 0, err
	}
	podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, podName, kmeta.GetOptions{})
	if err != nil {
		return 0, err
	}

	var totalMememoryUsage kresource.Quantity
	for _, container := range podMetrics.Containers {
		memory := container.Usage.Memory()
		if memory != nil {
			totalMememoryUsage.Add(*container.Usage.Memory())
		}
	}

	node, err := k8sClient.ClientSet().CoreV1().Nodes().Get(ctx, pod.Spec.NodeName, kmeta.GetOptions{})
	if err != nil {
		return 0, err
	}

	nodeMemory := node.Status.Allocatable.Memory()

	memRatio := totalMememoryUsage.AsApproximateFloat64() / nodeMemory.AsApproximateFloat64()

	return memRatio, nil
}
