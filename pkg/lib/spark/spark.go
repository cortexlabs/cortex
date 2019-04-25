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

package spark

import (
	"strings"

	sparkop "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1alpha1"
	clientset "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/client/clientset/versioned"
	clientsettyped "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/client/clientset/versioned/typed/sparkoperator.k8s.io/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
)

type Client struct {
	sparkClientset clientset.Interface
	sparkClient    clientsettyped.SparkApplicationInterface
}

var (
	doneStates = []string{
		string(sparkop.CompletedState),
		string(sparkop.FailedState),
		string(sparkop.FailedSubmissionState),
		string(sparkop.UnknownState),
	}

	runningStates = []string{
		string(sparkop.NewState),
		string(sparkop.SubmittedState),
		string(sparkop.RunningState),
	}

	successStates = []string{
		string(sparkop.CompletedState),
	}

	failureStates = []string{
		string(sparkop.FailedState),
		string(sparkop.FailedSubmissionState),
		string(sparkop.UnknownState),
	}

	SuccessCondition = "status.applicationState.state in (" + strings.Join(successStates, ",") + ")"
	FailureCondition = "status.applicationState.state in (" + strings.Join(failureStates, ",") + ")"
)

func New(restConfig *rest.Config, namespace string) (*Client, error) {
	var err error
	client := &Client{}
	client.sparkClientset, err = clientset.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "spark", "kubeconfig")
	}

	client.sparkClient = client.sparkClientset.SparkoperatorV1alpha1().SparkApplications(namespace)
	return client, nil
}

func (c *Client) List(opts *metav1.ListOptions) ([]sparkop.SparkApplication, error) {
	if opts == nil {
		opts = &metav1.ListOptions{}
	}
	sparkList, err := c.sparkClient.List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return sparkList.Items, nil
}

func (c *Client) ListByLabels(labels map[string]string) ([]sparkop.SparkApplication, error) {
	opts := &metav1.ListOptions{
		LabelSelector: k8s.LabelSelector(labels),
	}
	return c.List(opts)
}

func (c *Client) ListByLabel(labelKey string, labelValue string) ([]sparkop.SparkApplication, error) {
	return c.ListByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) Delete(appName string) (bool, error) {
	err := c.sparkClient.Delete(appName, &metav1.DeleteOptions{})
	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func IsDone(sparkApp *sparkop.SparkApplication) bool {
	return slices.HasString(doneStates, string(sparkApp.Status.AppState.State))
}
