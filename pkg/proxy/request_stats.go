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

package proxy

import "sync"

type RequestStats struct {
	sync.Mutex
	counts []int
}

func (s *RequestStats) Append(val int) {
	s.Lock()
	defer s.Unlock()
	s.counts = append(s.counts, val)
}

func (s *RequestStats) GetAllAndDelete() []int {
	var output []int
	s.Lock()
	defer s.Unlock()
	output = s.counts
	s.counts = []int{}
	return output
}

func (s *RequestStats) Report() float64 {
	requestCounts := s.GetAllAndDelete()

	total := 0.0
	if len(requestCounts) > 0 {
		for _, val := range requestCounts {
			total += float64(val)
		}

		total /= float64(len(requestCounts))
	}

	return total
}
