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

package batchcontrollers

import (
	"context"
	"time"

	batch "github.com/cortexlabs/cortex/operator/apis/batch/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
)

var _ = Describe("BatchJob controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		BatchJobName      = "test-batch-job"
		BatchJobNamespace = "default"
		timeout           = time.Second * 10
		interval          = time.Millisecond * 250
	)

	Context("Reconcile", func() {
		It("Should update the status correctly", func() {
			By("Creating a new BatchJob")
			ctx := context.Background()
			batchJob := &batch.BatchJob{
				ObjectMeta: kmeta.ObjectMeta{
					Name:      BatchJobName,
					Namespace: BatchJobNamespace,
				},
				Spec: batch.BatchJobSpec{
					APIName: "test-api",
					APIId:   "123456",
					Workers: 1,
				},
			}
			Expect(k8sClient.Create(ctx, batchJob)).Should(Succeed())

			By("Checking created spec matches the original")
			batchJobLookupKey := ktypes.NamespacedName{Name: batchJob.Name, Namespace: batchJob.Namespace}
			createdBatchJob := &batch.BatchJob{}

			// Check that the resource was created correctly (i.e. if the spec matches)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, batchJobLookupKey, createdBatchJob)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdBatchJob.Spec).Should(Equal(batchJob.Spec))
		})
	})
})
