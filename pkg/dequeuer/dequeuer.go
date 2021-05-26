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
	_waitTime          = 10 * time.Second
	_visibilityTimeout = 30 * time.Second
	_notFoundSleepTime = 10 * time.Second
	_renewalPeriod     = 10 * time.Second
)

type MessageHandler interface {
	Handle(*sqs.Message) error
}

type SQSDequeuerConfig struct {
	Region           string
	QueueURL         string
	StopIfNoMessages bool
}

type SQSDequeuer struct {
	aws                *awslib.Client
	config             SQSDequeuerConfig
	hasDeadLetterQueue bool
	waitTimeSeconds    *int64
	visibilityTimeout  *int64
	notFoundSleepTime  time.Duration
	renewalPeriod      time.Duration
	log                *zap.SugaredLogger
	done               chan struct{}
}

func NewSQSDequeuer(config SQSDequeuerConfig, awsClient *awslib.Client, logger *zap.SugaredLogger) (*SQSDequeuer, error) {
	attr, err := GetQueueAttributes(awsClient, config.QueueURL)
	if err != nil {
		return nil, err
	}

	return &SQSDequeuer{
		aws:                awsClient,
		config:             config,
		hasDeadLetterQueue: attr.RedrivePolicy,
		waitTimeSeconds:    aws.Int64(int64(_waitTime.Seconds())),
		visibilityTimeout:  aws.Int64(int64(_visibilityTimeout.Seconds())),
		notFoundSleepTime:  _notFoundSleepTime,
		renewalPeriod:      _renewalPeriod,
		log:                logger,
		done:               make(chan struct{}),
	}, nil
}

func (d *SQSDequeuer) ReceiveMessage() ([]*sqs.Message, error) {
	output, err := d.aws.SQS().ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:              aws.String(d.config.QueueURL),
		MaxNumberOfMessages:   aws.Int64(_maxNumberOfMessages),
		MessageAttributeNames: aws.StringSlice(_messageAttributes),
		VisibilityTimeout:     d.visibilityTimeout,
		WaitTimeSeconds:       d.waitTimeSeconds,
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return output.Messages, nil
}

func (d *SQSDequeuer) Start(messageHandler MessageHandler) error {
	noMessagesInPreviousIteration := false

loop:
	for {
		select {
		case <-d.done:
			break loop
		default:
			messages, err := d.ReceiveMessage()
			if err != nil {
				return err
			}

			if len(messages) == 0 {
				queueAttributes, err := GetQueueAttributes(d.aws, d.config.QueueURL)
				if err != nil {
					return err
				}

				if queueAttributes.TotalMessages() == 0 {
					if noMessagesInPreviousIteration && d.config.StopIfNoMessages {
						d.log.Info("no messages found in queue, exiting ...")
						return nil
					}
					noMessagesInPreviousIteration = true
				}
				time.Sleep(d.notFoundSleepTime)
				continue
			}

			noMessagesInPreviousIteration = false
			message := messages[0]
			receiptHandle := *message.ReceiptHandle
			done := d.StartMessageRenewer(receiptHandle)
			d.handleMessage(message, messageHandler, done)
		}
	}

	return nil
}

func (d *SQSDequeuer) Shutdown() {
	d.done <- struct{}{}
}

func (d *SQSDequeuer) handleMessage(message *sqs.Message, messageHandler MessageHandler, done chan struct{}) {
	err := messageHandler.Handle(message)
	if err != nil {
		d.log.Errorw("error during message handling", "error", err)
	}

	done <- struct{}{}
	isOnJobComplete := isOnJobCompleteMessage(message)

	if !isOnJobComplete && d.hasDeadLetterQueue && err != nil {
		// expire messages when head letter queue is configured to facilitate redrive policy
		// always delete onJobComplete messages regardless of dredrive policy because a new one will be added if an onJobComplete message has been consumed prematurely
		_, err = d.aws.SQS().ChangeMessageVisibility(
			&sqs.ChangeMessageVisibilityInput{
				QueueUrl:          &d.config.QueueURL,
				ReceiptHandle:     message.ReceiptHandle,
				VisibilityTimeout: aws.Int64(0),
			},
		)
		if err != nil {
			d.log.Errorw("failed to change sqs message visibility", "error", err)
		}
		return
	}

	_, err = d.aws.SQS().DeleteMessage(
		&sqs.DeleteMessageInput{
			QueueUrl:      &d.config.QueueURL,
			ReceiptHandle: message.ReceiptHandle,
		},
	)
	if err != nil {
		d.log.Errorw("failed to delete sqs message", "error", err)
	}
}

func (d *SQSDequeuer) StartMessageRenewer(receiptHandle string) chan struct{} {
	done := make(chan struct{})
	ticker := time.NewTicker(d.renewalPeriod)
	startTime := time.Now()
	go func() {
		defer ticker.Stop()
		for true {
			select {
			case <-done:
				return
			case tickerTime := <-ticker.C:
				newVisibilityTimeout := tickerTime.Sub(startTime) + d.renewalPeriod
				_, err := d.aws.SQS().ChangeMessageVisibility(
					&sqs.ChangeMessageVisibilityInput{
						QueueUrl:          &d.config.QueueURL,
						ReceiptHandle:     &receiptHandle,
						VisibilityTimeout: aws.Int64(int64(newVisibilityTimeout.Seconds())),
					},
				)
				if err != nil {
					d.log.Errorw("failed to renew message visibility timeout", "error", err)
				}
			}
		}
	}()
	return done
}
