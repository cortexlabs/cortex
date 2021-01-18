/*
Copyright 2021 Cortex Labs, Inc.

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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/random"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/lib/logging"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

const (
	_enqueuingLivenessPeriod = 20 * time.Second
	_s3DownloadChunkSize     = 32 * 1024 * 1024
)

var operatorLogger = logging.GetOperatorLogger()

func randomMessageID() string {
	return random.String(40) // maximum is 80 (for sqs.SendMessageBatchRequestEntry.Id) but this ID may show up in a user error message
}

func enqueue(jobSpec *spec.BatchJob, submission *schema.BatchJobSubmission) (int, error) {
	livenessUpdater := func() error {
		return job.UpdateLiveness(jobSpec.JobKey)
	}

	livenessCron := cron.Run(livenessUpdater, operator.ErrorHandler(fmt.Sprintf("liveness check for %s", jobSpec.UserString())), _enqueuingLivenessPeriod)
	defer livenessCron.Cancel()

	totalBatches := 0
	var err error
	if submission.ItemList != nil {
		totalBatches, err = enqueueItems(jobSpec, submission.ItemList)
		if err != nil {
			return 0, err
		}
	} else if submission.FilePathLister != nil {
		totalBatches, err = enqueueS3Paths(jobSpec, submission.FilePathLister)
		if err != nil {
			return 0, err
		}
	} else if submission.DelimitedFiles != nil {
		totalBatches, err = enqueueS3FileContents(jobSpec, submission.DelimitedFiles)
		if err != nil {
			return 0, err
		}
	}

	randomMessageID := randomMessageID()
	_, err = config.AWS.SQS().SendMessage(&sqs.SendMessageInput{
		QueueUrl:               aws.String(jobSpec.SQSUrl),
		MessageBody:            aws.String("\"job_complete\""),
		MessageDeduplicationId: aws.String(randomMessageID), // prevent content based deduping
		MessageGroupId:         aws.String(randomMessageID), // aws recommends message group id per message to improve chances of exactly-once
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"job_complete": {
				DataType:    aws.String("String"),
				StringValue: aws.String("true"),
			},
			"api_name": {
				DataType:    aws.String("String"),
				StringValue: aws.String(jobSpec.APIName),
			},
			"job_id": {
				DataType:    aws.String("String"),
				StringValue: aws.String(jobSpec.ID),
			},
		},
	})
	if err != nil {
		return 0, errors.Wrap(err, "failed to enqueue job_complete placeholder")
	}

	return totalBatches, nil
}

func enqueueItems(jobSpec *spec.BatchJob, itemList *schema.ItemList) (int, error) {
	batchCount := len(itemList.Items) / itemList.BatchSize
	if len(itemList.Items)%itemList.BatchSize != 0 {
		batchCount++
	}

	jobLogger, err := operator.GetJobLogger(jobSpec.JobKey)
	if err != nil {
		return 0, nil
	}

	jobLogger.Infof("partitioning %d items found in job submission into %d batches of size %d", len(itemList.Items), batchCount, itemList.BatchSize)

	uploader := newSQSBatchUploader(jobSpec.SQSUrl, jobSpec.JobKey)

	for i := 0; i < batchCount; i++ {
		min := i * (itemList.BatchSize)
		max := (i + 1) * (itemList.BatchSize)
		if max > len(itemList.Items) {
			max = len(itemList.Items)
		}

		jsonBytes, err := json.Marshal(itemList.Items[min:max])
		if err != nil {
			if itemList.BatchSize == 1 {
				return 0, errors.Wrap(err, fmt.Sprintf("item %d", i))
			}
			return 0, errors.Wrap(err, fmt.Sprintf("items with index between %d to %d", min, max))
		}

		err = uploader.AddToBatch(randomMessageID(), pointer.String(string(jsonBytes)))
		if err != nil {
			if itemList.BatchSize == 1 {
				return 0, errors.Wrap(err, fmt.Sprintf("item %d", i))
			}
			return 0, errors.Wrap(err, fmt.Sprintf("items with index between %d to %d", min, max))
		}
		if uploader.TotalBatches%100 == 0 {
			jobLogger.Infof("enqueued %d batches", uploader.TotalBatches)
		}
	}

	err = uploader.Flush()
	if err != nil {
		return 0, err
	}

	return uploader.TotalBatches, nil
}

func enqueueS3Paths(jobSpec *spec.BatchJob, s3PathsLister *schema.FilePathLister) (int, error) {
	jobLogger, err := operator.GetJobLogger(jobSpec.JobKey)
	if err != nil {
		return 0, err
	}

	var s3PathList []string
	uploader := newSQSBatchUploader(jobSpec.SQSUrl, jobSpec.JobKey)

	err = s3IteratorFromLister(s3PathsLister.S3Lister, func(bucket string, s3Obj *s3.Object) (bool, error) {
		s3Path := awslib.S3Path(bucket, *s3Obj.Key)

		s3PathList = append(s3PathList, s3Path)
		if len(s3PathList) == s3PathsLister.BatchSize {
			err := addS3PathsToQueue(uploader, s3PathList)
			if err != nil {
				return false, err
			}
			s3PathList = nil

			if uploader.TotalBatches%100 == 0 {
				jobLogger.Infof("enqueued %d batches", uploader.TotalBatches)
			}
		}

		return true, nil
	})
	if err != nil {
		return 0, err
	}

	if len(s3PathList) > 0 {
		err := addS3PathsToQueue(uploader, s3PathList)
		if err != nil {
			return 0, err
		}
	}

	err = uploader.Flush()
	if err != nil {
		return 0, err
	}

	return uploader.TotalBatches, nil
}

func addS3PathsToQueue(uploader *sqsBatchUploader, s3PathList []string) error {
	jsonBytes, err := json.Marshal(s3PathList)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("batch %d", uploader.TotalBatches))
	}

	err = uploader.AddToBatch(randomMessageID(), pointer.String(string(jsonBytes)))
	if err != nil {
		return err
	}
	return nil
}

type jsonBuffer struct {
	BatchSize   int
	messageList []json.RawMessage
}

func newJSONBuffer(batchSize int) *jsonBuffer {
	return &jsonBuffer{
		BatchSize:   batchSize,
		messageList: make([]json.RawMessage, 0, batchSize),
	}
}

func (j *jsonBuffer) Add(jsonMessage json.RawMessage) {
	j.messageList = append(j.messageList, jsonMessage)
}

func (j *jsonBuffer) Clear() {
	j.messageList = make([]json.RawMessage, 0, j.BatchSize)
}

func (j *jsonBuffer) Length() int {
	return len(j.messageList)
}

func enqueueS3FileContents(jobSpec *spec.BatchJob, delimitedFiles *schema.DelimitedFiles) (int, error) {
	jobLogger, err := operator.GetJobLogger(jobSpec.JobKey)
	if err != nil {
		return 0, err
	}

	jsonMessageList := newJSONBuffer(delimitedFiles.BatchSize)
	uploader := newSQSBatchUploader(jobSpec.SQSUrl, jobSpec.JobKey)

	bytesBuffer := bytes.NewBuffer([]byte{})
	err = s3IteratorFromLister(delimitedFiles.S3Lister, func(bucket string, s3Obj *s3.Object) (bool, error) {
		s3Path := awslib.S3Path(bucket, *s3Obj.Key)
		jobLogger.Infof("enqueuing contents from file %s", s3Path)

		awsClientForBucket, err := awslib.NewFromClientS3Path(s3Path, config.AWS)
		if err != nil {
			return false, err
		}

		itemIndex := 0
		err = awsClientForBucket.S3FileIterator(bucket, s3Obj, _s3DownloadChunkSize, func(readCloser io.ReadCloser, isLastChunk bool) (bool, error) {
			_, err := bytesBuffer.ReadFrom(readCloser)
			if err != nil {
				return false, err
			}
			err = streamJSONToQueue(jobSpec, uploader, bytesBuffer, jsonMessageList, &itemIndex)
			if err != nil {
				if errors.CauseOrSelf(err) != io.ErrUnexpectedEOF || (errors.CauseOrSelf(err) == io.ErrUnexpectedEOF && isLastChunk) {
					return false, err
				}
			}
			return true, nil
		})
		if err != nil {
			return false, err
		}

		return true, nil
	})
	if err != nil {
		return 0, err
	}

	if jsonMessageList.Length() != 0 {
		err := addJSONObjectsToQueue(uploader, jsonMessageList)
		if err != nil {
			return 0, err
		}
	}
	err = uploader.Flush()
	if err != nil {
		return 0, err
	}

	return uploader.TotalBatches, nil
}

func streamJSONToQueue(jobSpec *spec.BatchJob, uploader *sqsBatchUploader, bytesBuffer *bytes.Buffer, jsonMessageList *jsonBuffer, itemIndex *int) error {
	jobLogger, err := operator.GetJobLogger(jobSpec.JobKey)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(bytesBuffer)
	for {
		var doc json.RawMessage

		err := dec.Decode(&doc)
		if err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			bytesBuffer.Reset()
			bytesBuffer.ReadFrom(dec.Buffered())
			return io.ErrUnexpectedEOF
		} else if err != nil {
			return errors.Wrap(err, fmt.Sprintf("item %d", *itemIndex))
		}

		if len(doc) > _messageSizeLimit {
			return errors.Wrap(ErrorMessageExceedsMaxSize(len(doc), _messageSizeLimit), fmt.Sprintf("item %d", *itemIndex))
		}
		*itemIndex++
		jsonMessageList.Add(doc)
		if jsonMessageList.Length() == jsonMessageList.BatchSize {
			err := addJSONObjectsToQueue(uploader, jsonMessageList)
			if err != nil {
				return err
			}
			jsonMessageList.Clear()

			if uploader.TotalBatches%100 == 0 {
				jobLogger.Infof("enqueued %d batches", uploader.TotalBatches)
			}
		}
	}

	return nil
}

func addJSONObjectsToQueue(uploader *sqsBatchUploader, jsonMessageList *jsonBuffer) error {
	jsonBytes, err := json.Marshal(jsonMessageList.messageList)
	if err != nil {
		return err
	}

	err = uploader.AddToBatch(randomMessageID(), pointer.String(string(jsonBytes)))
	if err != nil {
		return err
	}

	return nil
}
