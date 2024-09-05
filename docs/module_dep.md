This doc walks through general steps to update pkg dependencies of this project.

0. Write the following to go.mod
```go
module github.com/rancher/hosted-providers-e2e

go 1.22

require (
    github.com/onsi/ginkgo/v2 latest
    github.com/onsi/gomega latest
    github.com/epinio/epinio latest
    github.com/rancher/shepherd fb9a4f1         // rancher/shepherd HEAD commit
    github.com/stretchr/testify latest
    k8s.io/api v0.29.3
    k8s.io/apimachinery v0.29.3
)

replace (
    k8s.io/api => k8s.io/api v0.29.3
    k8s.io/client-go => github.com/rancher/client-go v1.29.3-rancher1
)
```

1. `go mod tidy` - This will resolve all the versions and dependencies, and generate the `go.sum` file.
2. `go mod vendor` - This will generate vendor files; these are ignored by .gitignore.


### Updating the dependecies

There are 4 different kind of packages we are concerned about:
1. Non-kubernetes, non-Rancher pkg: `onsi/ginkgo`, `onsi/gomega`, etc.
    - These pkgs can be updated as usual by either using `latest` or a specific version.
2. Non-kubernetes, Rancher pkg: `rancher/shepherd.
    - `rancher/shepherd` is always updated by using commit HEAD and then resolving using `go mod download`.
    - `github.com/rancher/shepherd` has many branches, currently we are using the `main` branch.
3. Kubernetes, non-Rancher pkg: k8s.io/api, k8s.io/apimachinery.
    - Kubernetes related packages are bumped to support a newer k8s version and must always be in sync to avoid unnecessary conflicts.
    - Ref: [k8s.io/api/tags](https://github.com/kubernetes/api/tags), [k8s.io/apimachinary/tags](https://github.com/kubernetes/apimachinery/tags)
4. Kubernetes, Rancher pkg: rancher/client-go.
    - `rancher/client-go` must always be in sync with Kubernetes related pkgs since it is always released in sync with the Kubernetes pkgs and tags are named aptly too, for a `k8s.io/api:v0.29.3`, there will be a `rancher/client-go:v1.29.3`.
    - Ref: [rancher/client-go/tags](https://github.com/rancher/client-go/tags)This doc walks through general steps to update pkg dependencies of this project.

0. Write the following to go.mod
```go
module github.com/rancher/hosted-providers-e2e

go 1.22

require (
    github.com/onsi/ginkgo/v2 latest
    github.com/onsi/gomega latest
    github.com/epinio/epinio latest
    github.com/rancher/shepherd 3690b7e         // shepherd main HEAD commit
    github.com/stretchr/testify latest
    k8s.io/api v0.29.3
    k8s.io/apimachinery v0.29.3
)

replace (
    k8s.io/api => k8s.io/api v0.29.3
    k8s.io/client-go => github.com/rancher/client-go v1.29.3-rancher1
)
```

1. `go mod tidy` - This will resolve all the versions and dependencies, and generate the `go.sum` file.
2. `go mod vendor` - This will generate vendor files; these are ignored by .gitignore.


### Updating the dependecies

There are 4 different kind of packages we are concerned about:
1. Non-kubernetes, non-Rancher pkg: `onsi/ginkgo`, `onsi/gomega`, etc.
    - These pkgs can be updated as usual by either using `latest` or a specific version.
2. Non-kubernetes, Rancher pkg: `rancher/shepherd.
    - `rancher/shepherd` is always updated by using commit HEAD and then resolving using `go mod download`.
    - `github.com/rancher/shepherd` has many branches, currently we are using the `main` branch.
    - `rancher/shepherd` is the main automation framework we use. It relies on `rancher/rancher` to decide when to upgrade to a newer k8s pkg.
3. Kubernetes, non-Rancher pkg: k8s.io/api, k8s.io/apimachinery.
    - Kubernetes related packages are bumped to support a newer k8s version and must always be in sync to avoid unnecessary conflicts.
    - Ref: [k8s.io/api/tags](https://github.com/kubernetes/api/tags), [k8s.io/apimachinary/tags](https://github.com/kubernetes/apimachinery/tags)
4. Kubernetes, Rancher pkg: rancher/client-go.
    - `rancher/client-go` must always be in sync with Kubernetes related pkgs since it is always released in sync with the Kubernetes pkgs and tags are named aptly too, for a `k8s.io/api:v0.29.3`, there will be a `rancher/client-go:v1.29.3-rancher1`.
    - Ref: [rancher/client-go/tags](https://github.com/rancher/client-go/tags)
