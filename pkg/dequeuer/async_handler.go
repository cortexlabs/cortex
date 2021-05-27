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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"go.uber.org/zap"
)

const (
	// CortexRequestIDHeader is the header containing the workload request id for the user container
	CortexRequestIDHeader = "X-Cortex-Request-ID"
)

type AsyncMessageHandler struct {
	config      AsyncMessageHandlerConfig
	aws         *awslib.Client
	log         *zap.SugaredLogger
	storagePath string
}

type AsyncMessageHandlerConfig struct {
	ClusterUID string
	Bucket     string
	APIName    string
	TargetURL  string
}

type userPayload struct {
	Body        io.ReadCloser
	ContentType string
}

func NewAsyncMessageHandler(config AsyncMessageHandlerConfig, awsClient *awslib.Client, logger *zap.SugaredLogger) *AsyncMessageHandler {
	return &AsyncMessageHandler{
		config:      config,
		aws:         awsClient,
		log:         logger,
		storagePath: awslib.S3Path(config.Bucket, fmt.Sprintf("%s/workloads/%s", config.ClusterUID, config.APIName)),
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
		h.log.Errorw("failed processing request", "id", requestID, "error", err)
		err = h.handleFailure(requestID) // FIXME: should only handle failure if user error (?)
		if err != nil {
			return errors.Wrap(err, "failed to handle message failure")
		}
	}
	return nil
}

func (h *AsyncMessageHandler) handleMessage(requestID string) error {
	h.log.Infow("processing workload", "id", requestID)

	err := h.updateStatus(requestID, status.AsyncStatusInProgress)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to update status to %s", status.AsyncStatusInProgress))
	}

	payload, err := h.getPayload(requestID)
	if err != nil {
		return errors.Wrap(err, "failed to get payload")
	}
	defer h.deletePayload(requestID)

	result, err := h.submitRequest(payload, requestID)
	if err != nil {
		return errors.Wrap(err, "failed to submit request to user container")
	}

	if err = h.uploadResult(requestID, result); err != nil {
		return errors.Wrap(err, "failed to upload result to storage")
	}

	if err = h.updateStatus(requestID, status.AsyncStatusCompleted); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to update status to %s", status.AsyncStatusCompleted))
	}

	h.log.Infow("workload processing complete", "id", requestID)

	return nil
}

func (h *AsyncMessageHandler) handleFailure(requestID string) error {
	err := h.updateStatus(requestID, status.AsyncStatusFailed)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to update status to %s", status.AsyncStatusFailed))
	}
	return nil
}

func (h *AsyncMessageHandler) updateStatus(requestID string, status status.AsyncStatus) error {
	key := fmt.Sprintf("%s/%s/status/%s", h.storagePath, requestID, status)
	err := h.aws.UploadStringToS3("", h.config.Bucket, key)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (h *AsyncMessageHandler) getPayload(requestID string) (*userPayload, error) {
	key := fmt.Sprintf("%s/%s/payload", h.storagePath, requestID)
	output, err := h.aws.S3().GetObject(
		&s3.GetObjectInput{
			Key:    aws.String(key),
			Bucket: aws.String(h.config.Bucket),
		},
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	contentType := "application/octet-stream"
	if output.ContentType != nil {
		contentType = *output.ContentType
	}

	return &userPayload{
		Body:        output.Body,
		ContentType: contentType,
	}, nil
}

func (h *AsyncMessageHandler) deletePayload(requestID string) {
	key := fmt.Sprintf("%s/%s/payload", h.storagePath, requestID)
	err := h.aws.DeleteS3File(h.config.Bucket, key)
	if err != nil {
		h.log.Errorw("failed to delete user payload", "error", err)
		telemetry.Error(errors.Wrap(err, "failed to delete user payload"))
	}
}

func (h *AsyncMessageHandler) submitRequest(payload *userPayload, requestID string) (interface{}, error) {
	httpClient := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, h.config.TargetURL, payload.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req.Header.Set("Content-Type", payload.ContentType)
	req.Header.Set(CortexRequestIDHeader, requestID)
	response, err := httpClient.Do(req)
	if err != nil {
		return nil, ErrorUserContainerNotReachable(err)
	}

	defer func() {
		_ = response.Body.Close()
	}()

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
	key := fmt.Sprintf("%s/%s/result.json", h.storagePath, requestID)
	if err := h.aws.UploadJSONToS3(result, h.config.Bucket, key); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
