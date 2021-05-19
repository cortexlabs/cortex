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

package dequeuer

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"go.uber.org/zap"
)

const (
	_maxNumberOfMessages = int64(1)
)

var (
	_messageAttributes = []string{"All"}
	_waitTimeSeconds   = 10 * time.Second
	_visibilityTimeout = 30 * time.Second
	_notFoundSleepTime = 10 * time.Second
	_renewalPeriod     = 10 * time.Second
)

type MessageHandler interface {
	Handle(*sqs.Message) error
}

type SQSHandlerConfig struct {
	Region           string
	QueueURL         string
	StopIfNoMessages bool
}

type SQSHandler struct {
	aws                *awslib.Client
	config             SQSHandlerConfig
	hasDeadLetterQueue bool
	waitTimeSeconds    *int64
	visibilityTimeout  *int64
	notFoundSleepTime  time.Duration
	renewalPeriod      time.Duration
	log                *zap.SugaredLogger
}

func NewSQSHandler(config SQSHandlerConfig, awsClient *awslib.Client, logger *zap.SugaredLogger) (*SQSHandler, error) {
	attr, err := GetQueueAttributes(awsClient, config.QueueURL)
	if err != nil {
		return nil, err
	}

	return &SQSHandler{
		aws:                awsClient,
		config:             config,
		hasDeadLetterQueue: attr.RedrivePolicy,
		waitTimeSeconds:    aws.Int64(int64(_waitTimeSeconds.Seconds())),
		visibilityTimeout:  aws.Int64(int64(_visibilityTimeout.Seconds())),
		notFoundSleepTime:  _notFoundSleepTime,
		renewalPeriod:      _renewalPeriod,
		log:                logger,
	}, nil
}

func (handler SQSHandler) ReceiveMessage() ([]*sqs.Message, error) {
	output, err := handler.aws.SQS().ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:              aws.String(handler.config.QueueURL),
		MaxNumberOfMessages:   aws.Int64(_maxNumberOfMessages),
		MessageAttributeNames: aws.StringSlice(_messageAttributes),
		VisibilityTimeout:     handler.visibilityTimeout,
		WaitTimeSeconds:       handler.waitTimeSeconds,
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return output.Messages, nil
}

func (handler SQSHandler) Start(messageHandler MessageHandler) error {
	noMessagesInPreviousIteration := false
	for true {
		messages, err := handler.ReceiveMessage()
		if err != nil {
			return err
		}

		if len(messages) == 0 {
			queueAttributes, err := GetQueueAttributes(handler.aws, handler.config.QueueURL)
			if err != nil {
				return err
			}

			if queueAttributes.TotalMessages() == 0 {
				if !noMessagesInPreviousIteration {
					handler.log.Info("no messages found in queue, retrying ...")
				}
				if noMessagesInPreviousIteration && handler.config.StopIfNoMessages {
					handler.log.Info("no messages found in queue, exiting ...")
					return nil
				}
				noMessagesInPreviousIteration = true
			}
			time.Sleep(handler.notFoundSleepTime)
			continue
		}

		noMessagesInPreviousIteration = false
		message := messages[0]
		receiptHandle := *message.ReceiptHandle
		done := handler.StartMessageRenewer(receiptHandle)
		handler.handleMessage(message, messageHandler, done)
	}

	return nil
}

func (handler SQSHandler) handleMessage(message *sqs.Message, messageHandler MessageHandler, done chan struct{}) {
	messageErr := messageHandler.Handle(message)
	// if messageErr != nil {
	// 	// TODO
	// }

	done <- struct{}{}
	isOnJobComplete := isOnJobCompleteMessage(message)

	if !isOnJobComplete && handler.hasDeadLetterQueue && messageErr != nil {
		// expire messages when head letter queue is configured to facilitate redrive policy
		// always delete onJobComplete messages regardless of dredrive policy because a new one will be added if an onJobComplete message has been consumed prematurely
		_, err := handler.aws.SQS().ChangeMessageVisibility(
			&sqs.ChangeMessageVisibilityInput{
				QueueUrl:          &handler.config.QueueURL,
				ReceiptHandle:     message.ReceiptHandle,
				VisibilityTimeout: aws.Int64(0),
			},
		)
		if err != nil {
			// TODO
			handler.log.Fatal(zap.Error(err))
		}
		return
	}

	_, err := handler.aws.SQS().DeleteMessage(
		&sqs.DeleteMessageInput{
			QueueUrl:      &handler.config.QueueURL,
			ReceiptHandle: message.ReceiptHandle,
		},
	)
	if err != nil {
		// TODO
		handler.log.Info(zap.Error(err))
	}
}

func (handler SQSHandler) StartMessageRenewer(receiptHandle string) chan struct{} {
	done := make(chan struct{})
	ticker := time.NewTicker(handler.renewalPeriod)
	startTime := time.Now()
	go func() {
		defer ticker.Stop()
		for true {
			select {
			case <-done:
				return
			case tickerTime := <-ticker.C:
				newVisibilityTimeout := tickerTime.Sub(startTime) + handler.renewalPeriod
				_, err := handler.aws.SQS().ChangeMessageVisibility(
					&sqs.ChangeMessageVisibilityInput{
						QueueUrl:          &handler.config.QueueURL,
						ReceiptHandle:     &receiptHandle,
						VisibilityTimeout: aws.Int64(int64(newVisibilityTimeout.Seconds())),
					},
				)
				if err != nil {
					// TODO err
					handler.log.Info(zap.Error(err))
				}
			}
		}
	}()
	return done
}
