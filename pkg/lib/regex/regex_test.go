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

package regex

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type regexpMatch struct {
	input string
	match bool
	subs  []string
}

func TestHasLeadingWhitespace(t *testing.T) {
	testcases := []regexpMatch{
		{
			input: " test",
			match: true,
		},
		{
			input: "	test",
			match: true,
		},
		{
			input: "\ntest",
			match: true,
		},
		{
			input: " test ",
			match: true,
		},
		{
			input: "	test	",
			match: true,
		},
		{
			input: "\ntest\n",
			match: true,
		},
		{
			input: "test",
			match: false,
		},
		{
			input: "te st",
			match: false,
		},
		{
			input: "te	st",
			match: false,
		},
		{
			input: "te\nst",
			match: false,
		},
		{
			input: "_",
			match: false,
		},
		{
			input: "test ",
			match: false,
		},
		{
			input: "test	",
			match: false,
		},
		{
			input: "test\n",
			match: false,
		},
		{
			input: " ",
			match: true,
		},
		{
			input: "	",
			match: true,
		},
		{
			input: "\n",
			match: true,
		},
		{
			input: "",
			match: false,
		},
	}

	for i := range testcases {
		match := HasLeadingWhitespace(testcases[i].input)
		assert.Equal(t, testcases[i].match, match, "input: "+testcases[i].input)
	}
}

func TestHasTrailingWhitespace(t *testing.T) {
	testcases := []regexpMatch{
		{
			input: " test",
			match: false,
		},
		{
			input: "	test",
			match: false,
		},
		{
			input: "\ntest",
			match: false,
		},
		{
			input: " test ",
			match: true,
		},
		{
			input: "	test	",
			match: true,
		},
		{
			input: "\ntest\n",
			match: true,
		},
		{
			input: "test",
			match: false,
		},
		{
			input: "te st",
			match: false,
		},
		{
			input: "te	st",
			match: false,
		},
		{
			input: "te\nst",
			match: false,
		},
		{
			input: "_",
			match: false,
		},
		{
			input: "test ",
			match: true,
		},
		{
			input: "test	",
			match: true,
		},
		{
			input: "test\n",
			match: true,
		},
		{
			input: " ",
			match: true,
		},
		{
			input: "	",
			match: true,
		},
		{
			input: "\n",
			match: true,
		},
		{
			input: "",
			match: false,
		},
	}

	for i := range testcases {
		match := HasTrailingWhitespace(testcases[i].input)
		assert.Equal(t, testcases[i].match, match, "input: "+testcases[i].input)
	}
}

func TestAlphaNumericDashDotUnderscoreRegex(t *testing.T) {
	testcases := []regexpMatch{
		{
			input: "generic.package.com",
			match: true,
		},
		{
			input: "generic_package.com",
			match: true,
		},
		{
			input: "_value-123",
			match: true,
		},
		{
			input: "data.generic.package()",
			match: false,
		},
		{
			input: "!variable",
			match: false,
		},
		{
			input: "assignment=value",
			match: false,
		},
		{
			input: "LibrayPackage@model.net",
			match: false,
		},
		{
			input: "aaaabbbbcccABCDZX-123456789",
			match: true,
		},
	}

	for i := range testcases {
		match := _alphaNumericDashDotUnderscoreRegex.MatchString(testcases[i].input)
		if match != testcases[i].match {
			t.Errorf("No match for %q", testcases[i].input)
		}
	}
}

func TestAlphaNumericDashUnderscoreRegex(t *testing.T) {
	testcases := []regexpMatch{
		{
			input: "generic.package.com",
			match: false,
		},
		{
			input: "generic_package.com",
			match: false,
		},
		{
			input: "_value-123",
			match: true,
		},
		{
			input: "data.generic.package()",
			match: false,
		},
		{
			input: "!variable",
			match: false,
		},
		{
			input: "assignment=value",
			match: false,
		},
		{
			input: "LibrayPackage@model.net",
			match: false,
		},
		{
			input: "aaaabbbbcccABCDZX-123456789",
			match: true,
		},
		{
			input: "word1-word2_word3_word4",
			match: true,
		},
		{
			input: "____-----____",
			match: true,
		},
		{
			input: "(word)",
			match: false,
		},
	}

	for i := range testcases {
		match := _alphaNumericDashUnderscoreRegex.MatchString(testcases[i].input)
		if match != testcases[i].match {
			t.Errorf("No match for %q", testcases[i].input)
		}
	}
}

