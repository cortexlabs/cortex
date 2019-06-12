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

package argo

import (
	"strings"
	"time"

	awfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	twfv1 "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"k8s.io/client-go/rest"
)

var (
	doneStates = []string{
		string(awfv1.NodeSucceeded),
		string(awfv1.NodeSkipped),
		string(awfv1.NodeFailed),
		string(awfv1.NodeError),
	}
	runningStates = []string{
		string(awfv1.NodeRunning),
	}
)

type Client struct {
	workflowClient twfv1.WorkflowInterface
	namespace      string
}

func New(restConfig *rest.Config, namespace string) *Client {
	client := &Client{
		namespace: namespace,
	}
	wfcs := wfclientset.NewForConfigOrDie(restConfig)
	client.workflowClient = wfcs.ArgoprojV1alpha1().Workflows(namespace)
	return client
}

type WorkflowTask struct {
	Name             string
	Action           string
	Manifest         string
	SuccessCondition string
	FailureCondition string
	Dependencies     []string
	Labels           map[string]string
}

func (c *Client) NewWorkflow(name string, labels ...map[string]string) *awfv1.Workflow {
	name = "argo-" + name
	if !strings.HasSuffix(name, "-") && !strings.HasSuffix(name, "_") {
		name = name + "-"
	}
	allLabels := maps.MergeStrMaps(labels...)

	return &awfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name,
			Namespace:    c.namespace,
			Labels:       allLabels,
		},
		Spec: awfv1.WorkflowSpec{
			ServiceAccountName: "argo-executor",
			Entrypoint:         "DAG",
			Templates: []awfv1.Template{
				{
					Name: "DAG",
					DAG: &awfv1.DAGTemplate{
						Tasks: []awfv1.DAGTask{},
					},
				},
			},
		},
	}
}

func AddTask(wf *awfv1.Workflow, task *WorkflowTask) *awfv1.Workflow {
	if task == nil {
		return wf
	}

	DAGTask := awfv1.DAGTask{
		Name:         task.Name,
		Template:     task.Name,
		Dependencies: slices.RemoveEmptiesAndUnique(task.Dependencies),
	}

	// All tasks are added to the DAG template which is first
	wf.Spec.Templates[0].DAG.Tasks = append(wf.Spec.Templates[0].DAG.Tasks, DAGTask)

	labels := task.Labels
	labels["argo"] = "true"

	template := awfv1.Template{
		Name: task.Name,
		Resource: &awfv1.ResourceTemplate{
			Action:           task.Action,
			Manifest:         task.Manifest,
			SuccessCondition: task.SuccessCondition,
			FailureCondition: task.FailureCondition,
		},
		Metadata: awfv1.Metadata{
			Labels: labels,
		},
	}

	wf.Spec.Templates = append(wf.Spec.Templates, template)

	return wf
}

func EnableGC(spec metav1.Object) {
	ownerReferences := spec.GetOwnerReferences()
	ownerReferences = append(ownerReferences, metav1.OwnerReference{
		APIVersion:         "argoproj.io/v1alpha1",
		Kind:               "Workflow",
		Name:               "{{workflow.name}}",
		UID:                "{{workflow.uid}}",
		BlockOwnerDeletion: pointer.Bool(false),
	})
	spec.SetOwnerReferences(ownerReferences)
}

func NumTasks(wf *awfv1.Workflow) int {
	if wf == nil || len(wf.Spec.Templates) == 0 {
		return 0
	}
	return len(wf.Spec.Templates[0].DAG.Tasks)
}

