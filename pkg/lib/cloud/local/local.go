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

package local

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sparkop "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	corev1 "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/cloud/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/regex"
)

type Provider struct {
	accountID          string
	hashedAccountID    string
	baseDir            string
	operatorLocalMount *string
	operaterInCluster  bool
}

func New(baseDir string, operaterInCluster bool, operatorLocalMount *string) *Provider {
	return &Provider{
		accountID:          "abc",
		hashedAccountID:    "123",
		baseDir:            baseDir,
		operatorLocalMount: operatorLocalMount,
		operaterInCluster:  operaterInCluster,
	}
}

func (provider *Provider) OperatorBucketURL() *url.URL {
	url := url.URL{
		Scheme: "file",
		Host:   "localhost",
	}
	url.Path = provider.baseDir

	if !provider.operaterInCluster && provider.operatorLocalMount != nil {
		url.Path = filepath.Join(*provider.operatorLocalMount, provider.baseDir)
	}
	return &url
}

func (provider *Provider) WorkloadBucketURL() *url.URL {
	return &url.URL{
		Path: provider.baseDir,
	}
}

type fluentdLog struct {
	Log    string `json:"log"`
	Stream string `json:"stream"`
}

func (provider *Provider) GetLogs(prefix string) (string, error) {
	baseDir := filepath.Join(provider.OperatorBucketURL().Path, consts.LogsDir)
	files, _ := ioutil.ReadDir(baseDir)
	ignoreLogStreamNameRegexes := []*regexp.Regexp{
		regexp.MustCompile(`-exec-[0-9]+`),
		regexp.MustCompile(`_spark-init-`),
		regexp.MustCompile(`_cortex_serve-`),
	}

	var allLogsBuf bytes.Buffer

	matchedFiles := []string{}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), prefix) {
			filename := filepath.Join(baseDir, file.Name())
			if !regex.MatchAnyRegex(filename, ignoreLogStreamNameRegexes) {
				matchedFiles = append(matchedFiles, filename)
			}
		}
	}

	for i, targetFile := range matchedFiles {
		f, _ := os.Open(targetFile)
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var log fluentdLog
			json.Unmarshal([]byte(scanner.Text()), &log)
			allLogsBuf.WriteString(log.Log)
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}
		if i < len(matchedFiles)-1 {
			allLogsBuf.WriteString("\n----------\n\n")
		}
	}
	return allLogsBuf.String(), nil
}

func (provider *Provider) AuthUser(authHeader string) (bool, error) {
	return true, nil
}

func (provider *Provider) HashedAccountID() string {
	return provider.hashedAccountID
}

func (provider *Provider) IsValidHadoopPath(path string) (bool, error) {
	isS3a, err := aws.IsValidS3a(path)
	if err != nil {
		return false, err
	}

	isLocal, err := IsValidPath(path)

	return (isLocal || isS3a), err
}

func (provider *Provider) EnvCredentials() []corev1.EnvVar {
	return []corev1.EnvVar{}
}

func (provider *Provider) SparkEnvCredentials() map[string]sparkop.NameKey {
	return map[string]sparkop.NameKey{}
}

func (provider *Provider) StorageVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "task-pv-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "task-pv-claim",
				},
			},
		},
	}
}

func (provider *Provider) StorageVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "task-pv-storage",
			MountPath: provider.baseDir,
		},
	}
}

func (provider *Provider) APIsBaseURL(kubernetes *k8s.Client) (string, error) {
	clusterHost, err := url.Parse(kubernetes.RestConfig.Host)
	if err != nil {
		return "", err
	}

	host := strings.Split(clusterHost.Host, ":")[0]
	clusterHost.Scheme = "https"

	service, err := kubernetes.GetService("nginx-controller-apis")
	if err != nil {
		return "", err
	}

	for _, port := range service.Spec.Ports {
		if port.Name == clusterHost.Scheme {
			clusterHost.Host = fmt.Sprintf("%s:%d", host, port.NodePort)
			return clusterHost.String(), nil
		}
	}

	return "", errors.New("Port for HTTPS not found")
}

func IsValidPath(path string) (bool, error) {
	pathURL, err := url.Parse(path)
	if err != nil {
		return false, err
	}

	return len(pathURL.Scheme) == 0 && len(pathURL.Host) == 0 && len(pathURL.Path) > 0, nil
}
