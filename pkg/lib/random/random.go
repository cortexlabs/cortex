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

package random

import (
	"math/rand"
	"time"
)

const (
	_uppercaseBytes = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	_lowercaseBytes = "abcdefghijklmnopqrstuvwxyz"
	_letterBytes    = _lowercaseBytes + _uppercaseBytes
	_numberBytes    = "0123456789"
	_stringBytes    = _letterBytes + _numberBytes

	_letterIdxBits = 6                     // 6 bits to represent a letter index
	_letterIdxMask = 1<<_letterIdxBits - 1 // All 1-bits, as many as _letterIdxBits
	_letterIdxMax  = 63 / _letterIdxBits   // # of letter indices fitting in 63 bits
)

func randomString(n int, src rand.Source, charset string) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for _letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), _letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), _letterIdxMax
		}
		if idx := int(cache & _letterIdxMask); idx < len(charset) {
			b[i] = charset[idx]
			i--
		}
		cache >>= _letterIdxBits
		remain--
	}

	return string(b)
}

// Digits generates a random string containing only digits
func Digits(n int) string {
	return randomString(n, rand.NewSource(time.Now().UnixNano()), _numberBytes)
}

// Letters generates a random string containing only english alphabet characters (upper and lower case)
func Letters(n int) string {
	return randomString(n, rand.NewSource(time.Now().UnixNano()), _letterBytes)
}

// LowercaseLetters generates a random string containing only lower case english alphabet characters
func LowercaseLetters(n int) string {
	return randomString(n, rand.NewSource(time.Now().UnixNano()), _lowercaseBytes)
}

// String generates a random string containing both digits and english alphabet characters (upper and lower)
func String(n int) string {
	return randomString(n, rand.NewSource(time.Now().UnixNano()), _stringBytes)
}

func LowercaseString(n int) string {
	return randomString(n, rand.NewSource(time.Now().UnixNano()), _lowercaseBytes+_numberBytes)
}
