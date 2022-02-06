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
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/random"
	"github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var localStackEndpoint string

const (
	_localStackDefaultRegion = "us-east-1"
)

func TestMain(m *testing.M) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	log.Println("Starting AWS localstack docker...")
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	options := &dockertest.RunOptions{
		Repository: "localstack/localstack",
		Tag:        "latest",
		PortBindings: map[dc.Port][]dc.PortBinding{
			"4566/tcp": {
				{HostPort: "4566"},
			},
		},
		Env: []string{"SERVICES=sqs,s3"},
	}

	resource, err := pool.RunWithOptions(options)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	err = resource.Expire(90)
	if err != nil {
		log.Fatal(err)
	}

	localStackEndpoint = fmt.Sprintf("localhost:%s", resource.GetPort("4566/tcp"))

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	// the minio client does not do service discovery for you (i.e. it does not check if connection can be established), so we have to use the health check
	if err := pool.Retry(func() error {
		url := fmt.Sprintf("http://%s/health", localStackEndpoint)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status code not OK")
		}
		return nil
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func testAWSClient(t *testing.T) *awslib.Client {
	t.Helper()

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Credentials:      credentials.NewStaticCredentials("test", "test", ""),
			Endpoint:         aws.String(localStackEndpoint),
			Region:           aws.String(_localStackDefaultRegion), // localstack default region
			DisableSSL:       aws.Bool(true),
			S3ForcePathStyle: aws.Bool(true),
		},
	})
	require.NoError(t, err)

	client, err := awslib.NewForSession(sess)
	require.NoError(t, err)

	return client
}

func newLogger(t *testing.T) *zap.SugaredLogger {
	t.Helper()

	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	logger, err := config.Build()
	require.NoError(t, err)

	logr := logger.Sugar()

	return logr
}

