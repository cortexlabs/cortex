// to update all:
//   go clean -modcache (optional)
//   rm go.mod go.sum
//   go mod tidy
//   replace these lines in go.mod:
//     github.com/GoogleCloudPlatform/spark-on-k8s-operator v1alpha1-0.5-2.4.0
//     github.com/argoproj/argo v2.3.0
//     github.com/cortexlabs/yaml v2.2.3
//     k8s.io/client-go v10.0.0
//     k8s.io/api 89a74a8d264df0e993299876a8cde88379b940ee
//     k8s.io/apimachinery 2b1284ed4c93a43499e781493253e2ac5959c4fd
//       (note: go to the commit for the client-go release and browse to Godeps/Godeps.json to find the SHAs for k8s.io/api and k8s.io/apimachinery)
//   go mod tidy
//   check the diff in this file

module github.com/cortexlabs/cortex

go 1.12

require (
	github.com/GoogleCloudPlatform/spark-on-k8s-operator v0.0.0-20181208011959-62db1d66dafa
	github.com/argoproj/argo v2.3.0+incompatible
	github.com/aws/aws-sdk-go v1.20.7
	github.com/cortexlabs/yaml v0.0.0-20190624201412-7f31702857b6
	github.com/davecgh/go-spew v1.1.1
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/go-openapi/spec v0.19.2 // indirect
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/gorilla/mux v1.7.2
	github.com/gorilla/websocket v1.4.0
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/json-iterator/go v1.1.6 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.3.0
	github.com/tcnksm/go-input v0.0.0-20180404061846-548a7d7a8ee8
	github.com/ugorji/go/codec v1.1.5-pre
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	k8s.io/api v0.0.0-20181204000039-89a74a8d264d
	k8s.io/apimachinery v0.0.0-20181127025237-2b1284ed4c93
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/klog v0.3.0 // indirect
	k8s.io/kube-openapi v0.0.0-20190603182131-db7b694dc208 // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)
