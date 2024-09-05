0. Write the following to go.mod
```go
module github.com/valaparthvi/rancher

go 1.19

require (
	github.com/onsi/ginkgo/v2 latest
	github.com/onsi/gomega latest
	github.com/rancher/rancher 9129a7b0496c
	github.com/stretchr/testify latest
	k8s.io/api v0.27.5
	k8s.io/apimachinery v0.27.5
)

replace (
	k8s.io/api => k8s.io/api v0.27.5
	k8s.io/client-go => github.com/rancher/client-go v1.27.4-rancher1
)
```
1. `go mod download` - This will resolve all the versions and add appropriate entry to `go.sum`.

---Output of go.mod---
```
module github.com/valaparthvi/rancher

go 1.19

require (
	github.com/onsi/ginkgo/v2 v2.12.1
	github.com/onsi/gomega v1.27.10
	github.com/rancher/rancher v0.0.0-20230921211342-9129a7b0496c
	github.com/stretchr/testify v1.8.4
	k8s.io/api v0.27.5
	k8s.io/apimachinery v0.27.5
)

replace (
	k8s.io/api => k8s.io/api v0.27.4
	k8s.io/client-go => github.com/rancher/client-go v1.27.4-rancher1
)
```
---------

(Optional) 2. `go mod tidy` - this will error out since we have not resolved `rancher/rancher/pkg/apis` and `rancher/rancher/pkg/client` dependencies.

```shell
$ go mod tidy
github.com/rancher/rancher/pkg/apis/ui.cattle.io/v1: reading github.com/rancher/rancher/pkg/apis/go.mod at revision pkg/apis/v0.0.0: unknown revision pkg/apis/v0.0.0
```

```shell
$ go list -m -json -mod=mod all #gosetup

go: github.com/rancher/rancher/pkg/apis@v0.0.0: invalid version: unknown revision pkg/apis/v0.0.0
go: github.com/rancher/rancher/pkg/client@v0.0.0: invalid version: unknown revision pkg/client/v0.0.0
```

3. `go mod edit -replace=github.com/rancher/rancher/pkg/apis=github.com/rancher/rancher/pkg/apis@v0.0.0-20230921211342-9129a7b0496c`
4. `go mod edit -replace=github.com/rancher/rancher/pkg/client=github.com/rancher/rancher/pkg/client@v0.0.0-20230921211342-9129a7b0496c`
```go
module github.com/valaparthvi/rancher

go 1.19

require (
	github.com/onsi/ginkgo/v2 v2.12.1
	github.com/onsi/gomega v1.27.10
	github.com/rancher/rancher v0.0.0-20230921211342-9129a7b0496c
	github.com/stretchr/testify v1.8.4
	k8s.io/api v0.27.5
	k8s.io/apimachinery v0.27.5
)

replace (
	k8s.io/api => k8s.io/api v0.27.4
	k8s.io/client-go => github.com/rancher/client-go v1.27.4-rancher1
	github.com/rancher/rancher/pkg/apis => github.com/rancher/rancher/pkg/apis v0.0.0-20230921211342-9129a7b0496c
	github.com/rancher/rancher/pkg/client => github.com/rancher/rancher/pkg/client v0.0.0-20230921211342-9129a7b0496c
)
```

5. `go mod tidy`
6. `go mod vendor`