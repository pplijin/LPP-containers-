# Tests description for aks/p0

## `p0_import_test.go`

- **Describe:** P0Import
    - **It:** should successfully import the cluster & add, delete, scale nodepool
      -  **By:** checking cluster name is same
      -  **By:** checking service account token secret
      -  **By:** checking all management nodes are ready
      -  **By:** checking all pods are ready
      -  **By:** scaling up the nodepool
      -  **By:** scaling down the nodepool
      -  **By:** adding a nodepool/s
      -  **By:** deleting the nodepool
  - **Context:** Upgrading K8s version
    - **It:** should be able to upgrade k8s version of the cluster
      -  **By:** upgrading the ControlPlane
      -  **By:** upgrading the NodePools

## `p0_provisioning_test.go`

- **Describe:** P0Provisioning
    - **It:** should successfully provision the cluster & add, delete, scale nodepool
      -  **By:** checking cluster name is same
      -  **By:** checking service account token secret
      -  **By:** checking all management nodes are ready
      -  **By:** checking all pods are ready
      -  **By:** adding a nodepool
      -  **By:** deleting the nodepool
      -  **By:** scaling up the nodepool
      -  **By:** scaling down the nodepool
  - **Context:** Upgrading K8s version
    - **It:** should be able to upgrade k8s version of the cluster
      -  **By:** upgrading the ControlPlane
      -  **By:** upgrading the NodePools

# Tests description for aks/support_matrix

## `support_matrix_import_test.go`

- **Describe:** SupportMatrixImport
    - **It:** should successfully import the cluster
      -  **By:** checking cluster name is same
      -  **By:** checking service account token secret
      -  **By:** checking all management nodes are ready
      -  **By:** checking all pods are ready

## `support_matrix_provisioning_test.go`

- **Describe:** SupportMatrixProvisioning
    - **It:** should successfully provision the cluster
      -  **By:** checking cluster name is same
      -  **By:** checking service account token secret
      -  **By:** checking all management nodes are ready
      -  **By:** checking all pods are ready

# Tests description for eks/p0

## `p0_import_test.go`

- **Describe:** P0Import
    - **It:** should successfully import the cluster & add, delete, scale nodepool
      -  **By:** checking cluster name is same
      -  **By:** checking service account token secret
      -  **By:** checking all management nodes are ready
      -  **By:** checking all pods are ready
      -  **By:** scaling up the NodeGroup
      -  **By:** scaling down the NodeGroup
      -  **By:** adding a NodeGroup/s
      -  **By:** deleting the NodeGroup
  - **Context:** Upgrading K8s version
    - **It:** should be able to upgrade k8s version of the cluster
      -  **By:** upgrading the ControlPlane
      -  **By:** upgrading the NodeGroups

## `p0_provisioning_test.go`

- **Describe:** P0Provisioning
    - **It:** should successfully provision the cluster & add, delete, scale nodepool
      -  **By:** checking cluster name is same
      -  **By:** checking service account token secret
      -  **By:** checking all management nodes are ready
      -  **By:** checking all pods are ready
      -  **By:** scaling up the NodeGroup
      -  **By:** scaling down the NodeGroup
      -  **By:** adding a NodeGroup
      -  **By:** deleting the NodeGroup
  - **Context:** Upgrading K8s version
    - **It:** should be able to upgrade k8s version of the cluster
      -  **By:** upgrading the ControlPlane
      -  **By:** upgrading the NodeGroups

# Tests description for eks/support_matrix

## `support_matrix_import_test.go`

- **Describe:** SupportMatrixImport
    - **It:** should successfully import the cluster
      -  **By:** checking cluster name is same
      -  **By:** checking service account token secret
      -  **By:** checking all management nodes are ready
      -  **By:** checking all pods are ready

## `support_matrix_provisioning_test.go`

- **Describe:** SupportMatrixProvisioning
    - **It:** should successfully provision the cluster
      -  **By:** checking cluster name is same
      -  **By:** checking service account token secret
      -  **By:** checking all management nodes are ready
      -  **By:** checking all pods are ready

# Tests description for gke/p0

## `p0_import_test.go`

- **Describe:** P0Import
    - **It:** should successfully import the cluster & add, delete, scale nodepool
      -  **By:** checking cluster name is same
      -  **By:** checking service account token secret
      -  **By:** checking all management nodes are ready
      -  **By:** checking all pods are ready
      -  **By:** scaling up the nodepool
      -  **By:** scaling down the nodepool
      -  **By:** adding a nodepool
      -  **By:** deleting the nodepool
  - **Context:** Upgrading K8s version
    - **It:** should be able to upgrade k8s version of the cluster
      -  **By:** upgrading the Controlplane & NodePools

## `p0_provisioning_test.go`

- **Describe:** P0Provisioning
    - **It:** should successfully provision the cluster & add, delete, scale nodepool
      -  **By:** checking cluster name is same
      -  **By:** checking service account token secret
      -  **By:** checking all management nodes are ready
      -  **By:** checking all pods are ready
      -  **By:** scaling up the nodepool
      -  **By:** scaling down the nodepool
      -  **By:** adding a nodepool
      -  **By:** deleting the nodepool
  - **Context:** Upgrading K8s version
    - **It:** should be able to upgrade k8s version of the cluster
      -  **By:** upgrading the ControlPlane
      -  **By:** upgrading the NodePools

# Tests description for gke/support_matrix

## `support_matrix_import_test.go`

- **Describe:** SupportMatrixImport
    - **It:** should successfully import the cluster
      -  **By:** checking cluster name is same
      -  **By:** checking service account token secret
      -  **By:** checking all management nodes are ready
      -  **By:** checking all pods are ready

## `support_matrix_provisioning_test.go`

- **Describe:** SupportMatrixProvisioning
    - **It:** should successfully provision the cluster
      -  **By:** checking cluster name is same
      -  **By:** checking service account token secret
      -  **By:** checking all management nodes are ready
      -  **By:** checking all pods are ready
