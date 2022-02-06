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

package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/types/async"
	"go.uber.org/zap"
)

// Service provides an interface to the async-gateway business logic
type Service interface {
	CreateWorkload(id string, apiName string, queueURL string, payload io.Reader, headers http.Header) (string, error)
	GetWorkload(id string, apiName string) (GetWorkloadResponse, error)
}

type service struct {
	logger     *zap.SugaredLogger
	storage    Storage
	clusterUID string
	session    session.Session
}

// NewService creates a new async-gateway service
func NewService(clusterUID string, storage Storage, logger *zap.SugaredLogger, session session.Session) Service {
	return &service{
		logger:     logger,
		storage:    storage,
		clusterUID: clusterUID,
		session:    session,
	}
}

// CreateWorkload enqueues an async workload request and uploads the request payload to S3
func (s *service) CreateWorkload(id string, apiName string, queueURL string, payload io.Reader, headers http.Header) (string, error) {
	prefix := async.StoragePath(s.clusterUID, apiName)
	log := s.logger.With(zap.String("id", id), zap.String("apiName", apiName))

	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(headers); err != nil {
		return "", errors.Wrap(err, "failed to dump headers")
	}

	headersPath := async.HeadersPath(prefix, id)
	log.Debugw("uploading headers", zap.String("path", headersPath))
	if err := s.storage.Upload(headersPath, buf, "application/json"); err != nil {
		return "", errors.Wrap(err, "failed to upload headers")
	}

	contentType := headers.Get("Content-Type")
	payloadPath := async.PayloadPath(prefix, id)
	log.Debugw("uploading payload", zap.String("path", payloadPath))
	if err := s.storage.Upload(payloadPath, payload, contentType); err != nil {
		return "", errors.Wrap(err, "failed to upload payload")
	}

	log.Debug("sending message to queue")
	queue := NewSQS(queueURL, &s.session)
	if err := queue.SendMessage(id, id); err != nil {
		return "", errors.Wrap(err, "failed to send message to queue")
	}

	statusPath := fmt.Sprintf("%s/%s/status/%s", prefix, id, async.StatusInQueue)
	log.Debug(fmt.Sprintf("setting status to %s", async.StatusInQueue))
	if err := s.storage.Upload(statusPath, strings.NewReader(""), "text/plain"); err != nil {
		return "", errors.Wrap(err, "failed to upload workload status")
	}

	return id, nil
}

// GetWorkload retrieves the status and result, if available, of a given workload
func (s *service) GetWorkload(id string, apiName string) (GetWorkloadResponse, error) {
	log := s.logger.With(zap.String("id", id), zap.String("apiName", apiName))

	st, err := s.getStatus(id, apiName)
	if err != nil {
		return GetWorkloadResponse{}, err
	}

	if st != async.StatusCompleted {
		return GetWorkloadResponse{
			ID:     id,
			Status: st,
		}, nil
	}

	// attempt to download user result
	prefix := async.StoragePath(s.clusterUID, apiName)
	resultPath := async.ResultPath(prefix, id)
	log.Debug("downloading user result", zap.String("path", resultPath))
	resultBuf, err := s.storage.Download(resultPath)
	if err != nil {
		return GetWorkloadResponse{}, err
	}

	var userResponse UserResponse
	if err = json.Unmarshal(resultBuf, &userResponse); err != nil {
		return GetWorkloadResponse{}, err
	}

	log.Debug("getting workload timestamp")
	timestamp, err := s.storage.GetLastModified(resultPath)
	if err != nil {
		return GetWorkloadResponse{}, err
	}

	return GetWorkloadResponse{
		ID:        id,
		Status:    st,
		Result:    &userResponse,
		Timestamp: &timestamp,
	}, nil
}

func (s *service) getStatus(id string, apiName string) (async.Status, error) {
	prefix := async.StoragePath(s.clusterUID, apiName)
	log := s.logger.With(zap.String("id", id))

	// download workload status
	statusPrefixPath := async.StatusPrefixPath(prefix, id)
	log.Debug("checking status", zap.String("path", statusPrefixPath))
	files, err := s.storage.List(statusPrefixPath)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return async.StatusNotFound, nil
	}

	// determine request status
	st := async.StatusInQueue
	for _, file := range files {
		fileStatus := async.Status(file)
		if !fileStatus.Valid() {
			st = fileStatus
			return "", fmt.Errorf("invalid workload status: %s", st)
		}
		if fileStatus == async.StatusInProgress {
			st = fileStatus
		}
		if fileStatus == async.StatusCompleted || fileStatus == async.StatusFailed {
			st = fileStatus
			break
		}
	}

	return st, nil
}