func TestValidDockerImage(t *testing.T) {
	testcases := []regexpMatch{
		{
			input: "",
			match: false,
		},
		{
			input: "short",
			match: true,
		},
		{
			input: "simple/name",
			match: true,
		},
		{
			input: "library/ubuntu",
			match: true,
		},
		{
			input: "library/ubuntu:latest",
			match: true,
		},
		{
			input: "docker/stevvooe/app",
			match: true,
		},
		{
			input: "aa/aa/aa/aa/aa/aa/aa/aa/aa/bb/bb/bb/bb/bb/bb",
			match: true,
		},
		{
			input: "aa/aa/bb/bb/bb",
			match: true,
		},
		{
			input: "a/a/a/a",
			match: true,
		},
		{
			input: "a/a/a/a/",
			match: false,
		},
		{
			input: "a//a/a",
			match: false,
		},
		{
			input: "a",
			match: true,
		},
		{
			input: "a/aa",
			match: true,
		},
		{
			input: "a/aa/a",
			match: true,
		},
		{
			input: "foo.com",
			match: true,
		},
		{
			input: "foo.com/",
			match: false,
		},
		{
			input: "foo.com:8080/bar",
			match: true,
		},
		{
			input: "foo.com:http/bar",
			match: false,
		},
		{
			input: "foo.com/bar",
			match: true,
		},
		{
			input: "foo.com/bar/baz",
			match: true,
		},
		{
			input: "localhost:8080/bar",
			match: true,
		},
		{
			input: "sub-dom1.foo.com/bar/baz/quux",
			match: true,
		},
		{
			input: "blog.foo.com/bar/baz",
			match: true,
		},
		{
			input: "a^a",
			match: false,
		},
		{
			input: "aa/asdf$$^/aa",
			match: false,
		},
		{
			input: "asdf$$^/aa",
			match: false,
		},
		{
			input: "aa-a/a",
			match: true,
		},
		{
			input: strings.Repeat("a/", 128) + "a",
			match: true,
		},
		{
			input: "a-/a/a/a",
			match: false,
		},
		{
			input: "foo.com/a-/a/a",
			match: false,
		},
		{
			input: "-foo/bar",
			match: false,
		},
		{
			input: "foo/bar-",
			match: false,
		},
		{
			input: "foo-/bar",
			match: false,
		},
		{
			input: "foo/-bar",
			match: false,
		},
		{
			input: "_foo/bar",
			match: false,
		},
		{
			input: "foo_bar",
			match: true,
		},
		{
			input: "foo_bar.com",
			match: true,
		},
		{
			input: "foo_bar.com:8080/app",
			match: false,
		},
		{
			input: "foo.com/foo_bar",
			match: true,
		},
		{
			input: "____/____",
			match: false,
		},
		{
			input: "_docker/_docker",
			match: false,
		},
		{
			input: "docker_/docker_",
			match: false,
		},
		{
			input: "b.gcr.io/test.example.com/my-app",
			match: true,
		},
		{
			input: "xn--n3h.com/myimage", // ☃.com in punycode
			match: true,
		},
		{
			input: "xn--7o8h.com/myimage", // 🐳.com in punycode
			match: true,
		},
		{
			input: "example.com/xn--7o8h.com/myimage", // 🐳.com in punycode
			match: true,
		},
		{
			input: "example.com/some_separator__underscore/myimage",
			match: true,
		},
		{
			input: "example.com/__underscore/myimage",
			match: false,
		},
		{
			input: "example.com/..dots/myimage",
			match: false,
		},
		{
			input: "example.com/.dots/myimage",
			match: false,
		},
		{
			input: "example.com/nodouble..dots/myimage",
			match: false,
		},
		{
			input: "example.com/nodouble..dots/myimage",
			match: false,
		},
		{
			input: "docker./docker",
			match: false,
		},
		{
			input: ".docker/docker",
			match: false,
		},
		{
			input: "docker-/docker",
			match: false,
		},
		{
			input: "-docker/docker",
			match: false,
		},
		{
			input: "do..cker/docker",
			match: false,
		},
		{
			input: "do__cker:8080/docker",
			match: false,
		},
		{
			input: "do__cker/docker",
			match: true,
		},
		{
			input: "b.gcr.io/test.example.com/my-app",
			match: true,
		},
		{
			input: "registry.io/foo/project--id.module--name.ver---sion--name",
			match: true,
		},
		{
			input: "registry.com:8080/myapp:tag",
			match: true,
			subs:  []string{"registry.com:8080/myapp", "tag", ""},
		},
		{
			input: "registry.com:8080/myapp@sha256:be178c0543eb17f5f3043021c9e5fcf30285e557a4fc309cce97ff9ca6182912",
			match: true,
			subs:  []string{"registry.com:8080/myapp", "", "sha256:be178c0543eb17f5f3043021c9e5fcf30285e557a4fc309cce97ff9ca6182912"},
		},
		{
			input: "registry.com:8080/myapp:tag2@sha256:be178c0543eb17f5f3043021c9e5fcf30285e557a4fc309cce97ff9ca6182912",
			match: true,
			subs:  []string{"registry.com:8080/myapp", "tag2", "sha256:be178c0543eb17f5f3043021c9e5fcf30285e557a4fc309cce97ff9ca6182912"},
		},
		{
			input: "registry.com:8080/myapp@sha256:badbadbadbad",
			match: false,
		},
		{
			input: "registry.com:8080/myapp:invalid~tag",
			match: false,
		},
		{
			input: "bad_hostname.com:8080/myapp:tag",
			match: false,
		},
		{
			input:// localhost treated as name, missing tag with 8080 as tag
			"localhost:8080@sha256:be178c0543eb17f5f3043021c9e5fcf30285e557a4fc309cce97ff9ca6182912",
			match: true,
			subs:  []string{"localhost", "8080", "sha256:be178c0543eb17f5f3043021c9e5fcf30285e557a4fc309cce97ff9ca6182912"},
		},
		{
			input: "localhost:8080/name@sha256:be178c0543eb17f5f3043021c9e5fcf30285e557a4fc309cce97ff9ca6182912",
			match: true,
			subs:  []string{"localhost:8080/name", "", "sha256:be178c0543eb17f5f3043021c9e5fcf30285e557a4fc309cce97ff9ca6182912"},
		},
		{
			input: "localhost:http/name@sha256:be178c0543eb17f5f3043021c9e5fcf30285e557a4fc309cce97ff9ca6182912",
			match: false,
		},
		{
			// localhost will be treated as an image name without a host
			input: "localhost@sha256:be178c0543eb17f5f3043021c9e5fcf30285e557a4fc309cce97ff9ca6182912",
			match: true,
			subs:  []string{"localhost", "", "sha256:be178c0543eb17f5f3043021c9e5fcf30285e557a4fc309cce97ff9ca6182912"},
		},
		{
			input: "registry.com:8080/myapp@bad",
			match: false,
		},
		{
			input: "registry.com:8080/myapp@2bad",
			match: false, // Support this as valid?
		},
		{
			input: "680880929103.dkr.ecr.eu-central-1.amazonaws.com/cortexlabs/python-predictor-cpu:latest",
			match: true,
		},
	}

	for i := range testcases {
		match := _dockerValidImage.MatchString(testcases[i].input)
		if match != testcases[i].match {
			t.Errorf("No match for %q", testcases[i].input)
		}
	}

}