func createQueue(t *testing.T, awsClient *awslib.Client) string {
	t.Helper()

	createQueueOutput, err := awsClient.SQS().CreateQueue(
		&sqs.CreateQueueInput{
			QueueName: aws.String(fmt.Sprintf("test-%s.fifo", random.Digits(5))),
			Attributes: aws.StringMap(
				map[string]string{
					sqs.QueueAttributeNameFifoQueue:         "true",
					sqs.QueueAttributeNameVisibilityTimeout: "60",
				},
			),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, createQueueOutput.QueueUrl)
	require.NotEmpty(t, *createQueueOutput.QueueUrl)

	queueURL := *createQueueOutput.QueueUrl
	return queueURL
}

func TestSQSDequeuer_ReceiveMessage(t *testing.T) {
	t.Parallel()

	awsClient := testAWSClient(t)
	queueURL := createQueue(t, awsClient)

	messageID := "12345"
	messageBody := "blah"
	sentMessage, err := awsClient.SQS().SendMessage(&sqs.SendMessageInput{
		MessageBody:            aws.String(messageBody),
		MessageDeduplicationId: aws.String(messageID),
		MessageGroupId:         aws.String(messageID),
		QueueUrl:               aws.String(queueURL),
	})
	require.NoError(t, err)

	logger := newLogger(t)
	defer func() { _ = logger.Sync() }()

	dq, err := NewSQSDequeuer(
		SQSDequeuerConfig{
			Region:           _localStackDefaultRegion,
			QueueURL:         queueURL,
			StopIfNoMessages: true,
			Workers:          1,
		}, awsClient, logger,
	)
	require.NoError(t, err)

	gotMessage, err := dq.ReceiveMessage()
	require.NoError(t, err)

	require.NotNil(t, gotMessage)
	require.Equal(t, messageBody, *gotMessage.Body)
	require.Equal(t, *sentMessage.MessageId, *gotMessage.MessageId)
}

func TestSQSDequeuer_StartMessageRenewer(t *testing.T) {
	t.Parallel()

	awsClient := testAWSClient(t)
	queueURL := createQueue(t, awsClient)

	logger := newLogger(t)
	defer func() { _ = logger.Sync() }()

	dq, err := NewSQSDequeuer(
		SQSDequeuerConfig{
			Region:           _localStackDefaultRegion,
			QueueURL:         queueURL,
			StopIfNoMessages: true,
			Workers:          1,
		}, awsClient, logger,
	)
	require.NoError(t, err)

	dq.renewalPeriod = time.Second
	dq.visibilityTimeout = aws.Int64(2)

	messageID := "12345"
	messageBody := "blah"
	_, err = awsClient.SQS().SendMessage(&sqs.SendMessageInput{
		MessageBody:            aws.String(messageBody),
		MessageDeduplicationId: aws.String(messageID),
		MessageGroupId:         aws.String(messageID),
		QueueUrl:               aws.String(queueURL),
	})
	require.NoError(t, err)

	message, err := dq.ReceiveMessage()
	require.NoError(t, err)
	require.NotNil(t, message)

	done := dq.StartMessageRenewer(*message.ReceiptHandle)
	defer func() {
		done <- struct{}{}
	}()

	require.Never(t, func() bool {
		msg, err := dq.ReceiveMessage()
		require.NoError(t, err)

		return msg != nil
	}, time.Second, 10*time.Second)
}

func TestSQSDequeuerTerminationOnEmptyQueue(t *testing.T) {
	t.Parallel()

	awsClient := testAWSClient(t)
	queueURL := createQueue(t, awsClient)

	logger := newLogger(t)
	defer func() { _ = logger.Sync() }()

	dq, err := NewSQSDequeuer(
		SQSDequeuerConfig{
			Region:           _localStackDefaultRegion,
			QueueURL:         queueURL,
			StopIfNoMessages: true,
			Workers:          1,
		}, awsClient, logger,
	)
	require.NoError(t, err)

	dq.notFoundSleepTime = 0
	dq.waitTimeSeconds = aws.Int64(0)

	messageID := "12345"
	messageBody := "blah"
	_, err = awsClient.SQS().SendMessage(&sqs.SendMessageInput{
		MessageBody:            aws.String(messageBody),
		MessageDeduplicationId: aws.String(messageID),
		MessageGroupId:         aws.String(messageID),
		QueueUrl:               aws.String(queueURL),
	})
	require.NoError(t, err)

	msgHandler := &messageHandlerFunc{
		HandleFunc: func(msg *sqs.Message) error {
			return nil
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- dq.Start(msgHandler, func() bool {
			return true
		})
	}()

	time.AfterFunc(10*time.Second, func() { errCh <- errors.New("timeout: dequeuer did not finish") })

	err = <-errCh
	require.NoError(t, err)
}

func TestSQSDequeuer_Shutdown(t *testing.T) {
	t.Parallel()

	awsClient := testAWSClient(t)
	queueURL := createQueue(t, awsClient)

	logger := newLogger(t)
	defer func() { _ = logger.Sync() }()

	dq, err := NewSQSDequeuer(
		SQSDequeuerConfig{
			Region:           _localStackDefaultRegion,
			QueueURL:         queueURL,
			StopIfNoMessages: true,
			Workers:          1,
		}, awsClient, logger,
	)
	require.NoError(t, err)

	dq.notFoundSleepTime = 0
	dq.waitTimeSeconds = aws.Int64(0)

	msgHandler := NewMessageHandlerFunc(
		func(message *sqs.Message) error {
			return nil
		},
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- dq.Start(msgHandler, func() bool {
			return true
		})
	}()

	time.AfterFunc(5*time.Second, func() { errCh <- errors.New("timeout: dequeuer did not exit") })

	dq.Shutdown()

	err = <-errCh
	require.NoError(t, err)
}

func TestSQSDequeuer_Start_HandlerError(t *testing.T) {
	t.Parallel()

	awsClient := testAWSClient(t)
	queueURL := createQueue(t, awsClient)

	logger := newLogger(t)
	defer func() { _ = logger.Sync() }()

	dq, err := NewSQSDequeuer(
		SQSDequeuerConfig{
			Region:           _localStackDefaultRegion,
			QueueURL:         queueURL,
			StopIfNoMessages: true,
			Workers:          1,
		}, awsClient, logger,
	)
	require.NoError(t, err)

	dq.waitTimeSeconds = aws.Int64(0)
	dq.notFoundSleepTime = 0
	dq.renewalPeriod = 500 * time.Millisecond
	dq.visibilityTimeout = aws.Int64(1)

	msgHandler := NewMessageHandlerFunc(
		func(message *sqs.Message) error {
			return fmt.Errorf("an error occurred")
		},
	)

	messageID := "12345"
	messageBody := "blah"
	_, err = awsClient.SQS().SendMessage(&sqs.SendMessageInput{
		MessageBody:            aws.String(messageBody),
		MessageDeduplicationId: aws.String(messageID),
		MessageGroupId:         aws.String(messageID),
		QueueUrl:               aws.String(queueURL),
	})
	require.NoError(t, err)

	go func() {
		err := dq.Start(msgHandler, func() bool {
			return true
		})
		require.NoError(t, err)
	}()

	require.Never(t, func() bool {
		msg, err := dq.ReceiveMessage()
		require.NoError(t, err)
		return msg != nil
	}, 5*time.Second, time.Second)
}

// this test seems to be non-deterministically timing out
// it seems to be an issue with the test, not the deqeueur
// func TestSQSDequeuer_MultipleWorkers(t *testing.T) {
// 	t.Parallel()

// 	awsClient := testAWSClient(t)
// 	queueURL := createQueue(t, awsClient)

// 	numMessages := 3
// 	expectedMsgs := make([]string, numMessages)
// 	for i := 0; i < numMessages; i++ {
// 		message := fmt.Sprintf("%d", i)
// 		expectedMsgs[i] = message
// 		_, err := awsClient.SQS().SendMessage(&sqs.SendMessageInput{
// 			MessageBody:            aws.String(message),
// 			MessageDeduplicationId: aws.String(message),
// 			MessageGroupId:         aws.String(message),
// 			QueueUrl:               aws.String(queueURL),
// 		})
// 		require.NoError(t, err)
// 	}

// 	logger := newLogger(t)
// 	defer func() { _ = logger.Sync() }()

// 	dq, err := NewSQSDequeuer(
// 		SQSDequeuerConfig{
// 			Region:           _localStackDefaultRegion,
// 			QueueURL:         queueURL,
// 			StopIfNoMessages: true,
// 			Workers:          numMessages,
// 		}, awsClient, logger,
// 	)
// 	require.NoError(t, err)

// 	dq.waitTimeSeconds = aws.Int64(0)
// 	dq.notFoundSleepTime = 0

// 	msgCh := make(chan string, numMessages)
// 	handler := NewMessageHandlerFunc(
// 		func(message *sqs.Message) error {
// 			msgCh <- *message.Body
// 			return nil
// 		},
// 	)

// 	errCh := make(chan error)
// 	go func() {
// 		errCh <- dq.Start(handler, func() bool { return true })
// 	}()

// 	receivedMessages := make([]string, numMessages)
// 	for i := 0; i < numMessages; i++ {
// 		receivedMessages[i] = <-msgCh
// 	}
// 	dq.Shutdown()

// 	// timeout test after 30 seconds
// 	time.AfterFunc(30*time.Second, func() {
// 		close(msgCh)
// 		errCh <- errors.New("test timed out")
// 	})

// 	require.Len(t, receivedMessages, numMessages)

// 	set := strset.FromSlice(receivedMessages)
// 	require.True(t, set.Has(expectedMsgs...))

// 	require.NoError(t, <-errCh)
// }
