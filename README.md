# HOSTED PROVIDERS E2E

[![AKS-E2E_head_2.8](https://github.com/rancher/hosted-providers-e2e/actions/workflows/aks.yaml/badge.svg?branch=main)](https://github.com/rancher/hosted-providers-e2e/actions/workflows/aks.yaml)

[![EKS-E2E_head_2.8](https://github.com/rancher/hosted-providers-e2e/actions/workflows/eks.yaml/badge.svg?branch=main)](https://github.com/rancher/hosted-providers-e2e/actions/workflows/eks.yaml)

[![GKE-E2E_head_2.8](https://github.com/rancher/hosted-providers-e2e/actions/workflows/gke.yaml/badge.svg?branch=main)](https://github.com/rancher/hosted-providers-e2e/actions/workflows/gke.yaml)

## How to run a test locally:

### Environment Variables:
Following are the common environment variables that need to be exported for running a test:
1. RANCHER_HOSTNAME - Public DNS where rancher is running. E.g. ec2-1-2-3-4.ap-south-1.compute.amazonaws.com or 1.2.3.4.sslip.io
2. RANCHER_PASSWORD - Admin Password for login. We currently only test with 'admin' user.
3. CATTLE_TEST_CONFIG: Config file containing cluster and cloud credential information, for e.g. cattle-config-provisioning.yaml and cattle-config-import.yaml in the root directory.
4. PROVIDER: Type of the hosted provider you want to test. Acceptable values - gke, eks, aks
5. DOWNSTREAM_K8S_MINOR_VERSION (optional): Downstream cluster Kubernetes version to test. If the env var is not provided, it uses a provider specific default value.
6. DOWNSTREAM_CLUSTER_CLEANUP (optional): If set to true, downstream cluster will be deleted. Default: false. 
7. RANCHER_CLIENT_DEBUG (optional, debug): Set to true to watch API requests and responses being sent to rancher.

#### To run K8s Chart support test cases:
1. KUBECONFIG: Upstream K8s' Kubeconfig file; usually it is k3s.yaml.

##### Upgrade Scenarios
1. RANCHER_UPGRADE_VERSION: Rancher version to test upgrade. This version can be in the following formats: 2.8.2, 2.8.2-rc3, devel/2.8
2. K8S_UPGRADE_MINOR_VERSION: K8s version to test. This value does not have to be exact, just the X.Y version. For e.g. 1.28. The complete version value will be fetched during the test.
3. RANCHER_VERSION: Base rancher version to begin with. Since chart support tests are basically upgrade scenarios, the base version should be a released version, if it is an unreleased version such as 2.8-head, the test will fail. This version can be in the following formats: 2.8.2, 2.8.2-rc3
4. RANCHER_CHANNEL (Optional): Acceptable values: latest (default), alpha, stable, prime

Note: These are E2E tests, so rancher (version=`RANCHER_VERSION`) will be installed by the test.

#### To run GKE:
1. GCP_CREDENTIALS - a Service Account with a JSON private key and provide the JSON here. These IAM roles are required:
   - Compute Engine: Compute Viewer (roles/compute.viewer)
   - Project: Viewer (roles/viewer)
   - Kubernetes Engine: Kubernetes Engine Admin (roles/container.admin)
   - Service Accounts: Service Account User (roles/iam.serviceAccountUser)
2. GKE_PROJECT_ID - Name of the Google Cloud Project
3. GKE_ZONE - Zone in which GKE must be provisioned (default: 'asia-south2-c'). This environment variable takes precedence over the config file variable.

#### To run EKS:
1. AWS_ACCESS_KEY_ID - AWS Access Key
2. AWS_SECRET_ACCESS_KEY - AWS Secret Key
3. EKS_REGION - Region in which EKS must be provisioned (default: 'ap-south-1'). This environment variable takes precedence over the config file variable.

#### To run AKS:
1. AKS_CLIENT_ID - Azure Client ID [Check Microsoft Entra ID to create or fetch value from an existing one](https://learn.microsoft.com/en-us/entra/identity-platform/howto-create-service-principal-portal)
2. AKS_CLIENT_SECRET - Azure Client Secret [Check Microsoft Entra ID to create or fetch value from an existing one](https://learn.microsoft.com/en-us/entra/identity-platform/howto-create-service-principal-portal)
3. AKS_SUBSCRIPTION_ID - Azure Subscription ID (In this case it is similar to a Google Cloud Project, but the value is an ID). [Check Azure Subscriptions](https://learn.microsoft.com/en-us/microsoft-365/enterprise/subscriptions-licenses-accounts-and-tenants-for-microsoft-cloud-offerings?view=o365-worldwide#subscriptions)
4. AKS_REGION - Region in which AKS must be provisioned (default: 'centralindia'). This environment variable takes precedence over the config file variable.

**Note:** It is advisable that all the Hosted Provider cluster be provisioned in APAC region, this is because we want to geolocalize all the resources created by hosted provider.

### Makefile targets to run tests
1. `make e2e-provisioning-tests` - Covers the _P0Provisioning_ test suite for a given `${PROVIDER}`
2. `make e2e-import-tests` - Covers the _P0Importing_ test suite for a given `${PROVIDER}`
3. `make e2e-support-matrix-importing-tests` - Covers the _SupportMatrixImporting_ test suite for a given `${PROVIDER}`
4. `make e2e-support-matrix-provisioning-tests` - Covers the _SupportMatrixProvisioning_ test suite for a given `${PROVIDER}`
5. `make e2e-k8s-chart-support-provisioning-tests` - Focuses on _K8sChartSupportProvisioning_ for a given `${PROVIDER}`
6. `make e2e-k8s-chart-support-importing-tests` - Focuses on _K8sChartSupportImport_ for a given `${PROVIDER}`
7. `make e2e-k8s-chart-support-importing-tests-upgrade` - Focuses on _K8sChartSupportUpgradeImport_ for a given `${PROVIDER}`
8. `make e2e-k8s-chart-support-provisioning-tests-upgrade` - Focuses on _K8sChartSupportUpgradeProvisioning_ for a given `${PROVIDER}`

Run `make help` to know about other targets.

### Example
**GKE Provisioning Tests**
```shell
GKE_PROJECT_ID=some-project GCP_CREDENTIALS=<credentials-json> PROVIDER=gke RANCHER_HOSTNAME=ec2-1-2-3-4.ap-south-1.compute.amazonaws.com RANCHER_PASSWORD=admin123 CATTLE_TEST_CONFIG=/home/pvala/go/src/github.com/rancher/hosted-providers-e2e/hosted/gke/cattle-config-provisioning.yaml make e2e-provisioning-tests
```

**GKE Import Tests**
```shell
GKE_PROJECT_ID=some-project GCP_CREDENTIALS=<credentials-json> PROVIDER=gke RANCHER_HOSTNAME=ec2-1-2-3-4.ap-south-1.compute.amazonaws.com RANCHER_PASSWORD=admin123 CATTLE_TEST_CONFIG=/home/pvala/go/src/github.com/rancher/hosted-providers-e2e/hosted/gke/cattle-config-import.yaml make e2e-import-tests
```

**EKS Provisioning Tests**
```shell
EKS_REGION=ap-south-1 AWS_ACCESS_KEY_ID=<key-id> AWS_SECRET_ACCESS_KEY=<key> PROVIDER=eks RANCHER_HOSTNAME=ec2-1-2-3-4.ap-south-1.compute.amazonaws.com RANCHER_PASSWORD=admin123 CATTLE_TEST_CONFIG=/home/pvala/go/src/github.com/rancher/hosted-providers-e2e/hosted/gke/cattle-config-provisioning.yaml make e2e-provisioning-tests
```

**EKS Import Tests**
```shell
EKS_REGION=ap-south-1 AWS_ACCESS_KEY_ID=<key-id> AWS_SECRET_ACCESS_KEY=<key> PROVIDER=eks RANCHER_HOSTNAME=ec2-1-2-3-4.ap-south-1.compute.amazonaws.com RANCHER_PASSWORD=admin123 CATTLE_TEST_CONFIG=/home/pvala/go/src/github.com/rancher/hosted-providers-e2e/hosted/gke/cattle-config-import.yaml make e2e-import-tests
```

**AKS Provisioning Tests**
```shell
AKS_REGION=centralindia AKS_CLIENT_ID=<client-id> AKS_CLIENT_SECRET=<secret> AKS_SUBSCRIPTION_ID=<subscription-id> PROVIDER=aks RANCHER_HOSTNAME=ec2-1-2-3-4.ap-south-1.compute.amazonaws.com RANCHER_PASSWORD=admin123 CATTLE_TEST_CONFIG=/home/pvala/go/src/github.com/rancher/hosted-providers-e2e/hosted/gke/cattle-config-provisioning.yaml make e2e-provisioning-tests
```

**AKS Import Tests**
```shell
AKS_REGION=centralindia AKS_CLIENT_ID=<client-id> AKS_CLIENT_SECRET=<secret> AKS_SUBSCRIPTION_ID=<subscription-id> PROVIDER=aks RANCHER_HOSTNAME=ec2-1-2-3-4.ap-south-1.compute.amazonaws.com RANCHER_PASSWORD=admin123 CATTLE_TEST_CONFIG=/home/pvala/go/src/github.com/rancher/hosted-providers-e2e/hosted/gke/cattle-config-import.yaml make e2e-import-tests
```
