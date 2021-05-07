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
	"fmt"

	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kcore "k8s.io/api/core/v1"
)

func K8sName(apiName string) string {
	return "api-" + apiName
}

type downloadContainerConfig struct {
	DownloadArgs []downloadContainerArg `json:"download_args"`
	LastLog      string                 `json:"last_log"` // string to log at the conclusion of the downloader (if "" nothing will be logged)
}

type downloadContainerArg struct {
	From             string `json:"from"`
	To               string `json:"to"`
	ToFile           bool   `json:"to_file"` // whether "To" path reflects the path to a file or just the directory in which "From" object is copied to
	Unzip            bool   `json:"unzip"`
	ItemName         string `json:"item_name"`          // name of the item being downloaded, just for logging (if "" nothing will be logged)
	HideFromLog      bool   `json:"hide_from_log"`      // if true, don't log where the file is being downloaded from
	HideUnzippingLog bool   `json:"hide_unzipping_log"` // if true, don't log when unzipping
}

func FileExistsProbe(fileName string) *kcore.Probe {
	return &kcore.Probe{
		InitialDelaySeconds: 3,
		TimeoutSeconds:      5,
		PeriodSeconds:       5,
		SuccessThreshold:    1,
		FailureThreshold:    1,
		Handler: kcore.Handler{
			Exec: &kcore.ExecAction{
				Command: []string{"/bin/bash", "-c", fmt.Sprintf("test -f %s", fileName)},
			},
		},
	}
}

func socketExistsProbe(socketName string) *kcore.Probe {
	return &kcore.Probe{
		InitialDelaySeconds: 3,
		TimeoutSeconds:      5,
		PeriodSeconds:       5,
		SuccessThreshold:    1,
		FailureThreshold:    1,
		Handler: kcore.Handler{
			Exec: &kcore.ExecAction{
				Command: []string{"/bin/bash", "-c", fmt.Sprintf("test -S %s", socketName)},
			},
		},
	}
}

func nginxGracefulStopper(apiKind userconfig.Kind) *kcore.Lifecycle {
	if apiKind == userconfig.RealtimeAPIKind {
		return &kcore.Lifecycle{
			PreStop: &kcore.Handler{
				Exec: &kcore.ExecAction{
					// the sleep is required to wait for any k8s-related race conditions
					// as described in https://medium.com/codecademy-engineering/kubernetes-nginx-and-zero-downtime-in-production-2c910c6a5ed8
					Command: []string{"/bin/sh", "-c", "sleep 5; /usr/sbin/nginx -s quit; while pgrep -x nginx; do sleep 1; done"},
				},
			},
		}
	}
	return nil
}

func waitAPIContainerToStop(apiKind userconfig.Kind) *kcore.Lifecycle {
	if apiKind == userconfig.RealtimeAPIKind {
		return &kcore.Lifecycle{
			PreStop: &kcore.Handler{
				Exec: &kcore.ExecAction{
					Command: []string{"/bin/sh", "-c", fmt.Sprintf("while curl localhost:%s/nginx_status; do sleep 1; done", DefaultPortStr)},
				},
			},
		}
	}
	return nil
}
