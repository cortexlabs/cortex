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

package spark

import (
	"strings"

	sparkop "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1alpha1"
	clientset "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/client/clientset/versioned"
	clientsettyped "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/client/clientset/versioned/typed/sparkoperator.k8s.io/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cortexlabs/cortex/pkg/api/context"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	cc "github.com/cortexlabs/cortex/pkg/operator/cortexconfig"
	"github.com/cortexlabs/cortex/pkg/operator/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
)

var sparkClientset clientset.Interface
var sparkClient clientsettyped.SparkApplicationInterface

var doneStates = []string{
	string(sparkop.CompletedState),
	string(sparkop.FailedState),
	string(sparkop.FailedSubmissionState),
	string(sparkop.UnknownState),
}

var runningStates = []string{
	string(sparkop.NewState),
	string(sparkop.SubmittedState),
	string(sparkop.RunningState),
}

var successStates = []string{
	string(sparkop.CompletedState),
}

var failureStates = []string{
	string(sparkop.FailedState),
	string(sparkop.FailedSubmissionState),
	string(sparkop.UnknownState),
}

var SuccessCondition = "status.applicationState.state in (" + strings.Join(successStates, ",") + ")"
var FailureCondition = "status.applicationState.state in (" + strings.Join(failureStates, ",") + ")"

func init() {
	var err error
	sparkClientset, err = clientset.NewForConfig(k8s.Config)
	if err != nil {
		errors.Exit(err, "spark", "kubeconfig")
	}

	sparkClient = sparkClientset.SparkoperatorV1alpha1().SparkApplications(cc.Namespace)
}

func Spec(workloadID string, ctx *context.Context, workloadType string, sparkCompute *userconfig.SparkCompute, args ...string) *sparkop.SparkApplication {
	var driverMemOverhead *string
	if sparkCompute.DriverMemOverhead != nil {
		driverMemOverhead = pointer.Str(s.Int64(sparkCompute.DriverMemOverhead.ToKi()) + "k")
	}
	var executorMemOverhead *string
	if sparkCompute.ExecutorMemOverhead != nil {
		executorMemOverhead = pointer.Str(s.Int64(sparkCompute.ExecutorMemOverhead.ToKi()) + "k")
	}
	var memOverheadFactor *string
	if sparkCompute.MemOverheadFactor != nil {
		memOverheadFactor = pointer.Str(s.Float64(*sparkCompute.MemOverheadFactor))
	}

	return &sparkop.SparkApplication{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "sparkoperator.k8s.io/v1alpha1",
			Kind:       "SparkApplication",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      workloadID,
			Namespace: cc.Namespace,
			Labels: map[string]string{
				"workloadID":   workloadID,
				"workloadType": workloadType,
				"appName":      ctx.App.Name,
			},
		},
		Spec: sparkop.SparkApplicationSpec{
			Type:                 sparkop.PythonApplicationType,
			PythonVersion:        pointer.Str("3"),
			Mode:                 sparkop.ClusterMode,
			Image:                &cc.SparkImage,
			ImagePullPolicy:      pointer.Str("Always"),
			MainApplicationFile:  pointer.Str("local:///src/spark_job/spark_job.py"),
			RestartPolicy:        sparkop.RestartPolicy{Type: sparkop.Never},
			MemoryOverheadFactor: memOverheadFactor,
			Arguments: []string{
				strings.TrimSpace(
					" --workload-id=" + workloadID +
						" --context=" + aws.S3Path(ctx.Key) +
						" --cache-dir=" + consts.ContextCacheDir +
						" " + strings.Join(args, " ")),
			},
			Deps: sparkop.Dependencies{
				PyFiles: []string{"local:///src/spark_job/spark_util.py", "local:///src/lib/*.py"},
			},
			Driver: sparkop.DriverSpec{
				SparkPodSpec: sparkop.SparkPodSpec{
					Cores:          pointer.Float32(sparkCompute.DriverCPU.ToFloat32()),
					Memory:         pointer.Str(s.Int64(sparkCompute.DriverMem.ToKi()) + "k"),
					MemoryOverhead: driverMemOverhead,
					Labels: map[string]string{
						"workloadID":   workloadID,
						"workloadType": workloadType,
						"appName":      ctx.App.Name,
						"userFacing":   "true",
					},
					EnvSecretKeyRefs: map[string]sparkop.NameKey{
						"AWS_ACCESS_KEY_ID": sparkop.NameKey{
							Name: "aws-credentials",
							Key:  "AWS_ACCESS_KEY_ID",
						},
						"AWS_SECRET_ACCESS_KEY": sparkop.NameKey{
							Name: "aws-credentials",
							Key:  "AWS_SECRET_ACCESS_KEY",
						},
					},
					EnvVars: map[string]string{
						"CORTEX_SPARK_VERBOSITY": ctx.Environment.LogLevel.Spark,
						"CORTEX_CONTEXT_S3_PATH": aws.S3Path(ctx.Key),
						"CORTEX_WORKLOAD_ID":     workloadID,
						"CORTEX_CACHE_DIR":       consts.ContextCacheDir,
					},
				},
				PodName:        &workloadID,
				ServiceAccount: pointer.Str("spark"),
			},
			Executor: sparkop.ExecutorSpec{
				SparkPodSpec: sparkop.SparkPodSpec{
					Cores:          pointer.Float32(sparkCompute.ExecutorCPU.ToFloat32()),
					Memory:         pointer.Str(s.Int64(sparkCompute.ExecutorMem.ToKi()) + "k"),
					MemoryOverhead: executorMemOverhead,
					Labels: map[string]string{
						"workloadID":   workloadID,
						"workloadType": workloadType,
						"appName":      ctx.App.Name,
					},
					EnvSecretKeyRefs: map[string]sparkop.NameKey{
						"AWS_ACCESS_KEY_ID": sparkop.NameKey{
							Name: "aws-credentials",
							Key:  "AWS_ACCESS_KEY_ID",
						},
						"AWS_SECRET_ACCESS_KEY": sparkop.NameKey{
							Name: "aws-credentials",
							Key:  "AWS_SECRET_ACCESS_KEY",
						},
					},
					EnvVars: map[string]string{
						"CORTEX_SPARK_VERBOSITY": ctx.Environment.LogLevel.Spark,
						"CORTEX_CONTEXT_S3_PATH": aws.S3Path(ctx.Key),
						"CORTEX_WORKLOAD_ID":     workloadID,
						"CORTEX_CACHE_DIR":       consts.ContextCacheDir,
					},
				},
				Instances: &sparkCompute.Executors,
			},
		},
	}
}

func List(opts *metav1.ListOptions) ([]sparkop.SparkApplication, error) {
	if opts == nil {
		opts = &metav1.ListOptions{}
	}
	sparkList, err := sparkClient.List(*opts)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return sparkList.Items, nil
}

func ListByLabels(labels map[string]string) ([]sparkop.SparkApplication, error) {
	opts := &metav1.ListOptions{
		LabelSelector: k8s.LabelSelector(labels),
	}
	return List(opts)
}

func ListByLabel(labelKey string, labelValue string) ([]sparkop.SparkApplication, error) {
	return ListByLabels(map[string]string{labelKey: labelValue})
}

func Delete(appName string) (bool, error) {
	err := sparkClient.Delete(appName, &metav1.DeleteOptions{})
	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err)
	}
	return true, nil
}

func IsDone(sparkApp *sparkop.SparkApplication) bool {
	return slices.IsStrInSlice(string(sparkApp.Status.AppState.State), doneStates)
}
