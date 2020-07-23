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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

const (
	_lastUpdatedFile = "last_updated"
	_fileBuffer      = 32 * 1024 * 1024
)

func cronErrHandler(cronName string) func(error) {
	return func(err error) {
		err = errors.Wrap(err, cronName+" cron failed")
		telemetry.Error(err)
		errors.PrintError(err)
	}
}

func updateLiveness(jobKey spec.JobKey) error {
	s3Key := path.Join(jobKey.PrefixKey(), _lastUpdatedFile)
	err := config.AWS.UploadJSONToS3(time.Now(), config.Cluster.Bucket, s3Key)
	if err != nil {
		return errors.Wrap(err, "failed to update liveness", jobKey.UserString())
	}
	return nil
}

func enqueue(jobSpec *spec.Job, submission *userconfig.JobSubmission) (int, error) {
	err := updateLiveness(jobSpec.JobKey)
	if err != nil {
		return 0, err
	}

	livenessUpdater := func() error {
		return updateLiveness(jobSpec.JobKey)
	}

	livenessCron := cron.Run(livenessUpdater, cronErrHandler(fmt.Sprintf("liveness check for %s", jobSpec.UserString())), 20*time.Second)
	defer livenessCron.Cancel()

	totalBatches := 0
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

	randomID := k8s.RandomName()
	_, err = config.AWS.SQS().SendMessage(&sqs.SendMessageInput{
		QueueUrl:               aws.String(jobSpec.SQSUrl),
		MessageBody:            aws.String("\"job_complete\""),
		MessageDeduplicationId: aws.String(randomID), // prevent content based deduping
		MessageGroupId:         aws.String(randomID), // aws recommends message group id per message to improve chances of exactly-once
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"job_complete": {
				DataType:    aws.String("String"),
				StringValue: aws.String("true"),
			},
		},
	})
	if err != nil {
		return 0, err
	}

	return totalBatches, nil
}

func enqueueItems(jobSpec *spec.Job, itemList *userconfig.ItemList) (int, error) {
	batchCount := len(itemList.Items) / *itemList.BatchSize
	if len(itemList.Items)%*itemList.BatchSize != 0 {
		batchCount++
	}

	writeToJobLogGroup(jobSpec.JobKey, fmt.Sprintf("partitioning %d items found in job submission into %d batches of size %d", len(itemList.Items), batchCount, *itemList.BatchSize))

	uploader := awslib.SQSBatchUploader{
		Client:   config.AWS,
		QueueURL: jobSpec.SQSUrl,
		Retries:  aws.Int(3),
	}

	for i := 0; i < batchCount; i++ {
		min := i * (*itemList.BatchSize)
		max := (i + 1) * (*itemList.BatchSize)
		if max > len(itemList.Items) {
			max = len(itemList.Items)
		}

		jsonBytes, err := json.Marshal(itemList.Items[min:max])
		if err != nil {
			return 0, errors.Wrap(err, fmt.Sprintf("batch %d", i))
		}
		randomID := k8s.RandomName()

		requestEntry := &sqs.SendMessageBatchRequestEntry{
			Id:                     aws.String(randomID),
			MessageBody:            aws.String(string(jsonBytes)),
			MessageDeduplicationId: aws.String(randomID), // prevent content based deduping
			MessageGroupId:         aws.String(randomID), // aws recommends message group id per message to improve chances of exactly-once
		}
		if err := requestEntry.Validate(); err != nil {
			return 0, err
		}

		err = uploader.AddToBatch(requestEntry)
		if err != nil {
			if *itemList.BatchSize > 1 {
				return 0, errors.Wrap(err, fmt.Sprintf("item %d", i))
			}
			return 0, errors.Wrap(err, fmt.Sprintf("items with index between %d to %d", min, max))
		}
		if uploader.TotalBatches%10 == 0 {
			writeToJobLogGroup(jobSpec.JobKey, fmt.Sprintf("enqueued %d batches", uploader.TotalBatches))
		}
	}

	err := uploader.Flush()
	if err != nil {
		return 0, err
	}

	return uploader.TotalBatches, nil
}

