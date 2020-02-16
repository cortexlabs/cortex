/*
Copyright 2020 Cortex Labs, Inc.

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

package k8s

import (
	"fmt"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func GetAnnotation(obj kmeta.Object, key string) (string, error) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return "", errors.New("annotation not found: " + key)
	}
	val, ok := annotations[key]
	if !ok {
		return "", errors.New("annotation not found: " + key)
	}
	return val, nil
}

func ParseBoolAnnotation(obj kmeta.Object, key string) (bool, error) {
	val, err := GetAnnotation(obj, key)
	if err != nil {
		return false, err
	}
	casted, ok := s.ParseBool(val)
	if !ok {
		return false, errors.New(fmt.Sprintf("unable to parse %s from annotation %s as bool", val, key))
	}
	return casted, nil
}

func ParseIntAnnotation(obj kmeta.Object, key string) (int, error) {
	val, err := GetAnnotation(obj, key)
	if err != nil {
		return 0, err
	}
	casted, ok := s.ParseInt(val)
	if !ok {
		return 0, errors.New(fmt.Sprintf("unable to parse %s from annotation %s as int", val, key))
	}
	return casted, nil
}

func ParseInt32Annotation(obj kmeta.Object, key string) (int32, error) {
	val, err := GetAnnotation(obj, key)
	if err != nil {
		return 0, err
	}
	casted, ok := s.ParseInt32(val)
	if !ok {
		return 0, errors.New(fmt.Sprintf("unable to parse %s from annotation %s as int32", val, key))
	}
	return casted, nil
}

func ParseInt64Annotation(obj kmeta.Object, key string) (int64, error) {
	val, err := GetAnnotation(obj, key)
	if err != nil {
		return 0, err
	}
	casted, ok := s.ParseInt64(val)
	if !ok {
		return 0, errors.New(fmt.Sprintf("unable to parse %s from annotation %s as int64", val, key))
	}
	return casted, nil
}

func ParseFloat32Annotation(obj kmeta.Object, key string) (float32, error) {
	val, err := GetAnnotation(obj, key)
	if err != nil {
		return 0, err
	}
	casted, ok := s.ParseFloat32(val)
	if !ok {
		return 0, errors.New(fmt.Sprintf("unable to parse %s from annotation %s as float32", val, key))
	}
	return casted, nil
}

func ParseFloat64Annotation(obj kmeta.Object, key string) (float64, error) {
	val, err := GetAnnotation(obj, key)
	if err != nil {
		return 0, err
	}
	casted, ok := s.ParseFloat64(val)
	if !ok {
		return 0, errors.New(fmt.Sprintf("unable to parse %s from annotation %s as float64", val, key))
	}
	return casted, nil
}

func ParseDurationAnnotation(obj kmeta.Object, key string) (time.Duration, error) {
	val, err := GetAnnotation(obj, key)
	if err != nil {
		return 0, err
	}
	casted, err := time.ParseDuration(val)
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("unable to parse %s from annotation %s as duration", val, key))
	}
	return casted, nil
}
