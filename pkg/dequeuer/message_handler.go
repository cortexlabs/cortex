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

import "github.com/aws/aws-sdk-go/service/sqs"

type MessageHandler interface {
	Handle(*sqs.Message) error
}

func NewMessageHandlerFunc(handleFunc func(*sqs.Message) error) MessageHandler {
	return &messageHandlerFunc{HandleFunc: handleFunc}
}

type messageHandlerFunc struct {
	HandleFunc func(message *sqs.Message) error
}

func (h *messageHandlerFunc) Handle(msg *sqs.Message) error {
	return h.HandleFunc(msg)
}
