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

package util

import (
	"fmt"
	"strconv"
	"time"
)

func MicrosecsStr(t time.Time) string {
	nanos := fmt.Sprintf("%09d", t.Nanosecond())
	return nanos[0:6]
}

func MillisecsStr(t time.Time) string {
	nanos := fmt.Sprintf("%09d", t.Nanosecond())
	return nanos[0:3]
}

func Timestamp(t time.Time) string {
	microseconds := MicrosecsStr(t)
	return fmt.Sprintf("%d-%02d-%02d-%02d-%02d-%02d-%s",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), microseconds)
}

func NowTimestamp() string {
	return Timestamp(time.Now())
}

func TimePtrsEqual(t1 *time.Time, t2 *time.Time) bool {
	if t1 == nil && t2 == nil {
		return true
	}
	if t1 == nil || t2 == nil {
		return false
	}
	return t1.Equal(*t2)
}

func CopyTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	tCopy := *t
	return &tCopy
}

func TimeDifference(t1 *time.Time, t2 *time.Time) string {
	var duration time.Duration
	if t1 == nil && t2 == nil {
		return "-"
	} else if t1 == nil {
		return "infinity"
	} else if t2 == nil {
		duration = time.Since(*t1)
	} else {
		duration = (*t2).Sub(*t1)
	}

	durationSecs := int(duration.Seconds())
	if durationSecs < 60 {
		return strconv.Itoa(durationSecs) + "s"
	} else if durationSecs < 3600 {
		return strconv.Itoa(durationSecs/60) + "m"
	} else if durationSecs < 48*3600 {
		return strconv.Itoa(durationSecs/3600) + "h"
	} else {
		return strconv.Itoa(durationSecs/(24*3600)) + "d"
	}
}

func TimeSince(t *time.Time) string {
	if t == nil {
		return "-"
	}
	now := time.Now()
	return TimeDifference(t, &now)
}

func TimeNowPtr() *time.Time {
	now := time.Now()
	return &now
}

func LocalTimestamp(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return (*t).Local().Format("2006-01-02 15:04:05 MST")
}

func LocalTimestampHuman(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return (*t).Local().Format("Monday, January 2, 2006 at 3:04pm MST")
}

func OlderThanSeconds(t time.Time, secs float64) bool {
	return time.Since(t).Seconds() > secs
}
