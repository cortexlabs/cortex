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

package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/cortexlabs/cortex/pkg/types/async"
	"go.uber.org/zap"
)

// Service provides an interface to the async-gateway business logic
type Service interface {
	CreateWorkload(id string, payload io.Reader, contentType string) (string, error)
	GetWorkload(id string) (GetWorkloadResponse, error)
}

type service struct {
	logger     *zap.SugaredLogger
	queue      Queue
	storage    Storage
	clusterUID string
	apiName    string
}

// NewService creates a new async-gateway service
func NewService(clusterUID, apiName string, queue Queue, storage Storage, logger *zap.SugaredLogger) Service {
	return &service{
		logger:     logger,
		queue:      queue,
		storage:    storage,
		clusterUID: clusterUID,
		apiName:    apiName,
	}
}

// CreateWorkload enqueues an async workload request and uploads the request payload to S3
func (s *service) CreateWorkload(id string, payload io.Reader, contentType string) (string, error) {
	prefix := async.StoragePath(s.clusterUID, s.apiName)
	log := s.logger.With(zap.String("id", id), zap.String("contentType", contentType))

	payloadPath := async.PayloadPath(prefix, id)
	log.Debug("uploading payload", zap.String("path", payloadPath))
	if err := s.storage.Upload(payloadPath, payload, contentType); err != nil {
		return "", err
	}

	log.Debug("sending message to queue")
	if err := s.queue.SendMessage(id, id); err != nil {
		return "", err
	}

	statusPath := fmt.Sprintf("%s/%s/status/%s", prefix, id, async.StatusInQueue)
	log.Debug(fmt.Sprintf("setting status to %s", async.StatusInQueue))
	if err := s.storage.Upload(statusPath, strings.NewReader(""), "text/plain"); err != nil {
		return "", err
	}

	return id, nil
}

// GetWorkload retrieves the status and result, if available, of a given workload
func (s *service) GetWorkload(id string) (GetWorkloadResponse, error) {
	log := s.logger.With(zap.String("id", id))

	st, err := s.getStatus(id)
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
	prefix := async.StoragePath(s.clusterUID, s.apiName)
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

func (s *service) getStatus(id string) (async.Status, error) {
	prefix := async.StoragePath(s.clusterUID, s.apiName)
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
