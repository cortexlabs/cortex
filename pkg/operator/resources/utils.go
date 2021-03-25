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

package resources

import (
	"bytes"

	pbparser "github.com/emicklei/proto"
)

func isGrpcStreamingEnabledInProto(protoBytes []byte) (bool, error) {
	protoReader := bytes.NewReader(protoBytes)
	parser := pbparser.NewParser(protoReader)
	proto, err := parser.Parse()
	if err != nil {
		return false, err
	}

	numStreamingServices := 0
	pbparser.Walk(proto,
		pbparser.WithService(func(service *pbparser.Service) {
			for _, elem := range service.Elements {
				if s, ok := elem.(*pbparser.RPC); ok {
					if s.StreamsRequest || s.StreamsReturns {
						numStreamingServices++
					}
				}
			}
		}),
	)
	if numStreamingServices > 0 {
		return true, nil
	}
	return false, nil
}
