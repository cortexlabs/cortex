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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/random"
	"go.uber.org/zap"
)

const (
	_defaultCortexVersion = "master"
	_batchKind            = "BatchAPI"
	_s3DownloadChunkSize  = 32 * 1024 * 1024
)

type ClusterEnv struct {
	ClusterName string
	Region      string
	Version     string
	Bucket      string
	APIName     string
	JobID       string
}

// FIXME: all these types should be shared with the cortex webserver (from where the payload is submitted)

type ItemList struct {
	Items     []json.RawMessage `json:"items"`
	BatchSize int               `json:"batch_size"`
}

type S3Lister struct {
	S3Paths  []string `json:"s3_paths"` // s3://<bucket_name>/key
	Includes []string `json:"includes"`
	Excludes []string `json:"excludes"`
}

type FilePathLister struct {
	S3Lister
	BatchSize int `json:"batch_size"`
}

type DelimitedFiles struct {
	S3Lister
	BatchSize int `json:"batch_size"`
}

type JobSubmission struct {
	ItemList       *ItemList       `json:"item_list"`
	FilePathLister *FilePathLister `json:"file_path_lister"`
	DelimitedFiles *DelimitedFiles `json:"delimited_files"`
}

func randomMessageID() string {
	return random.String(40) // maximum is 80 (for sqs.SendMessageBatchRequestEntry.Id) but this ID may show up in a user error message
}

type Enqueuer struct {
	aws        *awslib.Client
	clusterEnv ClusterEnv
	queueURL   string
	logger     *zap.Logger
}

func NewEnqueuer(clusterEnv ClusterEnv, queueURL string, logger *zap.Logger) (*Enqueuer, error) {
	awsClient, err := awslib.NewForRegion(clusterEnv.Region)
	if err != nil {
		return nil, err
	}

	return &Enqueuer{
		aws:        awsClient,
		clusterEnv: clusterEnv,
		queueURL:   queueURL,
		logger:     logger,
	}, nil
}

func (e *Enqueuer) Enqueue() (int, error) {
	submission, err := e.getJobPayload()
	if err != nil {
		return 0, err
	}

	totalBatches := 0
	if submission.ItemList != nil {
		totalBatches, err = e.enqueueItems(submission.ItemList)
		if err != nil {
			return 0, err
		}
	} else if submission.FilePathLister != nil {
		totalBatches, err = e.enqueueS3Paths(submission.FilePathLister)
		if err != nil {
			return 0, err
		}
	} else if submission.DelimitedFiles != nil {
		totalBatches, err = e.enqueueS3FileContents(submission.DelimitedFiles)
		if err != nil {
			return 0, err
		}
	}

	randomID := randomMessageID()
	_, err = e.aws.SQS().SendMessage(&sqs.SendMessageInput{
		QueueUrl:               aws.String(e.queueURL),
		MessageBody:            aws.String("\"job_complete\""),
		MessageDeduplicationId: aws.String(randomID), // prevent content based deduping
		MessageGroupId:         aws.String(randomID), // aws recommends message group id per message to improve chances of exactly-once
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"job_complete": {
				DataType:    aws.String("String"),
				StringValue: aws.String("true"),
			},
			"api_name": {
				DataType:    aws.String("String"),
				StringValue: aws.String(e.clusterEnv.APIName),
			},
			"job_id": {
				DataType:    aws.String("String"),
				StringValue: aws.String(e.clusterEnv.JobID),
			},
		},
	})
	if err != nil {
		return 0, errors.Wrap(err, "failed to enqueue job_complete placeholder")
	}

	return totalBatches, nil
}

