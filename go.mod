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
	github.com/aws/aws-sdk-go v1.16.17
	github.com/davecgh/go-spew v1.1.1
	github.com/emicklei/go-restful v2.8.0+incompatible // indirect
	github.com/go-openapi/spec v0.18.0 // indirect
	github.com/gogo/protobuf v1.2.0 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/google/btree v0.0.0-20180813153112-4030bb1f1f0c // indirect
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.6.2
	github.com/gorilla/websocket v1.4.0
	github.com/gregjones/httpcache v0.0.0-20181110185634-c63ab54fda8f // indirect
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/json-iterator/go v1.1.5 // indirect
	github.com/mitchellh/go-homedir v1.0.0
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/tcnksm/go-input v0.0.0-20180404061846-548a7d7a8ee8
	github.com/ugorji/go/codec v0.0.0-20181209151446-772ced7fd4c2
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca
	golang.org/x/crypto v0.0.0-20190103213133-ff983b9c42bc // indirect
	golang.org/x/net v0.0.0-20190110200230-915654e7eabc // indirect
	golang.org/x/oauth2 v0.0.0-20190110195249-fd3eaa146cbb // indirect
	golang.org/x/sys v0.0.0-20190109145017-48ac38b7c8cb // indirect
	golang.org/x/time v0.0.0-20181108054448-85acf8d2951c // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20181204000039-89a74a8d264d
	k8s.io/apimachinery v0.0.0-20181127025237-2b1284ed4c93
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/klog v0.1.0 // indirect
	k8s.io/kube-openapi v0.0.0-20181114233023-0317810137be // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)
