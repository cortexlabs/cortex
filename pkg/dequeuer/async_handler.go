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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/async"
	"go.uber.org/zap"
)

const (
	// CortexRequestIDHeader is the header containing the workload request id for the user container
	CortexRequestIDHeader = "X-Cortex-Request-ID"
)

type AsyncMessageHandler struct {
	config       AsyncMessageHandlerConfig
	aws          *awslib.Client
	log          *zap.SugaredLogger
	storagePath  string
	httpClient   *http.Client
	eventHandler RequestEventHandler
}

type AsyncMessageHandlerConfig struct {
	ClusterUID string
	Bucket     string
	APIName    string
	TargetURL  string
}

func NewAsyncMessageHandler(config AsyncMessageHandlerConfig, awsClient *awslib.Client, eventHandler RequestEventHandler, logger *zap.SugaredLogger) *AsyncMessageHandler {
	return &AsyncMessageHandler{
		config:       config,
		aws:          awsClient,
		log:          logger,
		storagePath:  async.StoragePath(config.ClusterUID, config.APIName),
		httpClient:   &http.Client{},
		eventHandler: eventHandler,
	}
}

func (h *AsyncMessageHandler) Handle(message *sqs.Message) error {
	if message == nil {
		return errors.ErrorUnexpected("got unexpected nil SQS message")
	}

	if message.Body == nil || *message.Body == "" {
		return errors.ErrorUnexpected("got unexpected sqs message with empty or nil body")
	}

	requestID := *message.Body
	err := h.handleMessage(requestID)
	if err != nil {
		return err
	}
	return nil
}

func (h *AsyncMessageHandler) handleMessage(requestID string) error {
	h.log.Infow("processing workload", "id", requestID)

	err := h.updateStatus(requestID, async.StatusInProgress)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to update status to %s", async.StatusInProgress))
	}

	payload, err := h.getPayload(requestID)
	if err != nil {
		updateStatusErr := h.updateStatus(requestID, async.StatusFailed)
		if updateStatusErr != nil {
			h.log.Errorw("failed to update status after failure to get payload", "id", requestID, "error", updateStatusErr)
		}
		return errors.Wrap(err, "failed to get payload")
	}
	defer func() {
		h.deletePayload(requestID)
		_ = payload.Close()
	}()

	headers, err := h.getHeaders(requestID)
	if err != nil {
		updateStatusErr := h.updateStatus(requestID, async.StatusFailed)
		if updateStatusErr != nil {
			h.log.Errorw("failed to update status after failure to get headers", "id", requestID, "error", updateStatusErr)
		}
		return errors.Wrap(err, "failed to get payload")
	}

	result, err := h.submitRequest(payload, headers, requestID)
	if err != nil {
		h.log.Errorw("failed to submit request to user container", "id", requestID, "error", err)
		updateStatusErr := h.updateStatus(requestID, async.StatusFailed)
		if updateStatusErr != nil {
			return errors.Wrap(updateStatusErr, fmt.Sprintf("failed to update status to %s", async.StatusFailed))
		}
		return nil
	}

	if err = h.uploadResult(requestID, result); err != nil {
		updateStatusErr := h.updateStatus(requestID, async.StatusFailed)
		if updateStatusErr != nil {
			h.log.Errorw("failed to update status after failure to upload result", "id", requestID, "error", updateStatusErr)
		}
		return errors.Wrap(err, "failed to upload result to storage")
	}

	if err = h.updateStatus(requestID, async.StatusCompleted); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to update status to %s", async.StatusCompleted))
	}

	h.log.Infow("workload processing complete", "id", requestID)

	return nil
}

func (h *AsyncMessageHandler) updateStatus(requestID string, status async.Status) error {
	key := async.StatusPath(h.storagePath, requestID, status)
	return h.aws.UploadStringToS3("", h.config.Bucket, key)
}

func (h *AsyncMessageHandler) getPayload(requestID string) (io.ReadCloser, error) {
	key := async.PayloadPath(h.storagePath, requestID)
	output, err := h.aws.S3().GetObject(
		&s3.GetObjectInput{
			Key:    aws.String(key),
			Bucket: aws.String(h.config.Bucket),
		},
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return output.Body, nil
}

func (h *AsyncMessageHandler) deletePayload(requestID string) {
	key := async.PayloadPath(h.storagePath, requestID)
	err := h.aws.DeleteS3File(h.config.Bucket, key)
	if err != nil {
		h.log.Errorw("failed to delete user payload", "error", err)
		telemetry.Error(errors.Wrap(err, "failed to delete user payload"))
	}
}

func (h *AsyncMessageHandler) submitRequest(payload io.Reader, headers http.Header, requestID string) (interface{}, error) {
	req, err := http.NewRequest(http.MethodPost, h.config.TargetURL, payload)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req.Header = headers
	req.Header.Set(CortexRequestIDHeader, requestID)

	startTime := time.Now()
	response, err := h.httpClient.Do(req)
	if err != nil {
		return nil, ErrorUserContainerNotReachable(err)
	}

	defer func() {
		_ = response.Body.Close()
	}()

	h.eventHandler.HandleEvent(
		RequestEvent{
			StatusCode: response.StatusCode,
			Duration:   time.Since(startTime),
		},
	)

	if response.StatusCode != http.StatusOK {
		return nil, ErrorUserContainerResponseStatusCode(response.StatusCode)
	}

	if !strings.HasPrefix(response.Header.Get("Content-Type"), "application/json") {
		return nil, ErrorUserContainerResponseMissingJSONHeader()
	}

	var result interface{}
	if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, ErrorUserContainerResponseNotJSONDecodable()
	}

	return result, nil
}

func (h *AsyncMessageHandler) uploadResult(requestID string, result interface{}) error {
	key := async.ResultPath(h.storagePath, requestID)
	return h.aws.UploadJSONToS3(result, h.config.Bucket, key)
}

func (h *AsyncMessageHandler) getHeaders(requestID string) (http.Header, error) {
	key := async.HeadersPath(h.storagePath, requestID)

	var headers http.Header
	if err := h.aws.ReadJSONFromS3(&headers, h.config.Bucket, key); err != nil {
		return nil, err
	}

	return headers, nil
}
