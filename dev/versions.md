# Upgrade notes

## Go modules

### Update non-versioned modules

1. `go clean -modcache`
1. `rm go.mod go.sum`
1. `go mod tidy`
1. replace these lines in go.mod:
    github.com/cortexlabs/yaml v2.2.4
    k8s.io/client-go v12.0.0
    k8s.io/api 7525909cc6da
    k8s.io/apimachinery 1799e75a0719
1. `go mod tidy`
1. check the diff in go.mod

### Update `cortexlabs/yaml`

1. Follow the "Update non-versioned modules" instructions, inserting the desired version for `cortexlabs/yaml`

### Update go client

1. go to the commit for the client-go release and browse to Godeps/Godeps.json to find the SHAs for k8s.io/api and k8s.io/apimachinery
1. Follow the "Update non-versioned modules" instructions, inserting the applicable versions for `k8s.io/*`
