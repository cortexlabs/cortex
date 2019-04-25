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

package resource

import (
	"time"

	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
)

type BaseSavedStatus struct {
	ResourceID   string     `json:"resource_id"`
	ResourceType Type       `json:"resource_type"`
	WorkloadID   string     `json:"workload_id"`
	AppName      string     `json:"app_name"`
	Start        *time.Time `json:"start"`
	End          *time.Time `json:"end"`
}

type DataSavedStatus struct {
	BaseSavedStatus
	ExitCode DataExitCode `json:"exit_code"`
}

type APISavedStatus struct {
	BaseSavedStatus
	APIName string `json:"api_name"`
}

type DataExitCode string

const (
	ExitCodeDataSucceeded DataExitCode = "succeeded"
	ExitCodeDataFailed    DataExitCode = "failed"
	ExitCodeDataKilled    DataExitCode = "killed"
	ExitCodeDataOOM       DataExitCode = "oom"
)

func DataSavedStatusPtrsEqual(savedStatus *DataSavedStatus, savedStatus2 *DataSavedStatus) bool {
	if savedStatus == nil && savedStatus2 == nil {
		return true
	}
	if savedStatus == nil || savedStatus2 == nil {
		return false
	}
	return savedStatus.Equal(*savedStatus2)
}

func APISavedStatusPtrsEqual(savedStatus *APISavedStatus, savedStatus2 *APISavedStatus) bool {
	if savedStatus == nil && savedStatus2 == nil {
		return true
	}
	if savedStatus == nil || savedStatus2 == nil {
		return false
	}
	return savedStatus.Equal(*savedStatus2)
}

func (savedStatus *DataSavedStatus) Equal(savedStatus2 DataSavedStatus) bool {
	if !savedStatus.BaseSavedStatus.Equal(savedStatus2.BaseSavedStatus) {
		return false
	}
	if savedStatus.ExitCode != savedStatus2.ExitCode {
		return false
	}
	return true
}

func (savedStatus *APISavedStatus) Equal(savedStatus2 APISavedStatus) bool {
	if !savedStatus.BaseSavedStatus.Equal(savedStatus2.BaseSavedStatus) {
		return false
	}
	if savedStatus.APIName != savedStatus2.APIName {
		return false
	}
	return true
}

func (savedStatus *BaseSavedStatus) Equal(savedStatus2 BaseSavedStatus) bool {
	if savedStatus.ResourceID != savedStatus2.ResourceID {
		return false
	}
	if savedStatus.ResourceType != savedStatus2.ResourceType {
		return false
	}
	if savedStatus.WorkloadID != savedStatus2.WorkloadID {
		return false
	}
	if savedStatus.AppName != savedStatus2.AppName {
		return false
	}
	if !libtime.PtrsEqual(savedStatus.Start, savedStatus2.Start) {
		return false
	}
	if !libtime.PtrsEqual(savedStatus.End, savedStatus2.End) {
		return false
	}
	return true
}

func (savedStatus *BaseSavedStatus) Copy() *BaseSavedStatus {
	if savedStatus == nil {
		return nil
	}
	return &BaseSavedStatus{
		ResourceID:   savedStatus.ResourceID,
		ResourceType: savedStatus.ResourceType,
		WorkloadID:   savedStatus.WorkloadID,
		AppName:      savedStatus.AppName,
		Start:        libtime.CopyPtr(savedStatus.Start),
		End:          libtime.CopyPtr(savedStatus.End),
	}
}

func (savedStatus *DataSavedStatus) Copy() *DataSavedStatus {
	if savedStatus == nil {
		return nil
	}
	baseSavedStatus := &savedStatus.BaseSavedStatus
	return &DataSavedStatus{
		BaseSavedStatus: *baseSavedStatus.Copy(),
		ExitCode:        savedStatus.ExitCode,
	}
}

func (savedStatus *APISavedStatus) Copy() *APISavedStatus {
	if savedStatus == nil {
		return nil
	}
	baseSavedStatus := &savedStatus.BaseSavedStatus
	return &APISavedStatus{
		BaseSavedStatus: *baseSavedStatus.Copy(),
		APIName:         savedStatus.APIName,
	}
}
