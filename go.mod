module github.com/cortexlabs/cortex

go 1.14

require (
	cloud.google.com/go v0.73.0
	cloud.google.com/go/storage v1.12.0
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.11 // indirect
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/aws/aws-sdk-go v1.36.2
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/containerd/containerd v1.4.3 // indirect
	github.com/cortexlabs/go-input v0.0.0-20200503032952-8b67a7a7b28d
	github.com/cortexlabs/yaml v0.0.0-20200511220111-581aea36a2e4
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/denormal/go-gitignore v0.0.0-20180930084346-ae8ad1d07817
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v0.0.0-00010101000000-000000000000
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/fatih/color v1.10.0
	github.com/getsentry/sentry-go v0.8.0
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/gobwas/glob v0.2.3
	github.com/google/uuid v1.1.2
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v0.9.3
	github.com/prometheus/common v0.4.0
	github.com/segmentio/backo-go v0.0.0-20200129164019-23eae7c10bd3 // indirect
	github.com/shirou/gopsutil v3.20.11+incompatible
	github.com/sirupsen/logrus v1.4.1 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	github.com/ugorji/go/codec v1.2.1
	github.com/xlab/treeprint v1.0.0
	github.com/xtgo/uuid v0.0.0-20140804021211-a0b114877d4c // indirect
	go.uber.org/zap v1.10.0
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb // indirect
	golang.org/x/oauth2 v0.0.0-20201203001011-0b49973bad19
	golang.org/x/sys v0.0.0-20201204225414-ed752295db88 // indirect
	golang.org/x/tools v0.0.0-20201206230334-368bee879bfd // indirect
	google.golang.org/api v0.36.0
	google.golang.org/genproto v0.0.0-20201204160425-06b3db808446
	google.golang.org/grpc v1.34.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/karalabe/cookiejar.v2 v2.0.0-20150724131613-8dcd6a7f4951
	gopkg.in/segmentio/analytics-go.v3 v3.1.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	gotest.tools v2.2.0+incompatible // indirect
	istio.io/api v0.0.0-20200911191701-0dc35ad5c478
	istio.io/client-go v0.0.0-20200807182027-d287a5abb594
	k8s.io/api v0.18.1
	k8s.io/apimachinery v0.18.1
	k8s.io/client-go v0.18.1
)

replace github.com/docker/docker => github.com/docker/engine v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