func (c *Client) Run(wf *awfv1.Workflow) error {
	_, err := c.workflowClient.Create(wf)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (c *Client) List(opts *metav1.ListOptions) ([]awfv1.Workflow, error) {
	if opts == nil {
		opts = &metav1.ListOptions{}
	}
	wfList, err := c.workflowClient.List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return wfList.Items, nil
}

func (c *Client) ListByLabels(labels map[string]string) ([]awfv1.Workflow, error) {
	opts := &metav1.ListOptions{
		LabelSelector: k8s.LabelSelector(labels),
	}
	return c.List(opts)
}

func (c *Client) ListByLabel(labelKey string, labelValue string) ([]awfv1.Workflow, error) {
	return c.ListByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) ListRunning(labels ...map[string]string) ([]awfv1.Workflow, error) {
	wfs, err := c.ListByLabels(maps.MergeStrMaps(labels...))
	if err != nil {
		return wfs, err
	}
	runningWfs := []awfv1.Workflow{}
	for _, wf := range wfs {
		if IsRunning(&wf) {
			runningWfs = append(runningWfs, wf)
		}
	}
	return runningWfs, nil
}

func (c *Client) ListDone(labels ...map[string]string) ([]awfv1.Workflow, error) {
	wfs, err := c.ListByLabels(maps.MergeStrMaps(labels...))
	if err != nil {
		return wfs, err
	}
	doneWfs := []awfv1.Workflow{}
	for _, wf := range wfs {
		if IsDone(&wf) {
			doneWfs = append(doneWfs, wf)
		}
	}
	return doneWfs, nil
}

func (c *Client) Delete(wfName string) (bool, error) {
	err := c.workflowClient.Delete(wfName, &metav1.DeleteOptions{})
	if k8serrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) DeleteMultiple(wfs []awfv1.Workflow) error {
	errs := []error{}
	for _, wf := range wfs {
		_, err := c.Delete(wf.Name)
		errs = append(errs, err)
	}
	return errors.FirstError(errs...)
}

func IsDone(wf *awfv1.Workflow) bool {
	if wf == nil {
		return true
	}
	return slices.HasString(doneStates, string(wf.Status.Phase))
}

func IsRunning(wf *awfv1.Workflow) bool {
	if wf == nil {
		return false
	}
	return slices.HasString(runningStates, string(wf.Status.Phase))
}

type WorkflowItem struct {
	Task       *awfv1.DAGTask
	Template   *awfv1.Template
	NodeStatus *awfv1.NodeStatus
	Labels     map[string]string
}

// ParseWorkflow returns task name -> *WorkflowItem
func ParseWorkflow(wf *awfv1.Workflow) map[string]*WorkflowItem {
	if wf == nil {
		return nil
	}
	pWf := make(map[string]*WorkflowItem)
	for _, task := range wf.Spec.Templates[0].DAG.Tasks {
		initTask(task, pWf)
	}
	for i, template := range wf.Spec.Templates {
		if i != 0 {
			addTemplate(template, pWf)
		}
	}
	for _, nodeStatus := range wf.Status.Nodes {
		addNodeStatus(nodeStatus, pWf)
	}
	return pWf
}

func initTask(task awfv1.DAGTask, pWf map[string]*WorkflowItem) {
	pWf[task.Name] = &WorkflowItem{
		Task: &task,
	}
}

func addTemplate(template awfv1.Template, pWf map[string]*WorkflowItem) {
	pWf[template.Name].Template = &template
	pWf[template.Name].Labels = template.Metadata.Labels
}

func addNodeStatus(nodeStatus awfv1.NodeStatus, pWf map[string]*WorkflowItem) {
	if nodeStatus.Type != awfv1.NodeTypePod {
		return
	}
	pWf[nodeStatus.TemplateName].NodeStatus = &nodeStatus
}

func (wfItem *WorkflowItem) StartedAt() *time.Time {
	if wfItem.NodeStatus != nil && !wfItem.NodeStatus.StartedAt.Time.IsZero() {
		return &wfItem.NodeStatus.StartedAt.Time
	}
	return nil
}

func (wfItem *WorkflowItem) FinishedAt() *time.Time {
	if wfItem.NodeStatus != nil && !wfItem.NodeStatus.FinishedAt.Time.IsZero() {
		return &wfItem.NodeStatus.FinishedAt.Time
	}
	return nil
}

func (wfItem *WorkflowItem) Phase() *awfv1.NodePhase {
	if wfItem.NodeStatus != nil {
		return &wfItem.NodeStatus.Phase
	}
	return nil
}

func (wfItem *WorkflowItem) Dependencies() strset.Set {
	if wfItem.Task != nil && wfItem.Task.Dependencies != nil {
		return strset.New(wfItem.Task.Dependencies...)
	}

	return strset.New()
}
