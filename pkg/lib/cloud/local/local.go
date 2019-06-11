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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sparkop "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	corev1 "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/cloud"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/random"
	"github.com/cortexlabs/cortex/pkg/lib/regex"
)

type Cloud struct {
	hashedAccountID string
	config          *cloud.Config
	*cloud.Storage
}

// type Config struct {
// 	Bucket             string
// 	OperatorLocalMount *string
// 	OperatorHostIP     *string
// 	OperatorInCluster  bool
// }

func NewFromConfig(config *cloud.Config) (cloud.Client, error) {
	url := url.URL{
		Scheme: "file",
		Host:   "localhost",
		Path:   config.Bucket,
	}

	if !config.OperatorInCluster && config.OperatorLocalMount != nil {
		url.Path = filepath.Join(*config.OperatorLocalMount, config.Bucket)
	}

	ctx := context.Background()
	bucket, err := blob.OpenBucket(ctx, url.String())
	if err != nil {
		return nil, err
	}
	return &Cloud{
		hashedAccountID: random.String(63),
		config:          config,
		Storage: &cloud.Storage{
			Bucket: bucket,
		},
	}, nil
}

func (c *Cloud) operatorBucketURL() *url.URL {
	url := url.URL{
		Scheme: "file",
		Host:   "localhost",
		Path:   c.config.Bucket,
	}

	if !c.config.OperatorInCluster && c.config.OperatorLocalMount != nil {
		url.Path = filepath.Join(*c.config.OperatorLocalMount, c.config.Bucket)
	}
	return &url
}

type fluentdLog struct {
	Log    string `json:"log"`
	Stream string `json:"stream"`
}

func (c *Cloud) GetLogs(prefix string) (string, error) {
	baseDir := filepath.Join(c.operatorBucketURL().Path, consts.LocalLogsDir)
	files, err := ioutil.ReadDir(baseDir)
	if err != nil {
		return "", errors.Wrap(err, "unable to access log directory", baseDir)
	}
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
		f, err := os.Open(targetFile)
		if err != nil {
			return "", errors.Wrap(err, "unable to open file", targetFile)
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var log fluentdLog
			json.Unmarshal([]byte(scanner.Text()), &log)
			allLogsBuf.WriteString(log.Log)
		}
		if err := scanner.Err(); err != nil {
			return "", errors.Wrap(err, "failed to read file", targetFile)
		}
		if i < len(matchedFiles)-1 {
			allLogsBuf.WriteString("\n----------\n\n")
		}
	}
	return allLogsBuf.String(), nil
}

func (c *Cloud) AuthUser(authHeader string) (bool, error) {
	return true, nil
}

func (c *Cloud) HashedAccountID() string {
	return c.hashedAccountID
}

func (c *Cloud) IsValidHadoopPath(path string) (bool, error) {
	isS3a, err := cloud.IsValidS3aPath(path)
	if err != nil {
		return false, err
	}

	isLocal, err := cloud.IsValidLocalPath(path)
	if err != nil {
		return false, err
	}

	return (isLocal || isS3a), nil
}

func (c *Cloud) EnvCredentials() []corev1.EnvVar {
	return []corev1.EnvVar{}
}

func (c *Cloud) SparkEnvCredentials() map[string]sparkop.NameKey {
	return map[string]sparkop.NameKey{}
}

func (c *Cloud) StorageVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: consts.CortexSharedVolume,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: consts.CortexSharedVolumeClaim,
				},
			},
		},
	}
}

func (c *Cloud) StorageVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      consts.CortexSharedVolume,
			MountPath: c.config.Bucket,
		},
	}
}

func (c *Cloud) APIsBaseURL(k8sClient *k8s.Client) (string, error) {
	clusterHost, err := url.Parse(k8sClient.RestConfig.Host)
	if err != nil {
		return "", err
	}

	host := strings.Split(clusterHost.Host, ":")[0]

	if c.config.OperatorHostIP != nil {
		host = *c.config.OperatorHostIP
	}

	clusterHost.Scheme = "https"

	service, err := k8sClient.GetService("nginx-controller-apis")
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

func (c *Cloud) BucketPath(key string) string {
	return filepath.Join(c.config.Bucket, key)
}

func (c *Cloud) Close() {
	if c.Bucket != nil {
		c.Bucket.Close()
	}
}

func (c *Cloud) ExternalPrefixExists(path string) (bool, error) {
	cloudType, err := cloud.ProviderTypeFromPath(path)
	if err != nil {
		return false, err
	}

	pathURL, err := url.Parse(path)
	if err != nil {
		return false, err
	}

	switch cloudType {
	case cloud.AWSProviderType:
		sess := session.Must(session.NewSession(&aws.Config{
			Credentials: credentials.AnonymousCredentials,
			Region:      aws.String(consts.DefaultAWSRegion),
		}))

		out, err := s3.New(sess).ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket: aws.String(pathURL.Host),
			Prefix: aws.String(pathURL.Path),
		})
		if err != nil {
			return false, errors.Wrap(err, path)
		}
		hasPrefix := *out.KeyCount > 0
		return hasPrefix, nil
	case cloud.LocalProviderType:
		if !c.config.OperatorInCluster {
			path = filepath.Join(*c.config.OperatorLocalMount, path)
		}
		return files.IsFileOrDir(path), nil
	default:
		return false, errors.Wrap(cloud.ErrorUnsupportedFilePath(path, "s3", "s3a", "local"))
	}
}
