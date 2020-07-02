/*
Copyright 2020 Cortex Labs, Inc.

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

package endpoints

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/batchapi"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/gorilla/mux"
)

func Batch(w http.ResponseWriter, r *http.Request) {
	fmt.Println("here")
	vars := mux.Vars(r)

	rw := http.MaxBytesReader(w, r.Body, 32<<10)

	bodyBytes, err := ioutil.ReadAll(rw)
	if err != nil {
		respondError(w, r, err)
		return
	}

	sub := batchapi.Submission{}

	err = json.Unmarshal(bodyBytes, &sub)
	if err != nil {
		respondError(w, r, err)
		return
	}

	jobSpec, err := batchapi.SubmitJob(vars["apiName"], sub)
	if err != nil {
		respondError(w, r, err)
		return
	}

	respond(w, jobSpec)

	// debug.Pp(sub)

	// objects, err := config.AWS.ListS3Prefix(config.Cluster.Bucket, filepath.Join("apis", sub.APIName), false, pointer.Int64(1))
	// if err != nil {
	// 	respondError(w, r, err)
	// 	return
	// }

	// apiSpec := spec.API{}
	// err = config.AWS.ReadMsgpackFromS3(&apiSpec, config.Cluster.Bucket, *objects[0].Key)
	// if err != nil {
	// 	respondError(w, r, err)
	// 	return
	// }

	// debug.Pp(apiSpec)

	// output, err := config.AWS.SQS().CreateQueue(
	// 	&sqs.CreateQueueInput{
	// 		QueueName: aws.String("batch"),
	// 	},
	// )

	// debug.Pp(output)
	// if err != nil {
	// 	debug.Pp(err.Error())
	// 	return
	// }

	// for _, item := range sub.Items {
	// 	_, err := config.AWS.SQS().SendMessage(&sqs.SendMessageInput{
	// 		QueueUrl:    output.QueueUrl,
	// 		MessageBody: aws.String(item),
	// 	})
	// 	if err != nil {
	// 		respondError(w, r, err)
	// 		return
	// 	}
	// }

	// apiSpec.Predictor.Env["SQS_QUEUE_URL"] = *output.QueueUrl

	// _, err = config.K8s.CreateJob(operator.PythonJobSpec(&apiSpec, sub.Parallelism))
	// if err != nil {
	// 	respondError(w, r, err)
	// 	return
	// }

}

func BatchDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	err := batchapi.StopJob(spec.JobID{APIName: vars["apiName"], ID: vars["jobID"]})
	if err != nil {
		respondError(w, r, err)
		return
	}

	respond(w, schema.DeleteResponse{
		Message: fmt.Sprintf("deleted job %s", vars["jobID"]),
	})
}

func BatchGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := spec.JobID{APIName: vars["apiName"], ID: vars["jobID"]}

	jobStatus, err := batchapi.GetJobStatus(jobID)
	if err != nil {
		respondError(w, r, err)
		return
	}

	spec, err := operator.DownloadAPISpec(jobStatus.APIName, jobStatus.APIID)
	if err != nil {
		respondError(w, r, err)
		return
	}

	response := schema.JobResponse{
		JobStatus: *jobStatus,
		APISpec:   *spec,
	}

	respond(w, response)
}
