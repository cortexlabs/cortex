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
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"
)

// Service provides an interface to the async-gateway business logic
type Service interface {
	CreateWorkload(id string, payload io.Reader, contentType string) (string, error)
	GetWorkload(id string) (GetWorkloadResponse, error)
}

type service struct {
	logger      *zap.Logger
	queue       Queue
	storage     Storage
	clusterName string
	apiName     string
}

// NewService creates a new async-gateway service
func NewService(clusterName, apiName string, queue Queue, storage Storage, logger *zap.Logger) Service {
	return &service{
		logger:      logger,
		queue:       queue,
		storage:     storage,
		clusterName: clusterName,
		apiName:     apiName,
	}
}

// CreateWorkload enqueues an async workload request and uploads the request payload to S3
func (s *service) CreateWorkload(id string, payload io.Reader, contentType string) (string, error) {
	prefix := s.workloadStoragePrefix()
	log := s.logger.With(zap.String("id", id), zap.String("contentType", contentType))

	payloadPath := fmt.Sprintf("%s/%s/payload", prefix, id)
	log.Debug("uploading payload", zap.String("path", payloadPath))
	if err := s.storage.Upload(payloadPath, payload, contentType); err != nil {
		return "", err
	}

	log.Debug("sending message to queue")
	if err := s.queue.SendMessage(id, id); err != nil {
		return "", err
	}

	statusPath := fmt.Sprintf("%s/%s/status/%s", prefix, id, StatusInQueue)
	log.Debug(fmt.Sprintf("setting status to %s", StatusInQueue))
	if err := s.storage.Upload(statusPath, strings.NewReader(""), "text/plain"); err != nil {
		return "", err
	}

	return id, nil
}

// GetWorkload retrieves the status and result, if available, of a given workload
func (s *service) GetWorkload(id string) (GetWorkloadResponse, error) {
	log := s.logger.With(zap.String("id", id))

	status, err := s.getStatus(id)
	if err != nil {
		return GetWorkloadResponse{}, err
	}

	if status != StatusCompleted {
		return GetWorkloadResponse{
			ID:     id,
			Status: status,
		}, nil
	}

	// attempt to download user result
	prefix := s.workloadStoragePrefix()
	resultPath := fmt.Sprintf("%s/%s/result.json", prefix, id)
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
		Status:    status,
		Result:    &userResponse,
		Timestamp: &timestamp,
	}, nil
}

func (s *service) getStatus(id string) (Status, error) {
	prefix := s.workloadStoragePrefix()
	log := s.logger.With(zap.String("id", id))

	// download workload status
	log.Debug("checking status", zap.String("path", fmt.Sprintf("%s/%s/status/*", prefix, id)))
	files, err := s.storage.List(fmt.Sprintf("%s/%s/status", prefix, id))
	if err != nil {
		return "", err
	}

	// determine request status
	status := StatusInQueue
	for _, file := range files {
		fileStatus := Status(file)
		if !fileStatus.Valid() {
			status = fileStatus
			return "", fmt.Errorf("invalid workload status: %s", status)
		}
		if fileStatus == StatusInProgress {
			status = fileStatus
		}
		if fileStatus == StatusCompleted || fileStatus == StatusFailed {
			status = fileStatus
			break
		}
	}

	return status, nil
}

func (s *service) workloadStoragePrefix() string {
	return fmt.Sprintf("%s/apis/%s/workloads", s.clusterName, s.apiName)
}
