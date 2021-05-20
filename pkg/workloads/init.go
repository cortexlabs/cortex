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

package workloads

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kcore "k8s.io/api/core/v1"
)

const (
	JobSpecPath = "/cortex/job_spec.json"
)

const (
	_downloaderInitContainerName = "downloader"
	_downloaderLastLog           = "downloading the serving image(s)"
	_kubexitInitContainerName    = "kubexit"
)

func KubexitInitContainer() kcore.Container {
	return kcore.Container{
		Name:            _kubexitInitContainerName,
		Image:           config.ClusterConfig.ImageKubexit,
		ImagePullPolicy: kcore.PullAlways,
		Command:         []string{"cp", "/bin/kubexit", "/cortex/kubexit"},
		VolumeMounts: []kcore.VolumeMount{
			CortexMount(),
		},
	}
}

func BatchInitContainer(job *spec.BatchJob) kcore.Container {
	downloadConfig := downloadContainerConfig{
		LastLog: _downloaderLastLog,
		DownloadArgs: []downloadContainerArg{
			{
				From:             aws.S3Path(config.ClusterConfig.Bucket, job.SpecFilePath(config.ClusterConfig.ClusterUID)),
				To:               JobSpecPath,
				Unzip:            false,
				ToFile:           true,
				ItemName:         "the job spec",
				HideFromLog:      true,
				HideUnzippingLog: true,
			},
		},
	}

	downloadArgsBytes, _ := json.Marshal(downloadConfig)
	downloadArgs := base64.URLEncoding.EncodeToString(downloadArgsBytes)

	return kcore.Container{
		Name:            _downloaderInitContainerName,
		Image:           config.ClusterConfig.ImageDownloader,
		ImagePullPolicy: kcore.PullAlways,
		Args:            []string{"--download=" + downloadArgs},
		EnvFrom:         baseClusterEnvVars(),
		Env: []kcore.EnvVar{
			{
				Name:  "CORTEX_LOG_LEVEL",
				Value: strings.ToUpper(userconfig.InfoLogLevel.String()),
			},
		},
		VolumeMounts: []kcore.VolumeMount{
			CortexMount(),
		},
	}
}