func (e *Enqueuer) getJobPayload() (JobSubmission, error) {
	// e.g. <cluster name>/jobs/<job_api_kind>/<cortex version>/<api_name>/<job_id>
	key := fmt.Sprintf(
		"%s/jobs/%s/%s/%s/%s/payload.json",
		e.clusterEnv.ClusterName, _batchKind, e.clusterEnv.Version, e.clusterEnv.APIName, e.clusterEnv.JobID,
	)

	buff := aws.WriteAtBuffer{}
	input := s3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(e.clusterEnv.Bucket),
	}

	if _, err := e.aws.S3Downloader().Download(&buff, &input); err != nil {
		return JobSubmission{}, err
	}

	var submission JobSubmission
	if err := json.Unmarshal(buff.Bytes(), &submission); err != nil {
		return JobSubmission{}, err
	}

	return submission, nil
}

func (e *Enqueuer) enqueueItems(itemList *ItemList) (int, error) {
	log := e.logger

	batchCount := len(itemList.Items) / itemList.BatchSize
	if len(itemList.Items)%itemList.BatchSize != 0 {
		batchCount++
	}

	log.Info(
		"partitioning items found in job submission into batches",
		zap.Int("numItems", len(itemList.Items)),
		zap.Int("batchCount", batchCount),
		zap.Int("batchSize", itemList.BatchSize),
	)

	uploader := newSQSBatchUploader(e.clusterEnv.APIName, e.clusterEnv.JobID, e.queueURL, e.aws.SQS())

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
			log.Info("enqueued batches", zap.Int("batchCount", uploader.TotalBatches))
		}
	}

	err := uploader.Flush()
	if err != nil {
		return 0, err
	}

	return uploader.TotalBatches, nil
}

func (e *Enqueuer) enqueueS3Paths(s3PathsLister *FilePathLister) (int, error) {
	log := e.logger

	var s3PathList []string
	uploader := newSQSBatchUploader(e.clusterEnv.APIName, e.clusterEnv.JobID, e.queueURL, e.aws.SQS())

	err := s3IteratorFromLister(e.aws, s3PathsLister.S3Lister, func(bucket string, s3Obj *s3.Object) (bool, error) {
		s3Path := awslib.S3Path(bucket, *s3Obj.Key)

		s3PathList = append(s3PathList, s3Path)
		if len(s3PathList) == s3PathsLister.BatchSize {
			err := addS3PathsToQueue(uploader, s3PathList)
			if err != nil {
				return false, err
			}
			s3PathList = nil

			if uploader.TotalBatches%100 == 0 {
				log.Info("enqueued batches", zap.Int("numBatches", uploader.TotalBatches))
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

func (e *Enqueuer) enqueueS3FileContents(delimitedFiles *DelimitedFiles) (int, error) {
	log := e.logger

	jsonMessageList := newJSONBuffer(delimitedFiles.BatchSize)
	uploader := newSQSBatchUploader(e.clusterEnv.APIName, e.clusterEnv.JobID, e.queueURL, e.aws.SQS())

	bytesBuffer := bytes.NewBuffer([]byte{})
	err := s3IteratorFromLister(e.aws, delimitedFiles.S3Lister, func(bucket string, s3Obj *s3.Object) (bool, error) {
		s3Path := awslib.S3Path(bucket, *s3Obj.Key)
		log.Info("enqueuing contents from file", zap.String("path", s3Path))

		awsClientForBucket, err := awslib.NewFromClientS3Path(s3Path, e.aws)
		if err != nil {
			return false, err
		}

		itemIndex := 0
		err = awsClientForBucket.S3FileIterator(bucket, s3Obj, _s3DownloadChunkSize, func(readCloser io.ReadCloser, isLastChunk bool) (bool, error) {
			_, err := bytesBuffer.ReadFrom(readCloser)
			if err != nil {
				return false, err
			}
			err = e.streamJSONToQueue(uploader, bytesBuffer, jsonMessageList, &itemIndex)
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

func (e *Enqueuer) streamJSONToQueue(uploader *sqsBatchUploader, bytesBuffer *bytes.Buffer, jsonMessageList *jsonBuffer, itemIndex *int) error {
	log := e.logger

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
				log.Info("enqueued batches", zap.Int("numBatches", uploader.TotalBatches))
			}
		}
	}

	return nil
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
