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
	"io"

	"go.uber.org/zap"
)

// Service provides an interface to the async-gateway business logic
type Service interface {
	CreateWorkload(id string, payload io.Reader, contentType string) (string, error)
	GetWorkload(id string) (GetWorkloadResponse, error)
}

type service struct {
	logger  *zap.Logger
	queue   Queue
	storage Storage
}

// NewService creates a new async-gateway service
func NewService(queue Queue, storage Storage, logger *zap.Logger) Service {
	return &service{
		logger:  logger,
		queue:   queue,
		storage: storage,
	}
}

// CreateWorkload enqueues an async workload request and uploads the request payload to cloud storage
func (s *service) CreateWorkload(id string, payload io.Reader, contentType string) (string, error) {
	s.logger.Debug("uploading payload", zap.Field{Key: "contentType", String: contentType})
	if err := s.storage.Upload(id, payload, contentType); err != nil {
		return "", err
	}

	s.logger.Debug("sending message to queue", zap.Field{Key: "id", String: id})
	if err := s.queue.SendMessage(id, id); err != nil {
		return "", err
	}

	return id, nil
}

// GetWorkload retrieves the status and result, if available, of a given workload
func (s *service) GetWorkload(id string) (GetWorkloadResponse, error) {
	panic("not implemented")
}
