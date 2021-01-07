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

package cache

import (
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"gopkg.in/karalabe/cookiejar.v2/collections/deque"
)

type Fifo struct {
	size       int
	seen       strset.Set
	eventQueue *deque.Deque
}

func NewFifoCache(cacheSize int) Fifo {
	return Fifo{
		size:       cacheSize,
		seen:       strset.New(),
		eventQueue: deque.New(),
	}
}

func (c *Fifo) Has(eventID string) bool {
	return c.seen.Has(eventID)
}

func (c *Fifo) Add(eventID string) {
	if c.eventQueue.Size() == c.size {
		eventID := c.eventQueue.PopLeft().(string)
		c.seen.Remove(eventID)
	}
	c.seen.Add(eventID)
	c.eventQueue.PushRight(eventID)
}
