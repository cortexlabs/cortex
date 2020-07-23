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
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

const MessageSizeLimit = 256 * 1024
const MaxMessagesPerBatch = 10
const CortexMessageSizeLimit = MessageSizeLimit - 2

type SQSBatchUploader struct {
	Client       *awslib.Client
	QueueURL     string
	Retries      *int // default 3 times
	messageList  []*sqs.SendMessageBatchRequestEntry
	startIndex   int
	endIndex     int
	totalBytes   int
	TotalBatches int
}

func (uploader *SQSBatchUploader) AddToBatch(id string, body *string) error {
	if len(*body) > MessageSizeLimit {
		return ErrorMessageExceedsMaxSize(len(*body), MessageSizeLimit)
	}

	message := &sqs.SendMessageBatchRequestEntry{
		Id:                     aws.String(id),
		MessageBody:            body,
		MessageDeduplicationId: aws.String(id), // prevent content based deduping
		MessageGroupId:         aws.String(id), // aws recommends message group id per message to improve chances of exactly-once
	}

	// TODO test what happens if messagebody exactly adds up to 256 KB
	if len(*message.MessageBody)+uploader.totalBytes > MessageSizeLimit || len(uploader.messageList) == MaxMessagesPerBatch {
		err := uploader.Flush()
		if err != nil {
			return err
		}

		uploader.startIndex = uploader.TotalBatches
	}

	uploader.endIndex = uploader.TotalBatches
	uploader.messageList = append(uploader.messageList, message)
	uploader.totalBytes += len(*message.MessageBody)
	uploader.TotalBatches++
	return nil
}

func (uploader *SQSBatchUploader) Flush() error {
	if len(uploader.messageList) == 0 {
		return nil
	}

	fmt.Println("flushing", uploader.TotalBatches)

	retries := 3
	if uploader.Retries != nil {
		retries = *uploader.Retries
	}

	var err error

	for retry := 0; retry < retries; retry++ {
		err = uploader.enqueueToSQS()
		if err == nil {
			uploader.messageList = nil
			uploader.startIndex = 0
			uploader.endIndex = 0
			uploader.totalBytes = 0
			return nil
		}
	}
	return errors.Wrap(err, fmt.Sprintf("failed after retrying %d times", retries))
}

func (uploader *SQSBatchUploader) enqueueToSQS() error {
	output, err := uploader.Client.SQS().SendMessageBatch(&sqs.SendMessageBatchInput{
		QueueUrl: aws.String(uploader.QueueURL),
		Entries:  uploader.messageList,
	})
	if err != nil {
		errorWrap := fmt.Sprintf("enqueuing batch %d", uploader.startIndex)
		if uploader.startIndex != uploader.endIndex {
			errorWrap = fmt.Sprintf("enqueuing batches between %d to %d", uploader.startIndex, uploader.endIndex)
		}

		if len(output.Failed) == 0 {
			return errors.Wrap(err, errorWrap)
		}

		return errors.Wrap(ErrorFailedToEnqueueMessages(*output.Failed[0].Message, uploader.startIndex, uploader.endIndex), errorWrap)
	}

	return nil
}
