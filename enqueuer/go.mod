module github.com/cortexlabs/enqueuer

go 1.16

require (
	github.com/aws/aws-sdk-go v1.38.15
	github.com/cortexlabs/cortex v0.32.0
	github.com/gobwas/glob v0.2.3
	go.uber.org/zap v1.16.0
)

replace (
	github.com/cortexlabs/cortex => ../
	github.com/docker/docker => github.com/docker/engine v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
)
