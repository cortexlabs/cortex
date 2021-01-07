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

package errors

func AddError(errs []error, err error, strs ...string) ([]error, bool) {
	ok := false
	if err != nil {
		errs = append(errs, Wrap(err, strs...))
		ok = true
	}
	return errs, ok
}

func AddErrors(errs []error, newErrs []error, strs ...string) ([]error, bool) {
	ok := false
	for _, err := range newErrs {
		if err != nil {
			errs = append(errs, Wrap(err, strs...))
			ok = true
		}
	}
	return errs, ok
}

func WrapAll(errs []error, strs ...string) []error {
	if !HasError(errs) {
		return nil
	}
	wrappedErrs := make([]error, len(errs))
	for i, err := range errs {
		wrappedErrs[i] = Wrap(err, strs...)
	}
	return wrappedErrs
}

func HasError(errs []error) bool {
	for _, err := range errs {
		if err != nil {
			return true
		}
	}
	return false
}

func AreAllErrors(errs []error) bool {
	for _, err := range errs {
		if err == nil {
			return false
		}
	}
	return true
}

func FirstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func MapHasError(errs map[string]error) bool {
	for _, err := range errs {
		if err != nil {
			return true
		}
	}
	return false
}

func FirstErrorInMap(errs map[string]error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func FirstKeyInErrorMap(errs map[string]error) string {
	for k, err := range errs {
		if err != nil {
			return k
		}
	}
	return ""
}

func NonNilErrorMapKeys(errs map[string]error) []string {
	var keys []string
	for k, err := range errs {
		if err != nil {
			keys = append(keys, k)
		}
	}
	return keys
}
