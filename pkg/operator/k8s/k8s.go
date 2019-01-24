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
	kubernetes "k8s.io/client-go/kubernetes"
	tappsv1b1 "k8s.io/client-go/kubernetes/typed/apps/v1beta1"
	tbatchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	tcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	textensionsv1b1 "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	rest "k8s.io/client-go/rest"
	clientcmd "k8s.io/client-go/tools/clientcmd"
	homedir "k8s.io/client-go/util/homedir"

	cc "github.com/cortexlabs/cortex/pkg/operator/cortexconfig"
	cr "github.com/cortexlabs/cortex/pkg/utils/configreader"

	"github.com/cortexlabs/cortex/pkg/utils/errors"
)

var (
	home      = homedir.HomeDir()
	Config    *rest.Config
	clientset *kubernetes.Clientset

	podClient        tcorev1.PodInterface
	serviceClient    tcorev1.ServiceInterface
	deploymentClient tappsv1b1.DeploymentInterface
	jobClient        tbatchv1.JobInterface
	ingressClient    textensionsv1b1.IngressInterface
)

var deletePolicy = metav1.DeletePropagationBackground
var deleteOpts = &metav1.DeleteOptions{
	PropagationPolicy: &deletePolicy,
}

func init() {
	var err error

	operatorInCluster := cr.MustBoolFromEnv("CONST_OPERATOR_IN_CLUSTER", &cr.BoolValidation{Default: true})
	if operatorInCluster {
		Config, err = rest.InClusterConfig()
	} else {
		kubeConfig := path.Join(home, ".kube", "config")
		Config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	}

	if err != nil {
		errors.Exit(err, "kubeconfig")
	}

	clientset, err = kubernetes.NewForConfig(Config)
	if err != nil {
		errors.Exit(err, "kubeconfig")
	}

	podClient = clientset.CoreV1().Pods(cc.Namespace)
	serviceClient = clientset.CoreV1().Services(cc.Namespace)
	deploymentClient = clientset.AppsV1beta1().Deployments(cc.Namespace)
	jobClient = clientset.BatchV1().Jobs(cc.Namespace)
	ingressClient = clientset.ExtensionsV1beta1().Ingresses(cc.Namespace)
}

// Lower case alphanumeric, '-', or '.'
func ValidName(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9\-\.]`)
	name = re.ReplaceAllLiteralString(name, "-")
	name = strings.ToLower(name)
	return name
}

// Lower case alphanumeric or '-', must start with alphabetic, end with alphanumeric
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

func Cpu(cpu string) k8sresource.Quantity {
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
