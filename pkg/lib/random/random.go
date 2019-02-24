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

package random

import (
	"math/rand"
	"time"
)

const uppercaseBytes = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const lowercaseBytes = "abcdefghijklmnopqrstuvwxyz"
const letterBytes = lowercaseBytes + uppercaseBytes
const numberBytes = "0123456789"
const stringBytes = letterBytes + numberBytes

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func randomString(n int, src rand.Source, charset string) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(charset) {
			b[i] = charset[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

// Digits generates a random string containing only digits
func Digits(n int) string {
	return randomString(n, rand.NewSource(time.Now().UnixNano()), numberBytes)
}

// Letters generates a random string containing only english alphabet characters (upper and lower case)
func Letters(n int) string {
	return randomString(n, rand.NewSource(time.Now().UnixNano()), letterBytes)
}

// LowercaseLetters generates a random string containing only lower case english alphabet characters
func LowercaseLetters(n int) string {
	return randomString(n, rand.NewSource(time.Now().UnixNano()), lowercaseBytes)
}

// String generates a random string containing both digits and english alphabet characters (upper and lower)
func String(n int) string {
	return randomString(n, rand.NewSource(time.Now().UnixNano()), stringBytes)
}

func LowercaseString(n int) string {
	return randomString(n, rand.NewSource(time.Now().UnixNano()), lowercaseBytes+numberBytes)
}
