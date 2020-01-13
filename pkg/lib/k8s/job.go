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
	kbatch "k8s.io/api/batch/v1"
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

var jobTypeMeta = kmeta.TypeMeta{
	APIVersion: "batch/v1",
	Kind:       "Job",
}

type JobSpec struct {
	Name        string
	Namespace   string
	PodSpec     PodSpec
	Labels      map[string]string
	Annotations map[string]string
}

func Job(spec *JobSpec) *kbatch.Job {
	if spec.Namespace == "" {
		spec.Namespace = "default"
	}
	if spec.PodSpec.Namespace == "" {
		spec.PodSpec.Namespace = spec.Namespace
	}
	if spec.PodSpec.Name == "" {
		spec.PodSpec.Name = spec.Name
	}

	parallelism := int32(1)
	backoffLimit := int32(0)
	completions := int32(1)

	job := &kbatch.Job{
		TypeMeta: jobTypeMeta,
		ObjectMeta: kmeta.ObjectMeta{
			Name:        spec.Name,
			Namespace:   spec.Namespace,
			Labels:      spec.Labels,
			Annotations: spec.Annotations,
		},
		Spec: kbatch.JobSpec{
			BackoffLimit: &backoffLimit,
			Parallelism:  &parallelism,
			Completions:  &completions,
			Template: kcore.PodTemplateSpec{
				ObjectMeta: kmeta.ObjectMeta{
					Name:      spec.PodSpec.Name,
					Namespace: spec.PodSpec.Namespace,
					Labels:    spec.PodSpec.Labels,
				},
				Spec: spec.PodSpec.K8sPodSpec,
			},
		},
	}
	return job
}

func (c *Client) CreateJob(job *kbatch.Job) (*kbatch.Job, error) {
	job.TypeMeta = jobTypeMeta
	job, err := c.jobClient.Create(job)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return job, nil
}

func (c *Client) UpdateJob(job *kbatch.Job) (*kbatch.Job, error) {
	job.TypeMeta = jobTypeMeta
	job, err := c.jobClient.Update(job)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return job, nil
}

func (c *Client) ApplyJob(job *kbatch.Job) (*kbatch.Job, error) {
	existing, err := c.GetJob(job.Name)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.CreateJob(job)
	}
	return c.UpdateJob(job)
}

func (c *Client) GetJob(name string) (*kbatch.Job, error) {
	job, err := c.jobClient.Get(name, kmeta.GetOptions{})
	if kerrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	job.TypeMeta = jobTypeMeta
	return job, nil
}

func (c *Client) DeleteJob(name string) (bool, error) {
	err := c.jobClient.Delete(name, deleteOpts)
	if kerrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) JobExists(name string) (bool, error) {
	job, err := c.GetJob(name)
	if err != nil {
		return false, err
	}
	return job != nil, nil
}

func (c *Client) ListJobs(opts *kmeta.ListOptions) ([]kbatch.Job, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	jobList, err := c.jobClient.List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range jobList.Items {
		jobList.Items[i].TypeMeta = jobTypeMeta
	}
	return jobList.Items, nil
}

func (c *Client) ListJobsByLabels(labels map[string]string) ([]kbatch.Job, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelSelector(labels),
	}
	return c.ListJobs(opts)
}

func (c *Client) ListJobsByLabel(labelKey string, labelValue string) ([]kbatch.Job, error) {
	return c.ListJobsByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) ListJobsWithLabelKeys(labelKeys ...string) ([]kbatch.Job, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelExistsSelector(labelKeys...),
	}
	return c.ListJobs(opts)
}

func JobMap(jobs []kbatch.Job) map[string]kbatch.Job {
	jobMap := map[string]kbatch.Job{}
	for _, job := range jobs {
		jobMap[job.Name] = job
	}
	return jobMap
}

func (c *Client) IsJobRunning(name string) (bool, error) {
	job, err := c.GetJob(name)
	if err != nil {
		return false, err
	}
	if job == nil {
		return false, nil
	}
	return job.Status.CompletionTime == nil, nil
}
