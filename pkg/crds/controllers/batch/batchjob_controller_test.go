/*
Copyright 2022 Cortex Labs, Inc.

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

package batchcontrollers_test

import (
	"context"
	"strings"
	"time"

	batch "github.com/cortexlabs/cortex/pkg/crds/apis/batch/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/random"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	kbatch "k8s.io/api/batch/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func uploadTestAPISpec(apiName string, apiID string) error {
	apiSpec := spec.API{
		API: &userconfig.API{
			Resource: userconfig.Resource{
				Name: apiName,
				Kind: userconfig.BatchAPIKind,
			},
			Pod: &userconfig.Pod{
				Port: pointer.Int32(8080),
				Containers: []*userconfig.Container{
					{
						Name:    "api",
						Image:   "quay.io/cortexlabs/batch-container-test:master",
						Command: []string{"/bin/run"},
						Compute: &userconfig.Compute{},
					},
				},
			},
		},
		ID:           apiID,
		SpecID:       random.String(5),
		PodID:        random.String(5),
		DeploymentID: random.String(5),
	}
	apiSpecKey := spec.Key(apiName, apiID, clusterConfig.ClusterUID)
	if err := awsClient.UploadJSONToS3(apiSpec, clusterConfig.Bucket, apiSpecKey); err != nil {
		return err
	}
	return nil
}

func deleteTestAPISpec(apiName string, apiID string) error {
	apiSpecKey := spec.Key(apiName, apiID, clusterConfig.ClusterUID)
	if err := awsClient.DeleteS3File(clusterConfig.Bucket, apiSpecKey); err != nil {
		return err
	}
	return nil
}

var _ = Describe("BatchJob controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		APIName           = "test-api"
		BatchJobNamespace = "default"
		timeout           = time.Second * 10
		interval          = time.Millisecond * 250
	)

	var (
		randomJobID string
		randomAPIID string
	)

	Context("Reconciliation", func() {
		BeforeEach(func() {
			// ensures the tests can be ran in rapid succession by avoiding the time limits of SQS queue creation
			randomJobID = strings.ToLower(random.String(5))
			randomAPIID = random.Digits(5)

			Expect(uploadTestAPISpec(APIName, randomAPIID)).To(Succeed())
		})

		AfterEach(func(done Done) {
			Expect(deleteTestAPISpec(APIName, randomAPIID)).To(Succeed())

			Expect(k8sClient.Delete(
				context.Background(),
				&batch.BatchJob{
					ObjectMeta: kmeta.ObjectMeta{Name: randomJobID, Namespace: BatchJobNamespace},
				},
			)).To(Succeed())
			close(done)
		})

		It("Should reach a batch job completed status", func() {
			When("Every child resource is created or finishes successfully", func() {
				By("Creating a new BatchJob")
				ctx := context.Background()
				batchJob := &batch.BatchJob{
					ObjectMeta: kmeta.ObjectMeta{
						Name:      randomJobID,
						Namespace: BatchJobNamespace,
					},
					Spec: batch.BatchJobSpec{
						APIName: APIName,
						APIID:   randomAPIID,
						Workers: 1,
					},
				}
				Expect(k8sClient.Create(ctx, batchJob)).To(Succeed())

				batchJobLookupKey := ktypes.NamespacedName{Name: batchJob.Name, Namespace: batchJob.Namespace}
				createdBatchJob := &batch.BatchJob{}

				// Check that the resource was created correctly (i.e. if the spec matches)
				Eventually(func() error {
					return k8sClient.Get(ctx, batchJobLookupKey, createdBatchJob)
				}, timeout, interval).Should(Succeed())
				Expect(createdBatchJob.Spec).Should(Equal(batchJob.Spec))

				By("Creating an SQS queue successfully")
				Eventually(func() bool {
					err := k8sClient.Get(ctx, batchJobLookupKey, createdBatchJob)
					if err != nil {
						return false
					}
					return createdBatchJob.Status.QueueURL != ""
				}, timeout, interval).Should(BeTrue())

				By("Reaching a completed enqueuer status")
				enqueuerJobLookupKey := ktypes.NamespacedName{
					Name:      batchJob.Spec.APIName + "-" + batchJob.Name + "-enqueuer",
					Namespace: batchJob.Namespace,
				}
				createdEnqueuerJob := &kbatch.Job{}

				// wait for the enqueuer job to be created
				Eventually(func() error {
					return k8sClient.Get(ctx, enqueuerJobLookupKey, createdEnqueuerJob)
				}, timeout, interval).Should(Succeed())

				// Mock the enqueuer status to match the success condition
				createdEnqueuerJob.Status.Succeeded = 1
				Expect(k8sClient.Status().Update(ctx, createdEnqueuerJob)).To(Succeed())

				Eventually(func() bool {
					err := k8sClient.Get(ctx, batchJobLookupKey, createdBatchJob)
					if err != nil {
						return false
					}
					return createdBatchJob.Status.Status == status.JobEnqueuing
				}, timeout, interval)

				By("Reaching a successful worker job status")

				workerJobLookupKey := ktypes.NamespacedName{
					Name:      batchJob.Spec.APIName + "-" + batchJob.Name,
					Namespace: BatchJobNamespace,
				}
				createdWorkerJob := &kbatch.Job{}

				// Wait for worker job to be created
				Eventually(func() error {
					return k8sClient.Get(ctx, workerJobLookupKey, createdWorkerJob)
				}, timeout, interval).Should(Succeed())

				// Mock the worker job status to match the success condition
				createdWorkerJob.Status.Succeeded = batchJob.Spec.Workers
				Expect(k8sClient.Status().Update(ctx, createdWorkerJob)).To(Succeed())

				Eventually(func() bool {
					err := k8sClient.Get(ctx, batchJobLookupKey, createdBatchJob)
					if err != nil {
						return false
					}
					return createdBatchJob.Status.Status == status.JobSucceeded
				}).Should(BeTrue())
			})
		})
	})

	Context("Reconcialiation TTL", func() {
		BeforeEach(func() {
			// ensures the tests can be ran in rapid succession by avoiding the time limits of SQS queue creation
			randomJobID = strings.ToLower(random.String(5))
			randomAPIID = random.Digits(5)

			Expect(uploadTestAPISpec(APIName, randomAPIID)).To(Succeed())
		})

		AfterEach(func(done Done) {
			Expect(deleteTestAPISpec(APIName, randomAPIID)).To(Succeed())
			close(done)
		})

		It("Should self clean-up when a completed status is reached and the TTL is exceeded", func() {
			By("Creating a new BatchJob")
			ttl := kmeta.Duration{Duration: time.Second * 10}
			ctx := context.Background()
			batchJob := &batch.BatchJob{
				ObjectMeta: kmeta.ObjectMeta{
					Name:      randomJobID,
					Namespace: BatchJobNamespace,
				},
				Spec: batch.BatchJobSpec{
					APIName: APIName,
					APIID:   randomAPIID,
					Workers: 1,
					TTL:     &ttl,
				},
			}
			Expect(k8sClient.Create(ctx, batchJob)).To(Succeed())

			By("Reaching a completed enqueuer status")
			enqueuerJobLookupKey := ktypes.NamespacedName{
				Name:      batchJob.Spec.APIName + "-" + batchJob.Name + "-enqueuer",
				Namespace: batchJob.Namespace,
			}
			createdEnqueuerJob := &kbatch.Job{}

			// wait for the enqueuer job to be created
			Eventually(func() error {
				return k8sClient.Get(ctx, enqueuerJobLookupKey, createdEnqueuerJob)
			}, timeout, interval).Should(Succeed())

			// Mock the enqueuer status to match the success condition
			createdEnqueuerJob.Status.Succeeded = 1
			Expect(k8sClient.Status().Update(ctx, createdEnqueuerJob)).To(Succeed())

			By("Reaching a successful worker job status")
			workerJobLookupKey := ktypes.NamespacedName{
				Name:      batchJob.Spec.APIName + "-" + batchJob.Name,
				Namespace: BatchJobNamespace,
			}
			createdWorkerJob := &kbatch.Job{}

			// Wait for worker job to be created
			Eventually(func() error {
				return k8sClient.Get(ctx, workerJobLookupKey, createdWorkerJob)
			}, timeout, interval).Should(Succeed())

			// Mock the worker job status to match the success condition
			completionTime := time.Now()
			createdWorkerJob.Status.Succeeded = batchJob.Spec.Workers
			createdWorkerJob.Status.CompletionTime = &kmeta.Time{Time: completionTime}
			Expect(k8sClient.Status().Update(ctx, createdWorkerJob)).To(Succeed())

			By("Waiting for the TTL to kick in")
			var deletionTime time.Time
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, client.ObjectKey{
					Name:      randomJobID,
					Namespace: BatchJobNamespace,
				}, &batch.BatchJob{}); err != nil {
					if kerrors.IsNotFound(err) {
						deletionTime = time.Now()
						return true
					}
				}
				return false
			}, ttl.Duration.Seconds()*2).Should(BeTrue())

			duration := deletionTime.Sub(completionTime)
			Expect(duration > ttl.Duration).Should(BeTrue())
		})

		It("Should self clean-up when enqueing fails and the TTL is exceeded", func() {
			By("Creating a new BatchJob")
			ttl := kmeta.Duration{Duration: time.Second * 10}
			ctx := context.Background()
			batchJob := &batch.BatchJob{
				ObjectMeta: kmeta.ObjectMeta{
					Name:      randomJobID,
					Namespace: BatchJobNamespace,
				},
				Spec: batch.BatchJobSpec{
					APIName: APIName,
					APIID:   randomAPIID,
					Workers: 1,
					TTL:     &ttl,
				},
			}
			Expect(k8sClient.Create(ctx, batchJob)).To(Succeed())

			By("Reaching a failed enqueuer status")
			enqueuerJobLookupKey := ktypes.NamespacedName{
				Name:      batchJob.Spec.APIName + "-" + batchJob.Name + "-enqueuer",
				Namespace: batchJob.Namespace,
			}
			createdEnqueuerJob := &kbatch.Job{}

			// wait for the enqueuer job to be created
			Eventually(func() error {
				return k8sClient.Get(ctx, enqueuerJobLookupKey, createdEnqueuerJob)
			}, timeout, interval).Should(Succeed())

			// Mock the enqueuer status to match the success condition
			createdEnqueuerJob.Status.Failed = 1
			Expect(k8sClient.Status().Update(ctx, createdEnqueuerJob)).To(Succeed())

			By("Waiting for the TTL to kick in")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name:      randomJobID,
					Namespace: BatchJobNamespace,
				}, &batch.BatchJob{})
				return kerrors.IsNotFound(err)
			}, ttl.Duration.Seconds()*2, interval).Should(BeTrue())
		})
	})
})
