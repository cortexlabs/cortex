module github.com/cortexlabs/cortex

go 1.12

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/aws/aws-sdk-go v1.25.31
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/containerd/containerd v1.3.0 // indirect
	github.com/cortexlabs/yaml v0.0.0-20190626164117-202ab3a3d475
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v0.0.0-00010101000000-000000000000
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/fatih/color v1.7.0
	github.com/getsentry/sentry-go v0.3.1
	github.com/google/uuid v1.0.0
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/websocket v1.4.1
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pkg/errors v0.8.1
	github.com/segmentio/backo-go v0.0.0-20160424052352-204274ad699c // indirect
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.4.0
	github.com/tcnksm/go-input v0.0.0-20180404061846-548a7d7a8ee8
	github.com/ugorji/go/codec v1.1.7
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca
	github.com/xtgo/uuid v0.0.0-20140804021211-a0b114877d4c // indirect
	golang.org/x/lint v0.0.0-20190313153728-d0100b6bd8b3 // indirect
	golang.org/x/tools v0.0.0-20190524140312-2c0ae7006135 // indirect
	google.golang.org/grpc v1.25.1 // indirect
	gopkg.in/karalabe/cookiejar.v2 v2.0.0-20150724131613-8dcd6a7f4951
	gopkg.in/segmentio/analytics-go.v3 v3.1.0
	gotest.tools v2.2.0+incompatible // indirect
	honnef.co/go/tools v0.0.0-20190523083050-ea95bdfd59fc // indirect
	k8s.io/api v0.0.0-20190620084959-7cf5895f2711
	k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
)

replace github.com/docker/docker => github.com/docker/engine v0.0.0-20191011211953-adfac697dc5b
