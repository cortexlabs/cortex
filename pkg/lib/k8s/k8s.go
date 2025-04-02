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

package k8s

import (
	"path"
	"regexp"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/random"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	istionetworkingclient "istio.io/client-go/pkg/clientset/versioned/typed/networking/v1beta1"
	kresource "k8s.io/apimachinery/pkg/api/resource"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kclientdynamic "k8s.io/client-go/dynamic"
	kclientset "k8s.io/client-go/kubernetes"
	kclientapps "k8s.io/client-go/kubernetes/typed/apps/v1"
	kclientautoscaling "k8s.io/client-go/kubernetes/typed/autoscaling/v2"
	kclientbatch "k8s.io/client-go/kubernetes/typed/batch/v1"
	kclientcore "k8s.io/client-go/kubernetes/typed/core/v1"
	kclientextensions "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	kclientrest "k8s.io/client-go/rest"
	kclientcmd "k8s.io/client-go/tools/clientcmd"
	kclienthomedir "k8s.io/client-go/util/homedir"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	_home         = kclienthomedir.HomeDir()
	_deletePolicy = kmeta.DeletePropagationBackground
	_deleteOpts   = kmeta.DeleteOptions{
		PropagationPolicy: &_deletePolicy,
	}
)

type Client struct {
	ctrl.Client
	RestConfig           *kclientrest.Config
	clientSet            *kclientset.Clientset
	istioClientSet       *istioclient.Clientset
	dynamicClient        kclientdynamic.Interface
	podClient            kclientcore.PodInterface
	nodeClient           kclientcore.NodeInterface
	serviceClient        kclientcore.ServiceInterface
	configMapClient      kclientcore.ConfigMapInterface
	secretClient         kclientcore.SecretInterface
	deploymentClient     kclientapps.DeploymentInterface
	jobClient            kclientbatch.JobInterface
	ingressClient        kclientextensions.IngressInterface
	hpaClient            kclientautoscaling.HorizontalPodAutoscalerInterface
	virtualServiceClient istionetworkingclient.VirtualServiceInterface
	Namespace            string
}

func New(namespace string, inCluster bool, restConfig *kclientrest.Config, scheme *runtime.Scheme) (*Client, error) {
	var err error
	client := &Client{
		Namespace: namespace,
	}
	if restConfig != nil {
		client.RestConfig = restConfig
	} else if inCluster {
		client.RestConfig, err = kclientrest.InClusterConfig()
	} else {
		kubeConfig := path.Join(_home, ".kube", "config")
		client.RestConfig, err = kclientcmd.BuildConfigFromFlags("", kubeConfig)
	}

	if err != nil {
		return nil, errors.Wrap(err, "kubeconfig")
	}

	client.clientSet, err = kclientset.NewForConfig(client.RestConfig)
	if err != nil {
		return nil, errors.Wrap(err, "kubeconfig")
	}

	client.dynamicClient, err = kclientdynamic.NewForConfig(client.RestConfig)
	if err != nil {
		return nil, errors.Wrap(err, "kubeconfig")
	}

	client.Client, err = ctrl.New(client.RestConfig, ctrl.Options{Scheme: scheme})
	if err != nil {
		return nil, errors.Wrap(err, "kubeconfig")
	}

	client.istioClientSet, err = istioclient.NewForConfig(client.RestConfig)
	if err != nil {
		return nil, errors.Wrap(err, "kubeconfig")
	}
	client.virtualServiceClient = client.istioClientSet.NetworkingV1beta1().VirtualServices(namespace)

	client.podClient = client.clientSet.CoreV1().Pods(namespace)
	client.nodeClient = client.clientSet.CoreV1().Nodes()
	client.serviceClient = client.clientSet.CoreV1().Services(namespace)
	client.configMapClient = client.clientSet.CoreV1().ConfigMaps(namespace)
	client.secretClient = client.clientSet.CoreV1().Secrets(namespace)
	client.deploymentClient = client.clientSet.AppsV1().Deployments(namespace)
	client.jobClient = client.clientSet.BatchV1().Jobs(namespace)
	client.ingressClient = client.clientSet.ExtensionsV1beta1().Ingresses(namespace)
	client.hpaClient = client.clientSet.AutoscalingV2().HorizontalPodAutoscalers(namespace)
	return client, nil
}

func (c *Client) ClientSet() *kclientset.Clientset {
	return c.clientSet
}

func (c *Client) IstioClientSet() *istioclient.Clientset {
	return c.istioClientSet
}

// to be safe, k8s sometimes needs all characters to be lower case, and the first to be a letter
func RandomName() string {
	return random.LowercaseLetters(1) + random.LowercaseString(62)
}

// ValidName ensures name contains only lower case alphanumeric, '-', or '.'
func ValidName(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9\-\.]`)
	name = re.ReplaceAllLiteralString(name, "-")
	name = strings.ToLower(name)
	return name
}

// ValidNameContainer ensures name contains only lower case alphanumeric or '-', must start with alphabetic, end with alphanumeric
func ValidNameContainer(name string) string {
	name = ValidName(name)

	dots := regexp.MustCompile(`[\.]`)
	name = dots.ReplaceAllLiteralString(name, "-")

	leading := regexp.MustCompile(`^[^a-z]*`)
	name = leading.ReplaceAllLiteralString(name, "")

	trailing := regexp.MustCompile(`[^a-z0-9]*$`)
	name = trailing.ReplaceAllLiteralString(name, "")

	if len(name) == 0 {
		name = "x"
	}

	return name
}

func CPU(cpu string) kresource.Quantity {
	return kresource.MustParse(cpu)
}

func Mem(mem string) kresource.Quantity {
	return kresource.MustParse(mem)
}

func LabelExistsSelector(labelKeys ...string) string {
	if len(labelKeys) == 0 {
		return ""
	}

	return strings.Join(labelKeys, ",")
}

func FieldSelectorNotIn(key string, values []string) string {
	selectors := make([]string, len(values))
	for i, value := range values {
		selectors[i] = key + "!=" + value
	}
	return strings.Join(selectors, ",")
}
