/*
Copyright 2019 Cortex Labs, Inc.

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

package context

import (
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/msgpack"
)

func (ctx Context) ToMsgpackBytes() ([]byte, error) {
	return msgpack.Marshal(ctx)
}

func FromMsgpackBytes(b []byte) (*Context, error) {
	var ctx Context
	err := msgpack.Unmarshal(b, &ctx)
	if err != nil {
		return nil, err
	}
	return &ctx, nil
}

func (ctx Context) MarshalJSON() ([]byte, error) {
	msgpackBytes, err := ctx.ToMsgpackBytes()
	if err != nil {
		return nil, err
	}
	msgpackJSONBytes, err := json.Marshal(&msgpackBytes)
	if err != nil {
		return nil, err
	}
	return msgpackJSONBytes, nil
}

func (ctx *Context) UnmarshalJSON(b []byte) error {
	var msgpackBytes []byte
	if err := json.Unmarshal(b, &msgpackBytes); err != nil {
		return err
	}
	ctxPtr, err := FromMsgpackBytes(msgpackBytes)
	if err != nil {
		return err
	}
	*ctx = *ctxPtr
	return nil
}
