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

	argowf "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	argoclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	argoclientapi "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclientrest "k8s.io/client-go/rest"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
)

var (
	doneStates = []string{
		string(argowf.NodeSucceeded),
		string(argowf.NodeSkipped),
		string(argowf.NodeFailed),
		string(argowf.NodeError),
	}
	runningStates = []string{
		string(argowf.NodeRunning),
	}
)

type Client struct {
	workflowClient argoclientapi.WorkflowInterface
	namespace      string
}

func New(restConfig *kclientrest.Config, namespace string) *Client {
	client := &Client{
		namespace: namespace,
	}
	wfcs := argoclientset.NewForConfigOrDie(restConfig)
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

func (c *Client) NewWorkflow(name string, labels ...map[string]string) *argowf.Workflow {
	name = "argo-" + name
	if !strings.HasSuffix(name, "-") && !strings.HasSuffix(name, "_") {
		name = name + "-"
	}
	allLabels := maps.MergeStrMaps(labels...)

	return &argowf.Workflow{
		ObjectMeta: kmeta.ObjectMeta{
			GenerateName: name,
			Namespace:    c.namespace,
			Labels:       allLabels,
		},
		Spec: argowf.WorkflowSpec{
			ServiceAccountName: "argo-executor",
			Entrypoint:         "DAG",
			Templates: []argowf.Template{
				{
					Name: "DAG",
					DAG: &argowf.DAGTemplate{
						Tasks: []argowf.DAGTask{},
					},
				},
			},
		},
	}
}

func AddTask(wf *argowf.Workflow, task *WorkflowTask) *argowf.Workflow {
	if task == nil {
		return wf
	}

	DAGTask := argowf.DAGTask{
		Name:         task.Name,
		Template:     task.Name,
		Dependencies: slices.RemoveEmptiesAndUnique(task.Dependencies),
	}

	// All tasks are added to the DAG template which is first
	wf.Spec.Templates[0].DAG.Tasks = append(wf.Spec.Templates[0].DAG.Tasks, DAGTask)

	labels := task.Labels
	labels["argo"] = "true"

	template := argowf.Template{
		Name: task.Name,
		Resource: &argowf.ResourceTemplate{
			Action:           task.Action,
			Manifest:         task.Manifest,
			SuccessCondition: task.SuccessCondition,
			FailureCondition: task.FailureCondition,
		},
		Metadata: argowf.Metadata{
			Labels: labels,
		},
	}

	wf.Spec.Templates = append(wf.Spec.Templates, template)

	return wf
}

func EnableGC(spec kmeta.Object) {
	ownerReferences := spec.GetOwnerReferences()
	ownerReferences = append(ownerReferences, kmeta.OwnerReference{
		APIVersion:         "argoproj.io/v1alpha1",
		Kind:               "Workflow",
		Name:               "{{workflow.name}}",
		UID:                "{{workflow.uid}}",
		BlockOwnerDeletion: pointer.Bool(false),
	})
	spec.SetOwnerReferences(ownerReferences)
}

func NumTasks(wf *argowf.Workflow) int {
	if wf == nil || len(wf.Spec.Templates) == 0 {
		return 0
	}
	return len(wf.Spec.Templates[0].DAG.Tasks)
}

func (c *Client) Run(wf *argowf.Workflow) error {
	_, err := c.workflowClient.Create(wf)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (c *Client) List(opts *kmeta.ListOptions) ([]argowf.Workflow, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	wfList, err := c.workflowClient.List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return wfList.Items, nil
}

func (c *Client) ListByLabels(labels map[string]string) ([]argowf.Workflow, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: k8s.LabelSelector(labels),
	}
	return c.List(opts)
}

func (c *Client) ListByLabel(labelKey string, labelValue string) ([]argowf.Workflow, error) {
	return c.ListByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) ListRunning(labels ...map[string]string) ([]argowf.Workflow, error) {
	wfs, err := c.ListByLabels(maps.MergeStrMaps(labels...))
	if err != nil {
		return wfs, err
	}
	runningWfs := []argowf.Workflow{}
	for _, wf := range wfs {
		if IsRunning(&wf) {
			runningWfs = append(runningWfs, wf)
		}
	}
	return runningWfs, nil
}

func (c *Client) ListDone(labels ...map[string]string) ([]argowf.Workflow, error) {
	wfs, err := c.ListByLabels(maps.MergeStrMaps(labels...))
	if err != nil {
		return wfs, err
	}
	doneWfs := []argowf.Workflow{}
	for _, wf := range wfs {
		if IsDone(&wf) {
			doneWfs = append(doneWfs, wf)
		}
	}
	return doneWfs, nil
}

func (c *Client) Delete(wfName string) (bool, error) {
	err := c.workflowClient.Delete(wfName, &kmeta.DeleteOptions{})
	if kerrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) DeleteMultiple(wfs []argowf.Workflow) error {
	errs := []error{}
	for _, wf := range wfs {
		_, err := c.Delete(wf.Name)
		errs = append(errs, err)
	}
	return errors.FirstError(errs...)
}

func IsDone(wf *argowf.Workflow) bool {
	if wf == nil {
		return true
	}
	return slices.HasString(doneStates, string(wf.Status.Phase))
}

func IsRunning(wf *argowf.Workflow) bool {
	if wf == nil {
		return false
	}
	return slices.HasString(runningStates, string(wf.Status.Phase))
}

type WorkflowItem struct {
	Task       *argowf.DAGTask
	Template   *argowf.Template
	NodeStatus *argowf.NodeStatus
	Labels     map[string]string
}

// ParseWorkflow returns task name -> *WorkflowItem
func ParseWorkflow(wf *argowf.Workflow) map[string]*WorkflowItem {
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

func initTask(task argowf.DAGTask, pWf map[string]*WorkflowItem) {
	pWf[task.Name] = &WorkflowItem{
		Task: &task,
	}
}

func addTemplate(template argowf.Template, pWf map[string]*WorkflowItem) {
	pWf[template.Name].Template = &template
	pWf[template.Name].Labels = template.Metadata.Labels
}

func addNodeStatus(nodeStatus argowf.NodeStatus, pWf map[string]*WorkflowItem) {
	if nodeStatus.Type != argowf.NodeTypePod {
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

func (wfItem *WorkflowItem) Phase() *argowf.NodePhase {
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
