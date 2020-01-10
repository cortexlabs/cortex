import (
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	kapps "k8s.io/api/apps/v1"
)

func APIPodComputeID(containers []kcore.Container) string {
	cpu, mem, gpu := APIPodCompute(containers)
	if cpu == nil {
		cpu = &k8s.Quantity{} // unexpected, since 0 is disallowed
	}
	podAPICompute := userconfig.APICompute{
		CPU: *cpu,
		Mem: mem,
		GPU: gpu,
	}
	return podAPICompute.IDWithoutReplicas()
}

func APIPodCompute(containers []kcore.Container) (*k8s.Quantity, *k8s.Quantity, int64) {
	var totalCPU *k8s.Quantity
	var totalMem *k8s.Quantity
	var totalGPU int64

	for _, container := range containers {
		if container.Name != _apiContainerName && container.Name != _tfServingContainerName {
			continue
		}

		requests := container.Resources.Requests
		if len(requests) == 0 {
			continue
		}

		if cpu, ok := requests[kcore.ResourceCPU]; ok {
			if totalCPU == nil {
				totalCPU = &k8s.Quantity{}
			}
			totalCPU.Add(cpu)
		}
		if mem, ok := requests[kcore.ResourceMemory]; ok {
			if totalMem == nil {
				totalMem = &k8s.Quantity{}
			}
			totalMem.Add(mem)
		}
		if gpu, ok := requests["nvidia.com/gpu"]; ok {
			gpuVal, ok := gpu.AsInt64()
			if ok {
				totalGPU += gpuVal
			}
		}
	}

	return totalCPU, totalMem, totalGPU
}

func doesAPIComputeNeedsUpdating(api *context.API, k8sDeployment *kapps.Deployment) bool {
	requestedReplicas := getRequestedReplicasFromDeployment(api, k8sDeployment, nil)
	if k8sDeployment.Spec.Replicas == nil || *k8sDeployment.Spec.Replicas != requestedReplicas {
		return true
	}

	curCPU, curMem, curGPU := APIPodCompute(k8sDeployment.Spec.Template.Spec.Containers)
	if !k8s.QuantityPtrsEqual(curCPU, &api.Compute.CPU) {
		return true
	}
	if !k8s.QuantityPtrsEqual(curMem, api.Compute.Mem) {
		return true
	}
	if curGPU != api.Compute.GPU {
		return true
	}

	return false
}

func (aw *APIWorkload) IsFailed(ctx *context.Context) (bool, error) {
	api := ctx.APIs.OneByID(aw.GetSingleResourceID())

	pods, err := config.Kubernetes.ListPodsByLabels(map[string]string{
		"appName":      ctx.App.Name,
		"workloadType": workloadTypeAPI,
		"apiName":      api.Name,
		"resourceID":   api.ID,
		"workloadID":   aw.GetWorkloadID(),
		"userFacing":   "true",
	})
	if err != nil {
		return false, err
	}

	for _, pod := range pods {
		podStatus := k8s.GetPodStatus(&pod)
		if podStatus == k8s.PodStatusFailed || podStatus == k8s.PodStatusKilled || podStatus == k8s.PodStatusKilledOOM {
			return true, nil
		}
	}

	return false, nil
}

func updateAPISavedStatusStartTime(savedStatus *resource.APISavedStatus, pods []kcore.Pod) {
	if savedStatus.Start != nil {
		return
	}

	for _, pod := range pods {
		podReadyTime := k8s.GetPodReadyTime(&pod)
		if podReadyTime == nil {
			continue
		}
		if savedStatus.Start == nil || (*podReadyTime).Before(*savedStatus.Start) {
			savedStatus.Start = podReadyTime
		}
	}
}