func TestValidECR(t *testing.T) {
	testcases := []regexpMatch{
		{
			input: "",
			match: false,
		},
		{
			input: "library/ubuntu:latest",
			match: false,
		},
		{
			input: "registry.com:8080/myapp:tag",
			match: false,
		},
		{
			input: "680880929102.dkr.ecr.eu-central-1.amazonaws.com/cortexlabs/python-predictor-cpu:latest",
			match: true,
		},
		{
			input: "localhost@sha256:be178c0543eb17f5f3043021c9e5fcf30285e557a4fc309cce97ff9ca6182912",
			match: false,
		},
		{
			input: "680880929102.dkr.ecr.eu-central-1.com/cortexlabs/image",
			match: false,
		},
		{
			input: "680880929102.dkr.ecr.us-east-1.amazonaws.com/registry",
			match: true,
		},
		{
			input: "1234567.dkr.ecr.us-west-1.amazonaws.com/registry/image:123",
			match: true,
		},
		{
			input: "680880929102.dkr.ecr.us-east-1.amazonaws.com",
			match: true,
		},
	}

	for i := range testcases {
		match := _ecrPattern.MatchString(testcases[i].input)
		if match != testcases[i].match {
			t.Errorf("No match for %q", testcases[i].input)
		}
	}
}
