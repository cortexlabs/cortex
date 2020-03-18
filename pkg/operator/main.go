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

package main

import (
	"log"
	"net/http"

	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/endpoints"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/gorilla/mux"
)

const _operatorPortStr = "8888"

func main() {
	if err := config.Init(); err != nil {
		exit.Error(err)
	}

	if err := operator.Init(); err != nil {
		exit.Error(err)
	}

	router := mux.NewRouter()

	routerWithoutAuth := router.NewRoute().Subrouter()
	routerWithoutAuth.Use(endpoints.PanicMiddleware)
	routerWithoutAuth.HandleFunc("/verifycortex", endpoints.VerifyCortex).Methods("GET")

	routerWithAuth := router.NewRoute().Subrouter()

	routerWithAuth.Use(endpoints.PanicMiddleware)
	routerWithAuth.Use(endpoints.ClientIDMiddleware)
	routerWithAuth.Use(endpoints.APIVersionCheckMiddleware)
	routerWithAuth.Use(endpoints.AuthMiddleware)

	routerWithAuth.HandleFunc("/info", endpoints.Info).Methods("GET")
	routerWithAuth.HandleFunc("/deploy", endpoints.Deploy).Methods("POST")
	routerWithAuth.HandleFunc("/refresh/{apiName}", endpoints.Refresh).Methods("POST")
	routerWithAuth.HandleFunc("/delete/{apiName}", endpoints.Delete).Methods("DELETE")
	routerWithAuth.HandleFunc("/get", endpoints.GetAPIs).Methods("GET")
	routerWithAuth.HandleFunc("/get/{apiName}", endpoints.GetAPI).Methods("GET")
	routerWithAuth.HandleFunc("/logs/{apiName}", endpoints.ReadLogs)

	log.Print("Running on port " + _operatorPortStr)
	log.Fatal(http.ListenAndServe(":"+_operatorPortStr, router))
}
