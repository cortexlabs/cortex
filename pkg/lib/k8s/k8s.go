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

package k8s

import (
	"path"
	"regexp"
	"strings"

	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	tappsv1b1 "k8s.io/client-go/kubernetes/typed/apps/v1beta1"
	tbatchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	tcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	textensionsv1b1 "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

var (
	home         = homedir.HomeDir()
	deletePolicy = metav1.DeletePropagationBackground
	deleteOpts   = &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}
)

type Client struct {
	RestConfig       *rest.Config
	clientset        *kubernetes.Clientset
	podClient        tcorev1.PodInterface
	serviceClient    tcorev1.ServiceInterface
	deploymentClient tappsv1b1.DeploymentInterface
	jobClient        tbatchv1.JobInterface
	ingressClient    textensionsv1b1.IngressInterface
	Namespace        string
}

func New(namespace string, inCluster bool) (*Client, error) {
	var err error
	client := &Client{
		Namespace: namespace,
	}
	if inCluster {
		client.RestConfig, err = rest.InClusterConfig()
	} else {
		kubeConfig := path.Join(home, ".kube", "config")
		client.RestConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	}

	if err != nil {
		return nil, errors.Wrap(err, "kubeconfig")
	}

	client.clientset, err = kubernetes.NewForConfig(client.RestConfig)
	if err != nil {
		return nil, errors.Wrap(err, "kubeconfig")
	}

	client.podClient = client.clientset.CoreV1().Pods(namespace)
	client.serviceClient = client.clientset.CoreV1().Services(namespace)
	client.deploymentClient = client.clientset.AppsV1beta1().Deployments(namespace)
	client.jobClient = client.clientset.BatchV1().Jobs(namespace)
	client.ingressClient = client.clientset.ExtensionsV1beta1().Ingresses(namespace)
	return client, nil
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

func CPU(cpu string) k8sresource.Quantity {
	return k8sresource.MustParse(cpu)
}

func Mem(mem string) k8sresource.Quantity {
	return k8sresource.MustParse(mem)
}

func LabelSelector(labels map[string]string) string {
	if labels == nil {
		return ""
	}
	labelSelectorStr := ""
	for key, value := range labels {
		labelSelectorStr = labelSelectorStr + "," + key + "=" + value
	}
	return strings.TrimPrefix(labelSelectorStr, ",")
}

func FieldSelectorNotIn(key string, values []string) string {
	selectors := make([]string, len(values))
	for i, value := range values {
		selectors[i] = key + "!=" + value
	}
	return strings.Join(selectors, ",")
}
