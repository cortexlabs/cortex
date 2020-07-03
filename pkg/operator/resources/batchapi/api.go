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

package batchapi

import (
	"encoding/json"
	"fmt"
	"math"
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
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

var jobIDMutex = sync.Mutex{}

// Job id creation optimized for listing the most recently created jobs in S3. S3 objects are listed in ascending UTF-8 binary order. This should work until the year 2262.
func MonotonicallyDecreasingJobID() string {
	jobIDMutex.Lock()
	defer jobIDMutex.Unlock()

	i := math.MaxInt64 - time.Now().UnixNano()
	return fmt.Sprintf("%x", i)
}

func QueuesPerAPI(apiName string) ([]string, error) {
	response, err := config.AWS.SQS().ListQueues(&sqs.ListQueuesInput{
		QueueNamePrefix: aws.String("cortex" + "-" + apiName),
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
	prevVirtualService, err := getVirtualService(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	api := spec.GetAPISpec(apiConfig, projectID, "") // Deployment ID not needed for BatchAPI spec

	if prevVirtualService == nil {
		if err := config.AWS.UploadMsgpackToS3(api, config.Cluster.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}

		err = applyK8sResources(api, prevVirtualService)
		if err != nil {
			go deleteK8sResources(api.Name)
			return nil, "", err
		}

		err = operator.AddAPIToAPIGateway(*api.Networking.Endpoint, api.Networking.APIGateway)
		if err != nil {
			return nil, "", err
		}
		return api, fmt.Sprintf("creating %s", api.Name), nil
	}

	if !areAPIsEqual(prevVirtualService, virtualServiceSpec(api)) {
		if err := config.AWS.UploadMsgpackToS3(api, config.Cluster.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}

		err = applyK8sResources(api, prevVirtualService)
		if err != nil {
			go deleteK8sResources(api.Name)
			return nil, "", err
		}

		if err := operator.UpdateAPIGatewayK8s(prevVirtualService, api); err != nil {
			go deleteK8sResources(api.Name) // Delete k8s if update fails?
			return nil, "", err
		}
		return api, fmt.Sprintf("updating %s", api.Name), nil
	}

	return api, fmt.Sprintf("%s is up to date", api.Name), nil
}

func areAPIsEqual(v1, v2 *istioclientnetworking.VirtualService) bool {
	return v1.Labels["apiName"] == v2.Labels["apiName"] &&
		v1.Labels["apiID"] == v2.Labels["apiID"] &&
		operator.DoCortexAnnotationsMatch(v1, v2)
}

func DeleteAPI(apiName string, keepCache bool) error {
	// best effort deletion, so don't handle error yet
	virtualService, vsErr := config.K8s.GetVirtualService(operator.K8sName(apiName))

	err := parallel.RunFirstErr(
		func() error {
			return vsErr
		},
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
		func() error {
			err := operator.RemoveAPIFromAPIGatewayK8s(virtualService)
			if err != nil {
				return err
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
func getVirtualService(apiName string) (*istioclientnetworking.VirtualService, error) {
	virtualService, err := config.K8s.GetVirtualService(operator.K8sName(apiName))
	return virtualService, err
}

// TODO rename
func applyK8sResources(api *spec.API, prevVirtualService *istioclientnetworking.VirtualService) error {
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
			_, err := config.K8s.DeleteVirtualService(operator.K8sName(apiName))
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
	virtualService, err := config.K8s.GetVirtualService(operator.K8sName(apiName))
	if err != nil {
		return false, err
	}
	return virtualService != nil, nil
}

// TODO add yaml
type Submission struct {
	Items       []json.RawMessage `json:"items"`
	Parallelism int               `json:"parallelism"`
	JobConfig   interface{}       `json:"config"`
}

func APIJobPrefix(apiName string) string {
	return filepath.Join("jobs", apiName, consts.CortexVersion)
}

func JobKey(apiName, jobID string) string {
	return filepath.Join(APIJobPrefix(apiName), jobID)
}

func SubmitJob(apiName string, submission Submission) (*spec.Job, error) {
	// submission validation
	if len(submission.Items) == 0 {
		fmt.Println("here")
		// throw error here
	}

	virtualService, err := getVirtualService(apiName)
	if err != nil {
		return nil, err
	}
	// TODO what happens if virtualService is nil

	apiID := virtualService.GetLabels()["apiID"]

	apiSpec, err := operator.DownloadAPISpec(apiName, apiID)
	if err != nil {
		return nil, err
	}

	jobID := MonotonicallyDecreasingJobID()

	tags := map[string]string{
		"apiName": apiSpec.Name,
		"apiID":   apiSpec.ID,
		"jobID":   jobID,
	}

	for key, value := range config.Cluster.Tags {
		tags[key] = value
	}

	output, err := config.AWS.SQS().CreateQueue(
		&sqs.CreateQueueInput{
			Attributes: map[string]*string{
				"FifoQueue":                 aws.String("true"),
				"ContentBasedDeduplication": aws.String("true"),
				"VisibilityTimeout":         aws.String("90"),
			},
			QueueName: aws.String("cortex" + "-" + apiName + "-" + jobID + ".fifo"),
			Tags:      aws.StringMap(tags),
		},
	)
	if err != nil {
		return nil, errors.Wrap(err) // TODO
	}

	jobSpec := spec.Job{
		JobID: spec.JobID{
			APIName: apiSpec.Name,
			ID:      jobID,
		},
		APIID:       apiSpec.ID,
		SQSUrl:      *output.QueueUrl,
		Config:      submission.JobConfig,
		Parallelism: submission.Parallelism,
		Created:     time.Now(),
	}

	err = SetEnqueuingStatus(&jobSpec)
	if err != nil {
		DeleteQueue(*output.QueueUrl)
		return nil, err
	}

	go DeployJob(&jobSpec, &submission)

	return &jobSpec, nil
}

func UploadJobSpec(jobSpec *spec.Job) error {
	err := config.AWS.UploadJSONToS3(jobSpec, config.Cluster.Bucket, spec.JobSpecKey(jobSpec.JobID))
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

func DeployJob(jobSpec *spec.Job, submission *Submission) {
	livenessUpdater := func() error {
		return UpdateLiveness(jobSpec.JobID)
	}

	livenessCron := cron.Run(livenessUpdater, cronErrHandler(fmt.Sprintf("failed liveness check", jobSpec.ID, jobSpec.APIName)), 10*time.Second)
	defer livenessCron.Cancel()

	err := operator.CreateLogGroupForJob(jobSpec.JobID)
	if err != nil {
		fmt.Println(err.Error()) // TODO
		SetErroredStatus(jobSpec.JobID)
		DeleteJob(jobSpec.JobID)
	}

	err = operator.WriteToJobLogGroup(jobSpec.JobID, "started enqueueing partitions")
	if err != nil {
		fmt.Println(err.Error()) //  TODO
	}

	err = Enqueue(jobSpec, submission)
	if err != nil {
		fmt.Println("failed to enqueue")
		operator.WriteToJobLogGroup(jobSpec.JobID, err.Error())
		SetErroredStatus(jobSpec.JobID)
		DeleteJob(jobSpec.JobID)
	}

	err = operator.WriteToJobLogGroup(jobSpec.JobID, "completed enqueueing partitions")
	if err != nil {
		fmt.Println(err.Error()) //  TODO
	}

	err = SetRunningStatus(jobSpec)
	if err != nil {
		// TODO here
	}

	err = ApplyK8sJob(jobSpec)
	if err != nil {
		fmt.Println("DeployJob")
		operator.WriteToJobLogGroup(jobSpec.JobID, err.Error())
		SetErroredStatus(jobSpec.JobID)
		DeleteJob(jobSpec.JobID)
	}
}

func Enqueue(jobSpec *spec.Job, submission *Submission) error {
	total := 0
	startTime := time.Now()
	for i, item := range submission.Items {
		// TODO kill the goroutine? This should automatically error when the next enqueue message fails
		// for k := 0; k < 100; k++ {
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

	debug.Pp(time.Now().Sub(startTime).Milliseconds())

	jobSpec.TotalPartitions = total
	err := SetRunningStatus(jobSpec)
	if err != nil {
		return err
	}

	return nil
}

func ApplyK8sJob(jobSpec *spec.Job) error {
	apiSpec, err := operator.DownloadAPISpec(jobSpec.APIName, jobSpec.APIID)
	if err != nil {
		return err // TODO
	}
	job := k8sJobSpec(apiSpec, jobSpec)

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

func StopJob(jobID spec.JobID) error {
	return errors.FirstError(
		SetStoppedStatus(jobID),
		DeleteJob(jobID),
	)
	return nil
}

func DeleteJob(jobID spec.JobID) error {
	_, err := config.K8s.DeleteJobs(&kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(map[string]string{"apiName": jobID.APIName, "jobID": jobID.ID}).String(),
	})
	if err != nil {
		return err
	}

	queueURL, err := operator.QueueURL(jobID)
	if err != nil {
		return err
	}

	fmt.Println(queueURL)
	_, err = config.AWS.SQS().DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	if err != nil {
		return err
	}

	return nil
}
