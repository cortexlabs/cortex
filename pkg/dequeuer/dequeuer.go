/*
Copyright 2022 Cortex Labs, Inc.

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
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"go.uber.org/zap"
)

var (
	_messageAttributes  = []string{"All"}
	_waitTime           = 10 * time.Second
	_visibilityTimeout  = 30 * time.Second
	_notFoundSleepTime  = 10 * time.Second
	_renewalPeriod      = 10 * time.Second
	_probeRefreshPeriod = 1 * time.Second
)

type SQSDequeuerConfig struct {
	Region           string
	QueueURL         string
	StopIfNoMessages bool
	Workers          int
}

type SQSDequeuer struct {
	aws                *awslib.Client
	config             SQSDequeuerConfig
	hasDeadLetterQueue bool
	waitTimeSeconds    *int64
	visibilityTimeout  *int64
	notFoundSleepTime  time.Duration
	renewalPeriod      time.Duration
	probeRefreshPeriod time.Duration
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
		hasDeadLetterQueue: attr.HasRedrivePolicy,
		waitTimeSeconds:    aws.Int64(int64(_waitTime.Seconds())),
		visibilityTimeout:  aws.Int64(int64(_visibilityTimeout.Seconds())),
		notFoundSleepTime:  _notFoundSleepTime,
		renewalPeriod:      _renewalPeriod,
		probeRefreshPeriod: _probeRefreshPeriod,
		log:                logger,
		done:               make(chan struct{}),
	}, nil
}

func (d *SQSDequeuer) ReceiveMessage() (*sqs.Message, error) {
	output, err := d.aws.SQS().ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:              aws.String(d.config.QueueURL),
		MaxNumberOfMessages:   aws.Int64(1),
		MessageAttributeNames: aws.StringSlice(_messageAttributes),
		VisibilityTimeout:     d.visibilityTimeout,
		WaitTimeSeconds:       d.waitTimeSeconds,
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(output.Messages) == 0 {
		return nil, nil
	}

	return output.Messages[0], nil
}

func (d *SQSDequeuer) Start(messageHandler MessageHandler, readinessProbeFunc func() bool) error {
	numWorkers := libmath.MaxInt(d.config.Workers, 1)

	d.log.Infof("Starting %d workers", numWorkers)
	errCh := make(chan error)
	doneChs := make([]chan struct{}, d.config.Workers)
	for i := 0; i < numWorkers; i++ {
		doneChs[i] = make(chan struct{})
		go func(i int) {
			errCh <- d.worker(messageHandler, readinessProbeFunc, doneChs[i])
		}(i)
	}

	select {
	case err := <-errCh:
		return err
	case <-d.done:
		for _, doneCh := range doneChs {
			doneCh <- struct{}{}
		}
	}

	return nil
}

func (d SQSDequeuer) worker(messageHandler MessageHandler, readinessProbeFunc func() bool, workerDone chan struct{}) error {
	noMessagesInPreviousIteration := false

loop:
	for {
		select {
		case <-workerDone:
			break loop
		default:
			if !readinessProbeFunc() {
				time.Sleep(d.probeRefreshPeriod)
				continue
			}

			message, err := d.ReceiveMessage()
			if err != nil {
				return err
			}

			if message == nil { // no message received
				queueAttributes, err := GetQueueAttributes(d.aws, d.config.QueueURL)
				if err != nil {
					telemetry.Error(err)
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
			receiptHandle := *message.ReceiptHandle
			renewerDone := d.StartMessageRenewer(receiptHandle)
			err = d.handleMessage(message, messageHandler, renewerDone)
			if err != nil {
				d.log.Error(err)
				telemetry.Error(err)
			}
		}
	}

	return nil
}

func (d *SQSDequeuer) Shutdown() {
	d.done <- struct{}{}
}

func (d *SQSDequeuer) handleMessage(message *sqs.Message, messageHandler MessageHandler, done chan struct{}) error {
	messageErr := messageHandler.Handle(message) // handle error later

	done <- struct{}{}
	isOnJobComplete := isOnJobCompleteMessage(message)

	if messageErr != nil && d.hasDeadLetterQueue && !isOnJobComplete {
		// expire messages when dead letter queue is configured to facilitate redrive policy.
		// always delete onJobComplete messages regardless of redrive policy because a new one will
		// be added if an onJobComplete message has been consumed prematurely
		_, err := d.aws.SQS().ChangeMessageVisibility(
			&sqs.ChangeMessageVisibilityInput{
				QueueUrl:          &d.config.QueueURL,
				ReceiptHandle:     message.ReceiptHandle,
				VisibilityTimeout: aws.Int64(0),
			},
		)
		if err != nil {
			return errors.Wrap(err, "failed to change sqs message visibility")
		}
		return nil
	}

	_, err := d.aws.SQS().DeleteMessage(
		&sqs.DeleteMessageInput{
			QueueUrl:      &d.config.QueueURL,
			ReceiptHandle: message.ReceiptHandle,
		},
	)
	if err != nil {
		return errors.Wrap(err, "failed to delete sqs message")
	}

	if messageErr != nil {
		return messageErr
	}

	return nil
}

func (d *SQSDequeuer) StartMessageRenewer(receiptHandle string) chan struct{} {
	done := make(chan struct{})
	ticker := time.NewTicker(d.renewalPeriod)
	startTime := time.Now()
	go func() {
		defer ticker.Stop()
		for {
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