func enqueueS3Paths(jobSpec *spec.Job, s3PathsLister *userconfig.FilePathLister) (int, error) {
	FilePathLister := []string{}
	uploader := &awslib.SQSBatchUploader{
		Client:   config.AWS,
		QueueURL: jobSpec.SQSUrl,
		Retries:  aws.Int(3),
	}

	err := config.AWS.S3IteratorFromLister(s3PathsLister.S3Lister, func(bucket string, s3Obj *s3.Object) (bool, error) {
		s3Path := awslib.S3Path(bucket, *s3Obj.Key)
		fmt.Println(s3Path)
		FilePathLister = append(FilePathLister, s3Path)
		if len(FilePathLister) == *s3PathsLister.BatchSize {
			err := addToBatch(uploader, FilePathLister)
			if err != nil {
				return false, err
			}
			FilePathLister = nil

			if uploader.TotalBatches%10 == 0 {
				writeToJobLogGroup(jobSpec.JobKey, fmt.Sprintf("enqueued %d batches", uploader.TotalBatches))
			}
		}

		return true, nil
	})
	if err != nil {
		return 0, err
	}

	if len(FilePathLister) > 0 {
		err := addToBatch(uploader, FilePathLister)
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

func addToBatch(uploader *awslib.SQSBatchUploader, FilePathLister []string) error {
	jsonBytes, err := json.Marshal(FilePathLister)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("batch %d", uploader.TotalBatches))
	}
	randomID := k8s.RandomName()

	err = uploader.AddToBatch(&sqs.SendMessageBatchRequestEntry{
		Id:                     aws.String(randomID),
		MessageBody:            aws.String(string(jsonBytes)),
		MessageDeduplicationId: aws.String(randomID), // prevent content based deduping
		MessageGroupId:         aws.String(randomID), // aws recommends message group id per message to improve chances of exactly-once
	})
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

func enqueueS3FileContents(jobSpec *spec.Job, delimitedFiles *userconfig.DelimitedFiles) (int, error) {
	jsonMessageList := newJSONBuffer(*delimitedFiles.BatchSize)
	uploader := &awslib.SQSBatchUploader{
		Client:   config.AWS,
		QueueURL: jobSpec.SQSUrl,
		Retries:  aws.Int(3),
	}

	bytesBuffer := bytes.NewBuffer([]byte{})
	err := config.AWS.S3IteratorFromLister(delimitedFiles.S3Lister, func(bucket string, s3Obj *s3.Object) (bool, error) {
		s3Path := awslib.S3Path(bucket, *s3Obj.Key)

		itemIndex := 0
		err := config.AWS.S3FileIterator(bucket, s3Obj, _fileBuffer, func(readCloser io.ReadCloser, isLastChunk bool) (bool, error) {
			_, err := bytesBuffer.ReadFrom(readCloser)
			if err != nil {
				return false, err
			}
			err = streamJSONToSQS(jobSpec, uploader, bytesBuffer, jsonMessageList, &itemIndex)
			if err != nil {
				if err != io.ErrUnexpectedEOF || (err == io.ErrUnexpectedEOF && isLastChunk) {
					return false, err
				}
			}
			return true, nil
		})
		if err != nil {
			return false, errors.Wrap(err, s3Path)
		}

		return true, nil
	})
	if err != nil {
		return 0, err
	}

	if jsonMessageList.Length() != 0 {
		err := writeJSONObjectsToSQS(uploader, jsonMessageList)
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

func streamJSONToSQS(jobSpec *spec.Job, uploader *awslib.SQSBatchUploader, bytesBuffer *bytes.Buffer, jsonMessageList *jsonBuffer, itemIndex *int) error {
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
			return err
		}

		if len(doc) > awslib.MessageSizeLimit {
			return errors.Wrap(awslib.ErrorMessageExceedsMaxSize(len(doc), awslib.MessageSizeLimit), fmt.Sprintf("item %d", *itemIndex))
		}
		*itemIndex++
		jsonMessageList.Add(doc)
		if jsonMessageList.Length() == jsonMessageList.BatchSize {
			err := writeJSONObjectsToSQS(uploader, jsonMessageList)
			if err != nil {
				return err
			}
			jsonMessageList.Clear()

			if uploader.TotalBatches%10 == 0 {
				writeToJobLogGroup(jobSpec.JobKey, fmt.Sprintf("enqueued %d batches", uploader.TotalBatches))
			}
		}
	}

	return nil
}

func writeJSONObjectsToSQS(uploader *awslib.SQSBatchUploader, jsonMessageList *jsonBuffer) error {
	jsonBytes, err := json.Marshal(jsonMessageList.messageList)
	if err != nil {
		return err
	}
	randomID := k8s.RandomName()
	err = uploader.AddToBatch(&sqs.SendMessageBatchRequestEntry{
		Id:                     aws.String(randomID),
		MessageBody:            aws.String(string(jsonBytes)),
		MessageDeduplicationId: aws.String(randomID), // prevent content based deduping
		MessageGroupId:         aws.String(randomID), // aws recommends message group id per message to improve chances of exactly-once
	})
	if err != nil {
		return err
	}

	return nil
}
