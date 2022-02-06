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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/random"
	"github.com/cortexlabs/cortex/pkg/types/async"
	"github.com/stretchr/testify/require"
)

const (
	_testBucket = "test"
)

func TestAsyncMessageHandler_Handle(t *testing.T) {
	t.Parallel()

	log := newLogger(t)
	defer func() { _ = log.Sync() }()

	awsClient := testAWSClient(t)

	requestID := random.String(8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, requestID, r.Header.Get(CortexRequestIDHeader))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))

	var requestEventsCount int
	eventHandler := NewRequestEventHandlerFunc(func(event RequestEvent) {
		requestEventsCount++
	})

	asyncHandler := NewAsyncMessageHandler(AsyncMessageHandlerConfig{
		ClusterUID: "cortex-test",
		Bucket:     _testBucket,
		APIName:    "async-test",
		TargetURL:  server.URL,
	}, awsClient, eventHandler, log)

	_, err := awsClient.S3().CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(_testBucket),
	})
	require.NoError(t, err)

	err = awsClient.UploadStringToS3("{}", asyncHandler.config.Bucket, async.PayloadPath(asyncHandler.storagePath, requestID))
	require.NoError(t, err)

	err = awsClient.UploadStringToS3("{}", asyncHandler.config.Bucket, async.HeadersPath(asyncHandler.storagePath, requestID))
	require.NoError(t, err)

	err = asyncHandler.Handle(&sqs.Message{
		Body:      aws.String(requestID),
		MessageId: aws.String(requestID),
	})
	require.NoError(t, err)

	_, err = awsClient.ReadStringFromS3(
		_testBucket,
		fmt.Sprintf("%s/%s/status/%s", asyncHandler.storagePath, requestID, async.StatusCompleted),
	)
	require.NoError(t, err)
	require.Equal(t, 1, requestEventsCount)
}

func TestAsyncMessageHandler_Handle_Errors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		message       *sqs.Message
		expectedError error
	}{
		{
			name:          "nil",
			message:       nil,
			expectedError: errors.ErrorUnexpected("got unexpected nil SQS message"),
		},
		{
			name:          "nil body",
			message:       &sqs.Message{},
			expectedError: errors.ErrorUnexpected("got unexpected sqs message with empty or nil body"),
		},
		{
			name:          "empty body",
			message:       &sqs.Message{Body: aws.String("")},
			expectedError: errors.ErrorUnexpected("got unexpected sqs message with empty or nil body"),
		},
	}

	log := newLogger(t)
	defer func() { _ = log.Sync() }()

	awsClient := testAWSClient(t)

	eventHandler := NewRequestEventHandlerFunc(func(event RequestEvent) {})

	asyncHandler := NewAsyncMessageHandler(AsyncMessageHandlerConfig{
		ClusterUID: "cortex-test",
		Bucket:     _testBucket,
		APIName:    "async-test",
		TargetURL:  "http://fake.cortex.dev",
	}, awsClient, eventHandler, log)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := asyncHandler.Handle(tt.message)
			require.EqualError(t, err, tt.expectedError.Error())
		})
	}
}
