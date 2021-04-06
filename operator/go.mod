module github.com/cortexlabs/cortex/operator

go 1.13

require (
	github.com/aws/aws-sdk-go v1.36.2 // indirect
	github.com/cortexlabs/cortex v0.32.0
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	k8s.io/api v0.18.1
	k8s.io/apimachinery v0.18.1
	k8s.io/client-go v0.18.1
	sigs.k8s.io/controller-runtime v0.5.0
)

replace (
	github.com/cortexlabs/cortex => ../
	github.com/docker/docker => github.com/docker/engine v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
)
