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

// CreateWorkload enqueues an async workload request and uploads the request payload to cloud storage
func (s *service) CreateWorkload(id string, payload io.Reader, contentType string) (string, error) {
	path := fmt.Sprintf("%s/%s/payload", s.workloadStoragePrefix(), id)
	s.logger.Debug("uploading payload", zap.String("contentType", contentType), zap.String("path", path))
	if err := s.storage.Upload(path, payload, contentType); err != nil {
		return "", err
	}

	s.logger.Debug("sending message to queue", zap.String("id", id))
	if err := s.queue.SendMessage(id, id); err != nil {
		return "", err
	}

	return id, nil
}

// GetWorkload retrieves the status and result, if available, of a given workload
func (s *service) GetWorkload(id string) (GetWorkloadResponse, error) {
	prefix := s.workloadStoragePrefix()

	// download workload status
	statusPath := fmt.Sprintf("%s/%s/status", prefix, id)
	statusBuf, err := s.storage.Download(statusPath)
	if err != nil {
		return GetWorkloadResponse{}, err
	}

	status := Status(statusBuf[:])
	switch status {
	case StatusFailed, StatusInProgress, StatusInQueue:
		return GetWorkloadResponse{
			ID:     id,
			Status: status,
		}, nil
	case StatusCompleted: // continues execution after switch/case, below
	default:
		return GetWorkloadResponse{}, fmt.Errorf("invalid workload status: %s", status)
	}

	// TODO: make parallel calls
	// attempt to download user result
	resultPath := fmt.Sprintf("%s/%s/result.json", prefix, id)
	resultBuf, err := s.storage.Download(resultPath)
	if err != nil {
		return GetWorkloadResponse{}, err
	}

	var userResponse UserResponse
	if err = json.Unmarshal(resultBuf, &userResponse); err != nil {
		return GetWorkloadResponse{}, err
	}

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

func (s *service) workloadStoragePrefix() string {
	return fmt.Sprintf("%s/apis/%s/workloads", s.clusterName, s.apiName)
}
