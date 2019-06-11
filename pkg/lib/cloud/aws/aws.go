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

package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	sparkop "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"gocloud.dev/blob/s3blob"
	corev1 "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/lib/cloud"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/regex"
)

type Cloud struct {
	awsAccountID         string
	hashedAccountID      string
	cloudWatchLogsClient *cloudwatchlogs.CloudWatchLogs
	config               *cloud.Config
	*cloud.Storage
}

// type Config struct {
// 	Bucket   string
// 	Region   string
// 	LogGroup string
// }

func NewFromConfig(config *cloud.Config) (cloud.Client, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:     aws.String(*config.Region),
		DisableSSL: aws.Bool(false),
	}))

	response, err := sts.New(sess).GetCallerIdentity(nil)
	if err != nil {
		errors.Exit(err, ErrorAuth("AWS"))
	}
	awsAccountID := *response.Account
	hashedAccountID := hash.String(awsAccountID)

	ctx := context.Background()
	bucket, err := s3blob.OpenBucket(ctx, sess, config.Bucket, nil)
	if err != nil {
		return nil, err
	}
	return &Cloud{
		awsAccountID:         awsAccountID,
		hashedAccountID:      hashedAccountID,
		cloudWatchLogsClient: cloudwatchlogs.New(sess),
		config:               config,
		Storage: &cloud.Storage{
			Bucket: bucket,
		},
	}, nil
}

type FluentdLog struct {
	Log    string `json:"log"`
	Stream string `json:"stream"`
	Docker struct {
		ContainerID string `json:"container_id"`
	} `json:"docker"`
	Kubernetes struct {
		ContainerName string `json:"container_name"`
		NamespaceName string `json:"namespace_name"`
		PodName       string `json:"pod_name"`
		OrphanedName  string `json:"orphaned_namespace"`
		NamespaceID   string `json:"namespace_id"`
	} `json:"kubernetes"`
}

func (c *Cloud) GetLogs(prefix string) (string, error) {
	logGroupNamePrefix := "var.log.containers."
	ignoreLogStreamNameRegexes := []*regexp.Regexp{
		regexp.MustCompile(`-exec-[0-9]+`),
		regexp.MustCompile(`_spark-init-`),
		regexp.MustCompile(`_cortex_serve-`),
	}

	logStreamsOut, err := c.cloudWatchLogsClient.DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		Limit:               aws.Int64(50),
		LogGroupName:        aws.String(*c.config.LogGroup),
		LogStreamNamePrefix: aws.String(logGroupNamePrefix + prefix),
	})
	if err != nil {
		return "", errors.Wrap(err, "cloudwatch logs", prefix)
	}

	var allLogsBuf bytes.Buffer

	for i, logStream := range logStreamsOut.LogStreams {
		if !regex.MatchAnyRegex(*logStream.LogStreamName, ignoreLogStreamNameRegexes) {
			getLogEventsInput := &cloudwatchlogs.GetLogEventsInput{
				LogGroupName:  c.config.LogGroup,
				LogStreamName: logStream.LogStreamName,
				StartFromHead: aws.Bool(true),
			}

			err := c.cloudWatchLogsClient.GetLogEventsPages(getLogEventsInput, func(logEventsOut *cloudwatchlogs.GetLogEventsOutput, lastPage bool) bool {
				for _, logEvent := range logEventsOut.Events {
					var log FluentdLog
					// millis := *logEvent.Timestamp
					// timestamp := time.Unix(0, millis*int64(time.Millisecond))
					json.Unmarshal([]byte(*logEvent.Message), &log)
					allLogsBuf.WriteString(log.Log)
				}
				return true
			})
			if err != nil {
				return "", errors.Wrap(err, "cloudwatch logs", prefix)
			}

			if i < len(logStreamsOut.LogStreams)-1 {
				allLogsBuf.WriteString("\n----------\n\n")
			}
		}
	}

	return allLogsBuf.String(), nil
}

func (c *Cloud) AuthUser(authHeader string) (bool, error) {
	parts := strings.Split(authHeader, "|")
	if len(parts) != 2 {
		return false, ErrorAuth("AWS")
	}

	accessKeyID, secretAccessKey := parts[0], parts[1]

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(*c.config.Region),
		DisableSSL:  aws.Bool(false),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return false, errors.WithStack(err)
	}
	userSTSClient := sts.New(sess)

	response, err := userSTSClient.GetCallerIdentity(nil)
	if awsErr, ok := err.(awserr.RequestFailure); ok {
		if awsErr.StatusCode() == 403 {
			return false, nil
		}
	}
	if err != nil {
		return false, errors.WithStack(err)
	}

	return *response.Account == c.awsAccountID, nil
}

func (c *Cloud) HashedAccountID() string {
	return c.hashedAccountID
}

func (c *Cloud) IsValidHadoopPath(path string) (bool, error) {
	return cloud.IsValidS3aPath(path)
}

func (c *Cloud) EnvCredentials() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "AWS_ACCESS_KEY_ID",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "aws-credentials",
					},
					Key: "AWS_ACCESS_KEY_ID",
				},
			},
		},
		{
			Name: "AWS_SECRET_ACCESS_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "aws-credentials",
					},
					Key: "AWS_SECRET_ACCESS_KEY",
				},
			},
		},
	}
}

func (c *Cloud) SparkEnvCredentials() map[string]sparkop.NameKey {
	return map[string]sparkop.NameKey{
		"AWS_ACCESS_KEY_ID": {
			Name: "aws-credentials",
			Key:  "AWS_ACCESS_KEY_ID",
		},
		"AWS_SECRET_ACCESS_KEY": {
			Name: "aws-credentials",
			Key:  "AWS_SECRET_ACCESS_KEY",
		}}
}

func (c *Cloud) StorageVolumes() []corev1.Volume {
	return []corev1.Volume{}
}

func (c *Cloud) StorageVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{}
}

func (c *Cloud) APIsBaseURL(kubernetes *k8s.Client) (string, error) {
	service, err := kubernetes.GetService("nginx-controller-apis")
	if err != nil {
		return "", err
	}
	if service == nil {
		return "", ErrorCortexInstallationBroken()
	}
	if len(service.Status.LoadBalancer.Ingress) == 0 {
		return "", ErrorLoadBalancerInitializing()
	}
	return "https://" + service.Status.LoadBalancer.Ingress[0].Hostname, nil
}

func (c *Cloud) BucketPath(key string) string {
	return fmt.Sprintf("s3://%s/%s", c.config.Bucket, key)
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

	if cloudType != cloud.AWSProviderType {
		return false, errors.Wrap(cloud.ErrorUnsupportedFilePath(path, "s3", "s3a"))
	}

	pathURL, err := url.Parse(path)
	if err != nil {
		return false, err
	}

	sess := session.Must(session.NewSession(&aws.Config{
		Region:     aws.String(*c.config.Region),
		DisableSSL: aws.Bool(false),
	}))
	fmt.Println(pathURL.Host)
	fmt.Println(pathURL.Path[1:])
	out, err := s3.New(sess).ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(pathURL.Host),
		Prefix: aws.String(pathURL.Path[1:]),
	})

	if err != nil {
		return false, errors.Wrap(err, path)
	}

	hasPrefix := *out.KeyCount > 0
	return hasPrefix, nil
}
