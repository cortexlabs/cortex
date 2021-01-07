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

package time

import (
	"fmt"
	"strconv"
	"strings"
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

func PtrsEqual(t1 *time.Time, t2 *time.Time) bool {
	if t1 == nil && t2 == nil {
		return true
	}
	if t1 == nil || t2 == nil {
		return false
	}
	return t1.Equal(*t2)
}

func CopyPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	tCopy := *t
	return &tCopy
}

func DifferenceStr(t1 *time.Time, t2 *time.Time) string {
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
		return strconv.Itoa(durationSecs/60) + "m" + strconv.Itoa(durationSecs-durationSecs/60*60) + "s"
	} else if durationSecs < 48*3600 {
		return strconv.Itoa(durationSecs/3600) + "h" + strconv.Itoa((durationSecs-durationSecs/3600*3600)/60) + "m"
	} else {
		return strconv.Itoa(durationSecs/(24*3600)) + "d" + strconv.Itoa((durationSecs-durationSecs/(24*3600)*(24*3600))/3600) + "h"
	}
}

func SinceStr(t *time.Time) string {
	if t == nil {
		return "-"
	}
	now := time.Now()
	return DifferenceStr(t, &now)
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

func LocalHourNow() string {
	return time.Now().Local().Format("3:04:05pm MST")
}

func MillisToTime(epochMillis int64) time.Time {
	seconds := epochMillis / 1000
	millis := epochMillis % 1000
	return time.Unix(seconds, millis*int64(time.Millisecond))
}

func ToMillis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

type Timer struct {
	names []string
	start time.Time
	last  time.Time
}

func StartTimer(names ...string) Timer {
	return Timer{
		names: names,
		start: time.Now(),
	}
}

func (t *Timer) Print(messages ...string) {
	now := time.Now()

	separator := ""
	if len(t.names)+len(messages) > 0 {
		separator = ": "
	}

	totalTime := fmt.Sprintf("%s total", now.Sub(t.start))

	stepTime := ""
	if !t.last.IsZero() {
		stepTime = fmt.Sprintf("%s step, ", now.Sub(t.last))
	}

	fmt.Println(strings.Join(append(t.names, messages...), ": ") + separator + stepTime + totalTime)

	t.last = now
}

func MustParseDuration(str string) time.Duration {
	d, err := time.ParseDuration(str)
	if err != nil {
		panic(err)
	}
	return d
}

func MaxDuration(duration time.Duration, durations ...time.Duration) time.Duration {
	max := duration
	for _, d := range durations {
		if d > max {
			max = d
		}
	}

	return max
}
