module github.com/cortexlabs/cortex

go 1.13

require (
	github.com/aws/amazon-vpc-cni-k8s v1.5.5
	github.com/aws/aws-sdk-go v1.26.8
	github.com/containerd/containerd v1.3.2 // indirect
	github.com/cortexlabs/yaml v0.0.0-20191227012959-6abcdc706492
	github.com/davecgh/go-spew v1.1.1
	github.com/denormal/go-gitignore v0.0.0-20180930084346-ae8ad1d07817
	github.com/denverdino/aliyungo v0.0.0-20200221080937-dd4992dc11f6 // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20200220113713-29f9e0ba54ea // indirect
	github.com/fatih/color v1.7.0
	github.com/getsentry/sentry-go v0.3.1
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/websocket v1.4.1
	github.com/iancoleman/strcase v0.0.0-20191112232945-16388991a334
	github.com/mattn/go-isatty v0.0.11 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/segmentio/backo-go v0.0.0-20160424052352-204274ad699c // indirect
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.4.0
	github.com/tcnksm/go-input v0.0.0-20180404061846-548a7d7a8ee8
	github.com/ugorji/go/codec v1.1.7
	github.com/weaveworks/eksctl v0.0.0-20200226165220-a591407516a6
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca
	github.com/xtgo/uuid v0.0.0-20140804021211-a0b114877d4c // indirect
	golang.org/x/crypto v0.0.0-20191219195013-becbf705a915 // indirect
	golang.org/x/oauth2 v0.0.0-20191202225959-858c2ad4c8b6 // indirect
	golang.org/x/sys v0.0.0-20191224085550-c709ea063b76 // indirect
	gopkg.in/karalabe/cookiejar.v2 v2.0.0-20150724131613-8dcd6a7f4951
	gopkg.in/segmentio/analytics-go.v3 v3.1.0
	gopkg.in/yaml.v2 v2.2.7 // indirect
	k8s.io/api v0.0.0-20191004102349-159aefb8556b
	k8s.io/apimachinery v0.0.0-20191004074956-c5d2f014d689
	k8s.io/client-go v11.0.1-0.20191029005444-8e4128053008+incompatible
	k8s.io/utils v0.0.0-20191218082557-f07c713de883 // indirect
)

replace (
	// Fix vendor/k8s.io/client-go/plugin/pkg/client/auth/azure/azure.go:246:4: cannot use expiresIn (type string) as type json.Number in field value
	// Fix vendor/k8s.io/client-go/plugin/pkg/client/auth/azure/azure.go:247:4: cannot use expiresOn (type string) as type json.Number in field value
	// Fix vendor/k8s.io/client-go/plugin/pkg/client/auth/azure/azure.go:248:4: cannot use expiresOn (type string) as type json.Number in field value
	// Fix vendor/k8s.io/client-go/plugin/pkg/client/auth/azure/azure.go:265:23: cannot use token.token.ExpiresIn (type json.Number) as type string in assignment
	// Fix vendor/k8s.io/client-go/plugin/pkg/client/auth/azure/azure.go:266:23: cannot use token.token.ExpiresOn (type json.Number) as type string in assignment
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v10.14.0+incompatible
	github.com/aws/aws-sdk-go => github.com/aws/aws-sdk-go v1.26.8
	github.com/awslabs/goformation => github.com/errordeveloper/goformation v0.0.0-20190507151947-a31eae35e596
	// Override version since auto-detected one fails with GOPROXY
	github.com/census-instrumentation/opencensus-proto => github.com/census-instrumentation/opencensus-proto v0.2.0
	github.com/docker/docker => github.com/docker/engine v1.4.2-0.20191113042239-ea84732a7725
	// Used to pin the k8s library versions regardless of what other dependencies enforce
	k8s.io/api => k8s.io/api v0.0.0-20190226173710-145d52631d00
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190226180157-bd0469a053ff
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221084156-01f179d85dbc
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190226174127-78295b709ec6
)
