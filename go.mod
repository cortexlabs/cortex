// locked dependency versions:
//   github.com/GoogleCloudPlatform/spark-on-k8s-operator v1alpha1-0.5-2.4.0
//   github.com/argoproj/argo v2.2.1
//   k8s.io/client-go v10.0.0
// go to the commit for the client-go release and browse to Godeps/Godeps.json to find these SHAs:
//   k8s.io/api 89a74a8d264df0e993299876a8cde88379b940ee
//   k8s.io/apimachinery 2b1284ed4c93a43499e781493253e2ac5959c4fd
//
// to update all:
//   rm go.mod go.sum
//   go mod tidy
//   (replace versions of locked dependencies above)
//   go mod tidy

module github.com/cortexlabs/cortex

require (
	github.com/GoogleCloudPlatform/spark-on-k8s-operator v0.0.0-20181208011959-62db1d66dafa
	github.com/argoproj/argo v2.2.1+incompatible
	github.com/aws/aws-sdk-go v1.16.23
	github.com/davecgh/go-spew v1.1.1
	github.com/emicklei/go-restful v2.8.0+incompatible // indirect
	github.com/go-openapi/spec v0.18.0 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gorilla/mux v1.6.2
	github.com/gorilla/websocket v1.4.0
	github.com/gregjones/httpcache v0.0.0-20181110185634-c63ab54fda8f // indirect
	github.com/mitchellh/go-homedir v1.0.0
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.3
	github.com/stretchr/testify v1.3.0
	github.com/tcnksm/go-input v0.0.0-20180404061846-548a7d7a8ee8
	github.com/ugorji/go/codec v0.0.0-20181209151446-772ced7fd4c2
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca
	gocloud.dev v0.13.0
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20181221193117-173ce66c1e39
	k8s.io/apimachinery v0.0.0-20190119020841-d41becfba9ee
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20181114233023-0317810137be // indirect
)
