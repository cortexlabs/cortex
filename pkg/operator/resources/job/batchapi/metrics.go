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

package batchapi

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
	_metricsRequestTimeoutSeconds = 10
	_completedMetricsFileKey      = "metrics.json"
)

func getBatchMetrics(jobKey spec.JobKey, t time.Time) (metrics.BatchMetrics, error) {
	var (
		jobBatchesSucceeded float64
		jobBatchesFailed    float64
		avgTimePerBatch     *float64
	)

	err := parallel.RunFirstErr(
		func() error {
			var err error
			jobBatchesSucceeded, err = getSucceededBatchesForJobMetric(config.Prometheus, jobKey, t)
			return err
		},
		func() error {
			var err error
			jobBatchesFailed, err = getFailedBatchesForJobMetric(config.Prometheus, jobKey, t)
			return err
		},
		func() error {
			var err error
			avgTimePerBatch, err = getAvgTimePerBatchMetric(config.Prometheus, jobKey, t)
			return err
		},
	)
	if err != nil {
		return metrics.BatchMetrics{}, err
	}

	return metrics.BatchMetrics{
		Succeeded:           int(jobBatchesSucceeded),
		Failed:              int(jobBatchesFailed),
		AverageTimePerBatch: avgTimePerBatch,
	}, nil
}

func getSucceededBatchesForJobMetric(promAPIv1 promv1.API, jobKey spec.JobKey, t time.Time) (float64, error) {
	query := fmt.Sprintf(
		"sum(cortex_batch_succeeded{api_name=\"%s\", job_id=\"%s\"})",
		jobKey.APIName, jobKey.ID,
	)

	values, err := queryPrometheusVec(promAPIv1, query, t)
	if err != nil {
		return 0, err
	}

	if values.Len() == 0 {
		return 0, nil
	}

	succeededBatches := float64(values[0].Value)
	return succeededBatches, nil
}

func getFailedBatchesForJobMetric(promAPIv1 promv1.API, jobKey spec.JobKey, t time.Time) (float64, error) {
	query := fmt.Sprintf(
		"sum(cortex_batch_failed{api_name=\"%s\", job_id=\"%s\"})",
		jobKey.APIName, jobKey.ID,
	)

	values, err := queryPrometheusVec(promAPIv1, query, t)
	if err != nil {
		return 0, err
	}

	if values.Len() == 0 {
		return 0, nil
	}

	failedBatches := float64(values[0].Value)
	return failedBatches, nil
}

func getAvgTimePerBatchMetric(promAPIv1 promv1.API, jobKey spec.JobKey, t time.Time) (*float64, error) {
	query := fmt.Sprintf(
		"sum(cortex_time_per_batch_sum{api_name=\"%s\", job_id=\"%s\"}) / sum(cortex_time_per_batch_count{api_name=\"%s\", job_id=\"%s\"})",
		jobKey.APIName, jobKey.ID,
		jobKey.APIName, jobKey.ID,
	)

	values, err := queryPrometheusVec(promAPIv1, query, t)
	if err != nil {
		return nil, err
	}

	if values.Len() == 0 {
		return nil, nil
	}

	avgTimePerBatch := float64(values[0].Value)
	return &avgTimePerBatch, nil
}

func queryPrometheusVec(promAPIv1 promv1.API, query string, t time.Time) (model.Vector, error) {
	ctx, cancel := context.WithTimeout(context.Background(), _metricsRequestTimeoutSeconds*time.Second)
	defer cancel()

	valuesQuery, err := promAPIv1.Query(ctx, query, t)
	if err != nil {
		return nil, err
	}

	values, ok := valuesQuery.(model.Vector)
	if !ok {
		return nil, errors.ErrorUnexpected("failed to convert metric to vector")
	}

	return values, nil
}

func saveMetricsToCloud(jobKey spec.JobKey) error {
	t := time.Now()
	batchMetrics, err := getBatchMetrics(jobKey, t)
	if err != nil {
		return err
	}

	s3Key := path.Join(jobKey.Prefix(config.ClusterName()), _completedMetricsFileKey)
	err = config.UploadJSONToBucket(batchMetrics, s3Key)
	if err != nil {
		return err
	}
	return nil
}

func readMetricsFromCloud(jobKey spec.JobKey) (metrics.BatchMetrics, error) {
	s3Key := path.Join(jobKey.Prefix(config.ClusterName()), _completedMetricsFileKey)
	batchMetrics := metrics.BatchMetrics{}
	err := config.ReadJSONFromBucket(&batchMetrics, s3Key)
	if err != nil {
		return batchMetrics, err
	}
	return batchMetrics, nil
}
