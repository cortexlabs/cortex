/*
Copyright 2022 Cortex Labs, Inc.

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

package strings

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLongestCommonPrefix(t *testing.T) {
	var strs []string
	var expected string

	strs = []string{
		"12345",
	}
	expected = "12345"
	require.Equal(t, expected, LongestCommonPrefix(strs...))

	strs = []string{
		"12345",
		"12345678",
	}
	expected = "12345"
	require.Equal(t, expected, LongestCommonPrefix(strs...))

	strs = []string{
		"12345",
		"12345678",
		"1239",
	}
	expected = "123"
	require.Equal(t, expected, LongestCommonPrefix(strs...))

	strs = []string{
		"123",
		"456",
	}
	expected = ""
	require.Equal(t, expected, LongestCommonPrefix(strs...))
}

func TestRemoveDuplicates(t *testing.T) {
	cases := []struct {
		name        string
		input       []string
		prefixRegex *regexp.Regexp
		expected    []string
	}{
		{
			name:     "abc",
			input:    []string{"a", "b", "a", "b", "a", "a", "a", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "nil",
			input:    nil,
			expected: nil,
		},
		{
			name:        "eksctl",
			prefixRegex: regexp.MustCompile(`^.*[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} \[.+] {2}`),
			input: []string{
				"2021-03-26 00:03:50 [ℹ]  eksctl version 0.40.0",
				"2021-03-26 00:03:50 [ℹ]  using region us-east-1",
				"2021-03-26 00:03:50 [ℹ]  subnets for us-east-1a - public:192.168.0.0/19 private:192.168.64.0/19",
				"2021-03-26 00:03:50 [ℹ]  subnets for us-east-1b - public:192.168.32.0/19 private:192.168.96.0/19",
				"2021-03-26 00:03:50 [ℹ]  nodegroup \"cx-operator\" will use \"ami-05edded4121b6bde8\" [AmazonLinux2/1.18]",
				"2021-03-26 00:03:50 [ℹ]  nodegroup \"cx-ws-spot\" will use \"ami-05edded4121b6bde8\" [AmazonLinux2/1.18]",
				"2021-03-26 00:03:50 [ℹ]  nodegroup \"cx-wd-cpu\" will use \"ami-05edded4121b6bde8\" [AmazonLinux2/1.18]",
				"2021-03-26 00:03:51 [ℹ]  nodegroup \"cx-wd-gpu\" will use \"ami-00a430391abee258d\" [AmazonLinux2/1.18]",
				"2021-03-26 00:03:51 [ℹ]  nodegroup \"cx-wd-inferentia\" will use \"ami-00a430391abee258d\" [AmazonLinux2/1.18]",
				"2021-03-26 00:03:51 [ℹ]  using Kubernetes version 1.18",
				"2021-03-26 00:03:51 [ℹ]  creating EKS cluster \"cortex\" in \"us-east-1\" region with un-managed nodes",
				"2021-03-26 00:03:51 [ℹ]  5 nodegroups (cx-operator, cx-wd-cpu, cx-wd-gpu, cx-wd-inferentia, cx-ws-spot) were included (based on the include/exclude rules)",
				"2021-03-26 00:03:51 [ℹ]  will create a CloudFormation stack for cluster itself and 5 nodegroup stack(s)",
				"2021-03-26 00:03:51 [ℹ]  will create a CloudFormation stack for cluster itself and 0 managed nodegroup stack(s)",
				"2021-03-26 00:03:51 [ℹ]  if you encounter any issues, check CloudFormation console or try 'eksctl utils describe-stacks --region=us-east-1 --cluster=cortex'",
				"2021-03-26 00:03:51 [ℹ]  CloudWatch logging will not be enabled for cluster \"cortex\" in \"us-east-1\"",
				"2021-03-26 00:03:51 [ℹ]  you can enable it with 'eksctl utils update-cluster-logging --enable-types={SPECIFY-YOUR-LOG-TYPES-HERE (e.g. all)} --region=us-east-1 --cluster=cortex'",
				"2021-03-26 00:03:51 [ℹ]  Kubernetes API endpoint access will use default of {publicAccess=true, privateAccess=false} for cluster \"cortex\" in \"us-east-1\"",
				"2021-03-26 00:03:51 [ℹ]  2 sequential tasks: { create cluster control plane \"cortex\", 3 sequential sub-tasks: { 2 sequential sub-tasks: { wait for control plane to become ready, tag cluster }, create addons, 5 parallel sub-tasks: { create nodegroup \"cx-operator\", create nodegroup \"cx-ws-spot\", create nodegroup \"cx-wd-cpu\", create nodegroup \"cx-wd-gpu\", create nodegroup \"cx-wd-inferentia\" } } }",
				"2021-03-26 00:03:51 [ℹ]  building cluster stack \"eksctl-cortex-cluster\"",
				"2021-03-26 00:03:52 [ℹ]  deploying stack \"eksctl-cortex-cluster\"",
				"2021-03-26 00:03:52 [ℹ]  waiting for CloudFormation stack \"eksctl-cortex-cluster\"",
				"2021-03-26 00:04:09 [ℹ]  waiting for CloudFormation stack \"eksctl-cortex-cluster\"",
				"2021-03-26 00:04:28 [ℹ]  waiting for CloudFormation stack \"eksctl-cortex-cluster\"",
				"2021-03-26 00:04:47 [ℹ]  waiting for CloudFormation stack \"eksctl-cortex-cluster\"",
				"2021-03-26 00:05:07 [ℹ]  waiting for CloudFormation stack \"eksctl-cortex-cluster\"",
				"2021-03-26 00:05:25 [ℹ]  waiting for CloudFormation stack \"eksctl-cortex-cluster\"",
			},
			expected: []string{
				"2021-03-26 00:03:50 [ℹ]  eksctl version 0.40.0",
				"2021-03-26 00:03:50 [ℹ]  using region us-east-1",
				"2021-03-26 00:03:50 [ℹ]  subnets for us-east-1a - public:192.168.0.0/19 private:192.168.64.0/19",
				"2021-03-26 00:03:50 [ℹ]  subnets for us-east-1b - public:192.168.32.0/19 private:192.168.96.0/19",
				"2021-03-26 00:03:50 [ℹ]  nodegroup \"cx-operator\" will use \"ami-05edded4121b6bde8\" [AmazonLinux2/1.18]",
				"2021-03-26 00:03:50 [ℹ]  nodegroup \"cx-ws-spot\" will use \"ami-05edded4121b6bde8\" [AmazonLinux2/1.18]",
				"2021-03-26 00:03:50 [ℹ]  nodegroup \"cx-wd-cpu\" will use \"ami-05edded4121b6bde8\" [AmazonLinux2/1.18]",
				"2021-03-26 00:03:51 [ℹ]  nodegroup \"cx-wd-gpu\" will use \"ami-00a430391abee258d\" [AmazonLinux2/1.18]",
				"2021-03-26 00:03:51 [ℹ]  nodegroup \"cx-wd-inferentia\" will use \"ami-00a430391abee258d\" [AmazonLinux2/1.18]",
				"2021-03-26 00:03:51 [ℹ]  using Kubernetes version 1.18",
				"2021-03-26 00:03:51 [ℹ]  creating EKS cluster \"cortex\" in \"us-east-1\" region with un-managed nodes",
				"2021-03-26 00:03:51 [ℹ]  5 nodegroups (cx-operator, cx-wd-cpu, cx-wd-gpu, cx-wd-inferentia, cx-ws-spot) were included (based on the include/exclude rules)",
				"2021-03-26 00:03:51 [ℹ]  will create a CloudFormation stack for cluster itself and 5 nodegroup stack(s)",
				"2021-03-26 00:03:51 [ℹ]  will create a CloudFormation stack for cluster itself and 0 managed nodegroup stack(s)",
				"2021-03-26 00:03:51 [ℹ]  if you encounter any issues, check CloudFormation console or try 'eksctl utils describe-stacks --region=us-east-1 --cluster=cortex'",
				"2021-03-26 00:03:51 [ℹ]  CloudWatch logging will not be enabled for cluster \"cortex\" in \"us-east-1\"",
				"2021-03-26 00:03:51 [ℹ]  you can enable it with 'eksctl utils update-cluster-logging --enable-types={SPECIFY-YOUR-LOG-TYPES-HERE (e.g. all)} --region=us-east-1 --cluster=cortex'",
				"2021-03-26 00:03:51 [ℹ]  Kubernetes API endpoint access will use default of {publicAccess=true, privateAccess=false} for cluster \"cortex\" in \"us-east-1\"",
				"2021-03-26 00:03:51 [ℹ]  2 sequential tasks: { create cluster control plane \"cortex\", 3 sequential sub-tasks: { 2 sequential sub-tasks: { wait for control plane to become ready, tag cluster }, create addons, 5 parallel sub-tasks: { create nodegroup \"cx-operator\", create nodegroup \"cx-ws-spot\", create nodegroup \"cx-wd-cpu\", create nodegroup \"cx-wd-gpu\", create nodegroup \"cx-wd-inferentia\" } } }",
				"2021-03-26 00:03:51 [ℹ]  building cluster stack \"eksctl-cortex-cluster\"",
				"2021-03-26 00:03:52 [ℹ]  deploying stack \"eksctl-cortex-cluster\"",
				"2021-03-26 00:03:52 [ℹ]  waiting for CloudFormation stack \"eksctl-cortex-cluster\"",
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			output := RemoveDuplicates(tt.input, tt.prefixRegex)
			require.Equal(t, tt.expected, output)
		})
	}
}
