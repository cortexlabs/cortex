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
		Env: []string{"SERVICES=sqs"},
	}

	resource, err := pool.RunWithOptions(options)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
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

type mockHandler struct {
	HandleFunc func(message *sqs.Message) error
}

func (h *mockHandler) Handle(msg *sqs.Message) error {
	return h.HandleFunc(msg)
}

func testAWSClient(t *testing.T) *awslib.Client {
	t.Helper()

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Credentials: credentials.NewStaticCredentials("test", "test", ""),
			Endpoint:    aws.String(localStackEndpoint),
			Region:      aws.String(_localStackDefaultRegion), // localstack default region
			DisableSSL:  aws.Bool(true),
		},
	})
	require.NoError(t, err)

	return &awslib.Client{
		Sess:   sess,
		Region: _localStackDefaultRegion,
	}

}

func newLogger(t *testing.T) *zap.SugaredLogger {
	t.Helper()

	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.FatalLevel)
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

	dq, err := NewSQSDequeuer(
		SQSDequeuerConfig{
			Region:           _localStackDefaultRegion,
			QueueURL:         queueURL,
			StopIfNoMessages: true,
		}, awsClient, newLogger(t),
	)
	require.NoError(t, err)

	gotMessages, err := dq.ReceiveMessage()
	require.NoError(t, err)

	require.Len(t, gotMessages, 1)
	require.Equal(t, messageBody, *gotMessages[0].Body)
	require.Equal(t, sentMessage.MessageId, gotMessages[0].MessageId)
}

func TestSQSDequeuer_StartMessageRenewer(t *testing.T) {
	t.Parallel()

	awsClient := testAWSClient(t)
	queueURL := createQueue(t, awsClient)

	dq, err := NewSQSDequeuer(
		SQSDequeuerConfig{
			Region:           _localStackDefaultRegion,
			QueueURL:         queueURL,
			StopIfNoMessages: true,
		}, awsClient, newLogger(t),
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

	messages, err := dq.ReceiveMessage()
	done := dq.StartMessageRenewer(*messages[0].ReceiptHandle)
	defer func() {
		done <- struct{}{}
	}()

	require.Never(t, func() bool {
		msgs, err := dq.ReceiveMessage()
		require.NoError(t, err)

		return len(msgs) > 0
	}, time.Second, 10*time.Second)
}

func TestSQSDequeuerTerminationOnEmptyQueue(t *testing.T) {
	t.Parallel()

	awsClient := testAWSClient(t)
	queueURL := createQueue(t, awsClient)

	dq, err := NewSQSDequeuer(
		SQSDequeuerConfig{
			Region:           _localStackDefaultRegion,
			QueueURL:         queueURL,
			StopIfNoMessages: true,
		}, awsClient, newLogger(t),
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

	msgHandler := &mockHandler{
		HandleFunc: func(msg *sqs.Message) error {
			return nil
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- dq.Start(msgHandler)
	}()

	time.AfterFunc(10*time.Second, func() { errCh <- errors.New("timeout: dequeuer did not finish") })

	err = <-errCh
	require.NoError(t, err)
}
