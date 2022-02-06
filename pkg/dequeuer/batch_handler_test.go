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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/require"
)

func TestBatchMessageHandler_Handle(t *testing.T) {
	t.Parallel()
	awsClient := testAWSClient(t)

	var callCount int
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		}),
	)

	logger := newLogger(t)
	defer func() { _ = logger.Sync() }()

	batchHandler := NewBatchMessageHandler(BatchMessageHandlerConfig{
		APIName:   "test",
		JobID:     "12345",
		Region:    _localStackDefaultRegion,
		TargetURL: server.URL,
	}, awsClient, &statsd.NoOpClient{}, logger)

	err := batchHandler.Handle(&sqs.Message{
		Body:      aws.String(""),
		MessageId: aws.String("1"),
	})

	require.Equal(t, callCount, 1)
	require.NoError(t, err)
}

func TestBatchMessageHandler_Handle_OnJobComplete(t *testing.T) {
	t.Parallel()
	awsClient := testAWSClient(t)
	queueURL := createQueue(t, awsClient)

	var callCount int
	mux := http.NewServeMux()
	mux.HandleFunc("/on-job-complete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		callCount++
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)

	logger := newLogger(t)
	defer func() { _ = logger.Sync() }()

	batchHandler := NewBatchMessageHandler(BatchMessageHandlerConfig{
		APIName:   "test",
		JobID:     "12345",
		Region:    _localStackDefaultRegion,
		TargetURL: server.URL,
		QueueURL:  queueURL,
	}, awsClient, &statsd.NoOpClient{}, logger)

	batchHandler.jobCompleteMessageDelay = 0

	err := batchHandler.Handle(&sqs.Message{
		Body: aws.String("job_complete"),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"job_complete": {
				DataType:    aws.String("String"),
				StringValue: aws.String("true"),
			},
			"api_name": {
				DataType:    aws.String("String"),
				StringValue: aws.String("test"),
			},
			"job_id": {
				DataType:    aws.String("String"),
				StringValue: aws.String("12345"),
			},
		},
		MessageId: aws.String("00000"),
	})

	require.NoError(t, err)
	require.Equal(t, callCount, 1)
}
