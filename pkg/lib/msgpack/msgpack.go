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

package msgpack

import (
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/ugorji/go/codec"
)

var _mh codec.MsgpackHandle

func init() {
	_mh.RawToString = true
}

func Marshal(obj interface{}) ([]byte, error) {
	var bytes []byte
	enc := codec.NewEncoderBytes(&bytes, &_mh)
	err := enc.Encode(obj)
	if err != nil {
		return nil, errors.Wrap(err, errors.Message(ErrorMarshalMsgpack()))
	}
	return bytes, nil
}

func MustMarshal(obj interface{}) []byte {
	msgpackBytes, err := Marshal(obj)
	if err != nil {
		panic(err)
	}
	return msgpackBytes
}

func UnmarshalToInterface(b []byte) (interface{}, error) {
	var obj interface{}
	err := Unmarshal(b, &obj)
	if err != nil {
		return nil, errors.Wrap(err, errors.Message(ErrorUnmarshalMsgpack()))
	}
	return obj, nil
}

func Unmarshal(b []byte, obj interface{}) error {
	dec := codec.NewDecoderBytes(b, &_mh)
	return dec.Decode(&obj)
}
