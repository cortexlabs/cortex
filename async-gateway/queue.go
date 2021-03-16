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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awssqs "github.com/aws/aws-sdk-go/service/sqs"
)

// Queue is an interface to abstract communication with event queues
type Queue interface {
	SendMessage(message string, uniqueID string) error
}

type sqs struct {
	queueURL string
	client   *awssqs.SQS
}

// NewSQS creates a new SQS client that satisfies the Queue interface
func NewSQS(queueURL string, sess *session.Session) Queue {
	client := awssqs.New(sess)

	return &sqs{queueURL: queueURL, client: client}
}

// SendMessage sends a string
func (q *sqs) SendMessage(message string, uniqueID string) error {
	_, err := q.client.SendMessage(&awssqs.SendMessageInput{
		MessageBody:            aws.String(message),
		MessageDeduplicationId: aws.String(uniqueID),
		MessageGroupId:         aws.String(uniqueID),
		QueueUrl:               aws.String(q.queueURL),
	})
	return err
}
