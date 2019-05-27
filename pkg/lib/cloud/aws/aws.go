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
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	sparkop "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/sts"
	corev1 "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/regex"
)

type Provider struct {
	Bucket               string
	Region               string
	logGroup             string
	awsAccountID         string
	hashedAccountID      string
	cloudWatchLogsClient *cloudwatchlogs.CloudWatchLogs
}

func (provider *Provider) OperatorBucketURL() *url.URL {
	return &url.URL{
		Scheme: "s3",
		Host:   provider.Bucket,
	}
}

func (provider *Provider) WorkloadBucketURL() *url.URL {
	return &url.URL{
		Scheme: "s3",
		Host:   provider.Bucket,
	}
}

func New(bucket string, region string, logGroup string) *Provider {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:     aws.String(region),
		DisableSSL: aws.Bool(false),
	}))

	response, err := sts.New(sess).GetCallerIdentity(nil)
	if err != nil {
		errors.Exit(err, ErrorAuth("AWS"))
	}
	awsAccountID := *response.Account
	hashedAccountID := hash.String(awsAccountID)

	return &Provider{
		Bucket:               bucket,
		Region:               region,
		awsAccountID:         awsAccountID,
		hashedAccountID:      hashedAccountID,
		cloudWatchLogsClient: cloudwatchlogs.New(sess),
	}
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

func (provider *Provider) GetLogs(prefix string) (string, error) {
	logGroupNamePrefix := "var.log.containers."
	ignoreLogStreamNameRegexes := []*regexp.Regexp{
		regexp.MustCompile(`-exec-[0-9]+`),
		regexp.MustCompile(`_spark-init-`),
		regexp.MustCompile(`_cortex_serve-`),
	}

	logStreamsOut, err := provider.cloudWatchLogsClient.DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		Limit:               aws.Int64(50),
		LogGroupName:        aws.String(provider.logGroup),
		LogStreamNamePrefix: aws.String(logGroupNamePrefix + prefix),
	})
	if err != nil {
		return "", errors.Wrap(err, "cloudwatch logs", prefix)
	}

	var allLogsBuf bytes.Buffer

	for i, logStream := range logStreamsOut.LogStreams {
		if !regex.MatchAnyRegex(*logStream.LogStreamName, ignoreLogStreamNameRegexes) {
			getLogEventsInput := &cloudwatchlogs.GetLogEventsInput{
				LogGroupName:  &provider.logGroup,
				LogStreamName: logStream.LogStreamName,
				StartFromHead: aws.Bool(true),
			}

			err := provider.cloudWatchLogsClient.GetLogEventsPages(getLogEventsInput, func(logEventsOut *cloudwatchlogs.GetLogEventsOutput, lastPage bool) bool {
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
				return "", errors.Wrap(err, "cloudwatch logs")
			}

			if i < len(logStreamsOut.LogStreams)-1 {
				allLogsBuf.WriteString("\n----------\n\n")
			}
		}
	}

	return allLogsBuf.String(), nil
}

func (provider *Provider) AuthUser(authHeader string) (bool, error) {
	parts := strings.Split(authHeader, "|")
	if len(parts) != 2 {
		return false, ErrorAuth("aws")
	}

	accessKeyID, secretAccessKey := parts[0], parts[1]

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(provider.Region),
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

	return *response.Account == provider.awsAccountID, nil
}

func (provider *Provider) HashedAccountID() string {
	return provider.hashedAccountID
}

func (provider *Provider) IsValidHadoopPath(path string) (bool, error) {
	return IsValidS3a(path)
}

func (provider *Provider) EnvCredentials() []corev1.EnvVar {
	envVars := []corev1.EnvVar{
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

	return envVars
}

func (provider *Provider) SparkEnvCredentials() map[string]sparkop.NameKey {
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

func (provider *Provider) StorageVolumes() []corev1.Volume {
	return []corev1.Volume{}
}

func (provider *Provider) StorageVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{}
}

func (provider *Provider) APIsBaseURL(kubernetes *k8s.Client) (string, error) {
	service, err := kubernetes.GetService("nginx-controller-apis")
	if err != nil {
		return "", err
	}
	if service == nil {
		return "", errors.New("ErrorCortexInstallationBroken()")
	}
	if len(service.Status.LoadBalancer.Ingress) == 0 {
		return "", errors.New("ErrorLoadBalancerInitializing()")
	}
	return "https://" + service.Status.LoadBalancer.Ingress[0].Hostname, nil
}

func IsValidS3a(path string) (bool, error) {
	pathURL, err := url.Parse(path)
	if err != nil {
		return false, err
	}

	return pathURL.Scheme == "s3a" && len(pathURL.Host) > 0 && len(pathURL.Path) > 0, nil
}

func IsValidS3(path string) (bool, error) {
	pathURL, err := url.Parse(path)
	if err != nil {
		return false, err
	}

	return pathURL.Scheme == "s3" && len(pathURL.Host) > 0 && len(pathURL.Path) > 0, nil
}
