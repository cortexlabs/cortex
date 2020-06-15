/*
Copyright 2020 Cortex Labs, Inc.

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

package batch

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/cloud"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	ok8s "github.com/cortexlabs/cortex/pkg/operator/k8s"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kcore "k8s.io/api/core/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kunstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	klabels "k8s.io/apimachinery/pkg/labels"
)

// TODO move

func QueuesPerAPI(apiName string) ([]string, error) {
	response, err := config.AWS.SQS().ListQueues(&sqs.ListQueuesInput{
		QueueNamePrefix: aws.String(apiName),
	})

	if err != nil {
		return nil, err
	}

	debug.Pp(response)

	queueNames := make([]string, len(response.QueueUrls))
	for i := range queueNames {
		queueNames[i] = *response.QueueUrls[i]
	}

	return queueNames, nil
}

// func GetLatestAPISpec
func UpdateAPI(apiConfig *userconfig.API, projectID string) (*spec.API, string, error) {
	prevVirtualService, err := getK8sResources(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	api := spec.GetAPISpec(apiConfig, projectID, "") // TODO deployment id
	if err := config.AWS.UploadMsgpackToS3(api, config.Cluster.Bucket, api.Key); err != nil {
		return nil, "", errors.Wrap(err, "upload api spec")
	}

	err = applyK8sResources(api, prevVirtualService)
	if err != nil {
		return nil, "", err
	}

	return api, "", nil
}

func DeleteAPI(apiName string, keepCache bool) error {
	err := parallel.RunFirstErr(
		func() error {
			return deleteK8sResources(apiName)
		},
		func() error {
			if keepCache {
				return nil
			}
			// best effort deletion
			deleteS3Resources(apiName)
			return nil
		},
		func() error {
			queues, _ := QueuesPerAPI(apiName)
			for _, queueURL := range queues {
				DeleteQueue(queueURL)
			}
			return nil
		},
	)

	if err != nil {
		return err
	}

	return nil
}

// TODO Remove
func getK8sResources(apiName string) (*kunstructured.Unstructured, error) {
	virtualService, err := config.K8s.GetVirtualService(K8sName(apiName))
	return virtualService, err
}

// TODO rename
func applyK8sResources(api *spec.API, prevVirtualService *kunstructured.Unstructured) error {
	newVirtualService := virtualServiceSpec(api)

	if prevVirtualService == nil {
		_, err := config.K8s.CreateVirtualService(newVirtualService)
		return err
	}

	_, err := config.K8s.UpdateVirtualService(prevVirtualService, newVirtualService)
	return err
}

func deleteK8sResources(apiName string) error {
	return parallel.RunFirstErr(
		func() error {
			_, err := config.K8s.DeleteJobs(&kmeta.ListOptions{
				LabelSelector: klabels.SelectorFromSet(map[string]string{"apiName": apiName}).String(),
			})
			return err
		},
		func() error {
			_, err := config.K8s.DeleteVirtualService(K8sName(apiName))
			return err
		},
	)
}

func deleteS3Resources(apiName string) error {
	return parallel.RunFirstErr(
		func() error {
			prefix := filepath.Join("apis", apiName)
			return config.AWS.DeleteS3Dir(config.Cluster.Bucket, prefix, true)
		},
		func() error {
			prefix := filepath.Join("jobs", apiName)
			return config.AWS.DeleteS3Dir(config.Cluster.Bucket, prefix, true)
		},
	)
}

func IsAPIDeployed(apiName string) (bool, error) {
	virtualService, err := config.K8s.GetVirtualService(K8sName(apiName))
	if err != nil {
		return false, err
	}
	return virtualService != nil, nil
}

// TODO rename
func virtualServiceSpec(api *spec.API) *kunstructured.Unstructured {
	return k8s.VirtualService(&k8s.VirtualServiceSpec{
		Name:        K8sName(api.Name),
		Gateways:    []string{"apis-gateway"},
		ServiceName: "operator",
		ServicePort: _defaultPortInt32,
		PrefixPath:  api.Endpoint,
		Rewrite:     pointer.String(filepath.Join("batch", api.Name)),
		Labels: map[string]string{
			"apiName": api.Name,
			"apiID":   api.ID,
			"apiType": api.Type.String(),
		},
	})
}

// TODO add yaml
type Submission struct {
	Items       []json.RawMessage `json:"items"`
	Parallelism int               `json:"parallelism"`
	JobConfig   interface{}       `json:"config"`
}

type JobSpec struct {
	ID              string      `json:"job_id"`
	APIName         string      `json:"api_name"`
	APIID           string      `json:"api_id"`
	SQSUrl          string      `json:"sqs_url"`
	Config          interface{} `json:"config"`
	Parallelism     int         `json:"parallelism"`
	TotalPartitions int         `json:"total_partitions"`
	Status          Code        `json:"status"`
	StartTime       time.Time   `json:"start_time"`
	EndTime         *time.Time  `json:"end_time"`
	LastUpdated     time.Time   `json:"last_updated"`
}

func JobKey(apiName, jobID string) string {
	return filepath.Join("jobs", apiName, consts.CortexVersion, jobID)
}

func SubmitJob(apiName string, submission Submission) (*JobSpec, error) {
	// submission validation
	if len(submission.Items) == 0 {
		fmt.Println("here")
		// throw error here
	}

	virtualService, err := getK8sResources(apiName)
	if err != nil {
		return nil, err
	}

	apiID := virtualService.GetLabels()["apiID"]

	apiSpec, err := cloud.DownloadAPISpec(apiName, apiID)
	if err != nil {
		return nil, err
	}

	jobID := k8s.RandomName()[20:]

	output, err := config.AWS.SQS().CreateQueue(
		&sqs.CreateQueueInput{
			Attributes: map[string]*string{
				"FifoQueue":                 aws.String("true"),
				"ContentBasedDeduplication": aws.String("true"),
			},
			QueueName: aws.String(apiName + "-" + jobID + ".fifo"),
			Tags: map[string]*string{
				clusterconfig.ClusterNameTag: &config.Cluster.ClusterName,
				"apiName":                    &apiSpec.Name,
				"apiID":                      &apiSpec.ID,
				"jobID":                      &jobID,
			},
		},
	)
	if err != nil {
		return nil, errors.Wrap(err) // TODO
	}

	jobSpec := JobSpec{
		ID:          jobID,
		APIID:       apiSpec.ID,
		APIName:     apiSpec.Name,
		SQSUrl:      *output.QueueUrl,
		Config:      submission.JobConfig,
		Parallelism: submission.Parallelism,
		LastUpdated: time.Now(),
		StartTime:   time.Now(),
		Status:      Enqueuing,
	}

	err = config.AWS.UploadJSONToS3(jobSpec, config.Cluster.Bucket, JobKey(apiName, jobID))
	if err != nil {
		DeleteQueue(*output.QueueUrl)
		return nil, err
	}

	t := ThreadSafeJobSpec{
		JobSpec: jobSpec,
	}

	go DeployJob(&t, &submission)

	return &jobSpec, nil
}

type ThreadSafeJobSpec struct {
	JobSpec
	mux sync.Mutex
}

func (t *ThreadSafeJobSpec) UpdateLiveness() error {
	t.mux.Lock()
	defer t.mux.Unlock()
	jobSpec, err := DownloadJobSpec(t.APIName, t.ID)
	if err != nil {
		return err // TODO
	}

	if jobSpec.Status != Enqueuing {
		return nil
	}

	jobSpec.LastUpdated = time.Now()

	err := UploadJobSpec(jobSpec)
	if err != nil {
		return err // TODO
	}

	return nil
}

func (t *ThreadSafeJobSpec) PushToS3() error {
	t.mux.Lock()
	defer t.mux.Unlock()

	t.LastUpdated = time.Now()
	err := UploadJobSpec(&t.JobSpec)
	if err != nil {
		return err // TODO
	}

	return nil
}

func UploadJobSpec(jobSpec *JobSpec) error {
	err := config.AWS.UploadJSONToS3(jobSpec, config.Cluster.Bucket, JobKey(jobSpec.APIName, jobSpec.ID))
	if err != nil {
		return err // TODO
	}

	return nil
}

func cronErrHandler(cronName string) func(error) { // TODO what happens if liveness fails? Delete API?
	return func(err error) {
		err = errors.Wrap(err, cronName+" cron failed")
		telemetry.Error(err)
		errors.PrintError(err)
	}
}

func DeployJob(jobSpec *ThreadSafeJobSpec, submission *Submission) {
	livenessCron := cron.Run(jobSpec.UpdateLiveness, cronErrHandler(fmt.Sprintf("enqueueing %s for api %s liveness failed", jobSpec.ID, jobSpec.APIName)), 10*time.Second)
	defer livenessCron.Cancel()

	err := Enqueue(jobSpec, submission)
	if err != nil {
		jobSpec.Status = Error
		jobSpec.PushToS3()
	}

	err = ApplyK8sJob(jobSpec)
	if err != nil {
		jobSpec.Status = Error
		jobSpec.PushToS3()
	}
}

func Enqueue(jobSpec *ThreadSafeJobSpec, submission *Submission) error {
	total := 0
	for i, item := range submission.Items {
		randomId := k8s.RandomName()

		_, err := config.AWS.SQS().SendMessage(&sqs.SendMessageInput{
			MessageDeduplicationId: aws.String(randomId),
			QueueUrl:               aws.String(jobSpec.SQSUrl),
			MessageBody:            aws.String(string(item)),
			MessageGroupId:         aws.String(randomId),
		})

		if err != nil {
			return errors.Wrap(errors.WithStack(err), fmt.Sprintf("item %d", i)) // TODO
		}
		total++
	}

	jobSpec.TotalPartitions = total
	jobSpec.Status = Running
	err := jobSpec.PushToS3()
	if err != nil {
		return err
	}

	return nil
}

func ApplyK8sJob(jobSpec *ThreadSafeJobSpec) error {
	apiSpec, err := cloud.DownloadAPISpec(jobSpec.APIName, jobSpec.APIID)
	if err != nil {
		return err // TODO
	}

	apiSpec.Predictor.Env["SQS_QUEUE_URL"] = jobSpec.SQSUrl // TODO
	apiSpec.Predictor.Env["CORTEX_JOB_SPEC"] = "s3://" + config.Cluster.Bucket + "/" + JobKey(jobSpec.APIName, jobSpec.ID)
	podSpec := ok8s.PythonPodSpec(apiSpec)
	for _, container := range podSpec.Containers {
		if container.Name == ok8s.APIContainerName {
			container.Env = append(container.Env, kcore.EnvVar{
				Name:  "CORTEX_SQS_QUEUE",
				Value: *output.QueueUrl,
			})
		}
	}

	job := ok8s.PythonJobSpec(apiSpec, podSpec, jobSpec.Parallelism)
	job.Labels["jobID"] = jobSpec.ID

	_, err = config.K8s.CreateJob(job)
	if err != nil {
		return err
	}

	return nil
}

func DeleteQueue(queueURL string) error {
	_, err := config.AWS.SQS().DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})

	return err
}

func DeleteJob(apiName, jobID string) error {
	_, err := config.K8s.DeleteJobs(&kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(map[string]string{"apiName": apiName}).String(),
	})
	if err != nil {
		return err
	}

	output, err := config.AWS.SQS().GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(apiName + "-" + jobID + ".fifo"),
	})
	if err != nil {
		return err
	}

	_, err = config.AWS.SQS().DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: output.QueueUrl,
	})
	if err != nil {
		return err
	}

	return nil
}

func DownloadJobSpec(apiName, jobID string) (*JobSpec, error) {
	jobSpec := JobSpec{}
	err := config.AWS.ReadJSONFromS3(&jobSpec, config.Cluster.Bucket, JobKey(apiName, jobID))
	if err != nil {
		return nil, err // TODO
	}

	return &jobSpec, nil
}

func GetJobStatus(apiName, jobID string) error {
	jobSpec := JobSpec{}
	err := config.AWS.ReadJSONFromS3(&jobSpec, config.Cluster.Bucket, JobKey(apiName, jobID))
	if err != nil {
		return err // TODO
	}

	jobStatus := JobStatus{
		Total: jobSpec.TotalPartitions,
	}

	queueURL, err := QueueURL(apiName, jobID)
	if err != nil {
		return err // TODO
	}

}

type JobStatus struct {
	Total                   int
	InQueue                 int
	InProgress              int
	Succeeded               int
	Failed                  int
	StartTime               *time.Time
	EndTime                 *time.Time
	AverageTimePerPartition *float64
	WorkerStats             *status.SubReplicaCounts
}
