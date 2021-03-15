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

// CreateWorkload enqueues an async workload request and uploads the request payload to cloud storage
func (s *service) CreateWorkload(id string, payload io.Reader, contentType string) (string, error) {
	prefix := s.workloadStoragePrefix()
	log := s.logger.With(zap.String("id", "id"), zap.String("contentType", contentType))

	payloadPath := fmt.Sprintf("%s/%s/payload", prefix, id)
	log.Debug("uploading payload", zap.String("path", payloadPath))
	if err := s.storage.Upload(payloadPath, payload, contentType); err != nil {
		return "", err
	}

	log.Debug("sending message to queue")
	if err := s.queue.SendMessage(id, id); err != nil {
		return "", err
	}

	statusPath := fmt.Sprintf("%s/%s/status", prefix, id)
	log.Debug(fmt.Sprintf("setting status to %s", StatusInQueue))
	if err := s.storage.Upload(statusPath, strings.NewReader(string(StatusInQueue)), "text/plain"); err != nil {
		return "", err
	}

	return id, nil
}

// GetWorkload retrieves the status and result, if available, of a given workload
func (s *service) GetWorkload(id string) (GetWorkloadResponse, error) {
	prefix := s.workloadStoragePrefix()
	log := s.logger.With(zap.String("id", id))

	// download workload status
	statusPath := fmt.Sprintf("%s/%s/status", prefix, id)
	log.Debug("downloading status file", zap.String("path", statusPath))
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

	// attempt to download user result
	resultPath := fmt.Sprintf("%s/%s/result.json", prefix, id)
	log.Debug("donwloading user result", zap.String("path", resultPath))
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

func (s *service) workloadStoragePrefix() string {
	return fmt.Sprintf("%s/apis/%s/workloads", s.clusterName, s.apiName)
}
