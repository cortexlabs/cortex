// to update all:
//   go clean -modcache (optional)
//   rm go.mod go.sum
//   go mod tidy
//   replace these lines in go.mod:
//     github.com/cortexlabs/yaml v2.2.4
//     k8s.io/client-go v12.0.0
//     k8s.io/api 7525909cc6da
//     k8s.io/apimachinery 1799e75a0719
//       (note: go to the commit for the client-go release and browse to Godeps/Godeps.json to find the SHAs for k8s.io/api and k8s.io/apimachinery)
//   go mod tidy
//   check the diff in this file

module github.com/cortexlabs/cortex

go 1.12

require (
	github.com/aws/aws-sdk-go v1.20.20
	github.com/cortexlabs/yaml v0.0.0-20190626164117-202ab3a3d475
	github.com/davecgh/go-spew v1.1.1
	github.com/fatih/color v1.7.0
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/websocket v1.4.0
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/mattn/go-colorable v0.1.2 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.3.0
	github.com/tcnksm/go-input v0.0.0-20180404061846-548a7d7a8ee8
	github.com/ugorji/go/codec v1.1.7
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	k8s.io/api v0.0.0-20190620084959-7cf5895f2711
	k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
	k8s.io/utils v0.0.0-20190712204705-3dccf664f023 // indirect
)
