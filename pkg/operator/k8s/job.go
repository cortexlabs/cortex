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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const JobSuccessCondition = "status.succeeded > 0"
const JobFailureCondition = "status.failed > 0"

var jobTypeMeta = metav1.TypeMeta{
	APIVersion: "batch/v1",
	Kind:       "Job",
}

type JobSpec struct {
	Name      string
	Namespace string
	PodSpec   PodSpec
	Labels    map[string]string
}

func Job(spec *JobSpec) *batchv1.Job {
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

	job := &batchv1.Job{
		TypeMeta: jobTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
			Labels:    spec.Labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Parallelism:  &parallelism,
			Completions:  &completions,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
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

func CreateJob(spec *JobSpec) (*batchv1.Job, error) {
	job, err := jobClient.Create(Job(spec))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return job, nil
}

func UpdateJob(job *batchv1.Job) (*batchv1.Job, error) {
	job, err := jobClient.Update(job)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return job, nil
}

func GetJob(name string) (*batchv1.Job, error) {
	job, err := jobClient.Get(name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	job.TypeMeta = jobTypeMeta
	return job, nil
}

func DeleteJob(name string) (bool, error) {
	err := jobClient.Delete(name, deleteOpts)
	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func JobExists(name string) (bool, error) {
	job, err := GetJob(name)
	if err != nil {
		return false, err
	}
	return job != nil, nil
}

func ListJobs(opts *metav1.ListOptions) ([]batchv1.Job, error) {
	if opts == nil {
		opts = &metav1.ListOptions{}
	}
	jobList, err := jobClient.List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range jobList.Items {
		jobList.Items[i].TypeMeta = jobTypeMeta
	}
	return jobList.Items, nil
}

func ListJobsByLabels(labels map[string]string) ([]batchv1.Job, error) {
	opts := &metav1.ListOptions{
		LabelSelector: LabelSelector(labels),
	}
	return ListJobs(opts)
}

func ListJobsByLabel(labelKey string, labelValue string) ([]batchv1.Job, error) {
	return ListJobsByLabels(map[string]string{labelKey: labelValue})
}

func JobMap(jobs []batchv1.Job) map[string]batchv1.Job {
	jobMap := map[string]batchv1.Job{}
	for _, job := range jobs {
		jobMap[job.Name] = job
	}
	return jobMap
}
