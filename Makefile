##################
### USED BY CI ###
##################

STANDARD_TEST_OPTIONS= -v -r --timeout=3h --keep-going --randomize-all --randomize-suites
BUILD_DATE= $(shell date +'%Y%m%d')

install-k3s: ## Install K3s with default options; installed on the local machine
	curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=${K3S_VERSION} sh -s - --write-kubeconfig-mode 644
	## Wait for K3s to start
	timeout 2m bash -c "until ! kubectl get pod -A 2>/dev/null | grep -Eq 'ContainerCreating|CrashLoopBackOff'; do sleep 1; done"

install-helm: ## Install Helm on the local machine
	curl --silent --location https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz | tar xz -C .
	sudo mv linux-amd64/helm /usr/local/bin
	sudo chown root:root /usr/local/bin/helm
	sudo rm -rf linux-amd64/ helm-*.tar.gz

install-cert-manager: ## Install cert-manager via Helm on the k8s cluster
	kubectl create namespace cert-manager
	helm repo add jetstack https://charts.jetstack.io
	helm repo update
	helm install cert-manager --namespace cert-manager jetstack/cert-manager \
		--set installCRDs=true \
		--set extraArgs[0]=--enable-certificate-owner-ref=true
	kubectl rollout status deployment cert-manager -n cert-manager --timeout=120s

install-rancher: ## Install Rancher via Helm on the k8s cluster
	helm repo add rancher-prime-devel https://charts.optimus.rancher.io/server-charts/latest
	helm repo update
	helm install rancher --devel rancher-prime-devel/rancher \
		--namespace cattle-system \
		--create-namespace \
		--version ${RANCHER_VERSION} \
		--set global.cattle.psp.enabled=false \
		--set hostname=${RANCHER_HOSTNAME} \
		--set bootstrapPassword=${RANCHER_PASSWORD} \
		--set replicas=1 \
		--set rancherImageTag=v${RANCHER_VERSION} \
		--set rancherImage=stgregistry.suse.com/rancher/rancher \
		--set 'extraEnv[0].name=CATTLE_AGENT_IMAGE' \
		--set 'extraEnv[0].value=stgregistry.suse.com/rancher/rancher-agent:v${RANCHER_VERSION}' \
		--wait
	kubectl rollout status deployment rancher -n cattle-system --timeout=300s

install-rancher-hosted-nightly-chart: ## Install Rancher via Helm with hosted providers nightly chart
	helm repo add rancher-prime-devel https://charts.optimus.rancher.io/server-charts/latest
	helm repo update
	helm install rancher --devel rancher-prime-devel/rancher \
		--namespace cattle-system \
		--version ${RANCHER_VERSION} \
		--create-namespace \
		--set global.cattle.psp.enabled=false \
		--set hostname=${RANCHER_HOSTNAME} \
		--set bootstrapPassword=${RANCHER_PASSWORD} \
		--set replicas=1 \
		--set rancherImageTag=v${RANCHER_VERSION} \
		--set 'extraEnv[0].name=CATTLE_SKIP_HOSTED_CLUSTER_CHART_INSTALLATION' \
		--set-string 'extraEnv[0].value=true' \
		--set rancherImage=stgregistry.suse.com/rancher/rancher \
		--set 'extraEnv[0].name=CATTLE_AGENT_IMAGE' \
		--set 'extraEnv[0].value=stgregistry.suse.com/rancher/rancher-agent:v${RANCHER_VERSION}' \
		--wait
	kubectl rollout status deployment rancher -n cattle-system --timeout=300s
	helm install ${PROVIDER}-operator-crds  oci://ttl.sh/${PROVIDER}-operator/rancher-${PROVIDER}-operator-crd --version ${BUILD_DATE}
	helm install ${PROVIDER}-operator oci://ttl.sh/${PROVIDER}-operator/rancher-${PROVIDER}-operator --version ${BUILD_DATE} --namespace cattle-system

deps: ## Install the Go dependencies
	go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo
	go install -mod=mod github.com/onsi/gomega
	go mod tidy

prepare-e2e-ci-rancher-hosted-nightly-chart: install-k3s install-helm install-cert-manager install-rancher-hosted-nightly-chart ## Setup Rancher with nightly hosted provider charts on the local machine
prepare-e2e-ci-rancher: install-k3s install-helm install-cert-manager install-rancher ## Setup Rancher on the local machine

e2e-import-tests: deps	## Run the 'P0Import' test suite for a given ${PROVIDER}
	ginkgo ${STANDARD_TEST_OPTIONS} --nodes 2 --focus "P0Import" ./hosted/${PROVIDER}/p0/

e2e-provisioning-tests: deps ## Run the 'P0Provisioning' test suite for a given ${PROVIDER}
	ginkgo ${STANDARD_TEST_OPTIONS} --nodes 2 --focus "P0Provisioning" ./hosted/${PROVIDER}/p0/

e2e-p1-import-tests: deps	## Run the 'P1Import' test suite for a given ${PROVIDER}
	ginkgo ${STANDARD_TEST_OPTIONS} --focus "P1Import" ./hosted/${PROVIDER}/p1/

e2e-p1-provisioning-tests: deps ## Run the 'P1Provisioning' test suite for a given ${PROVIDER}
	ginkgo ${STANDARD_TEST_OPTIONS} --focus "P1Provisioning" ./hosted/${PROVIDER}/p1/


# Support Matrix test has not been parallelized for EKS because we hit resource limits when running it in parallel
e2e-support-matrix-import-tests: deps ## Run the 'SupportMatrixImport' test suite for a given ${PROVIDER}
ifeq (${PROVIDER}, eks)
	ginkgo ${STANDARD_TEST_OPTIONS} --focus "SupportMatrixImport" ./hosted/${PROVIDER}/support_matrix/
else
	ginkgo ${STANDARD_TEST_OPTIONS} --nodes 2 --focus "SupportMatrixImport" ./hosted/${PROVIDER}/support_matrix/
endif


e2e-support-matrix-provisioning-tests: deps ## Run the 'SupportMatrixProvisioning' test suite for a given ${PROVIDER}
ifeq (${PROVIDER}, eks)
	ginkgo ${STANDARD_TEST_OPTIONS} --focus "SupportMatrixProvisioning" ./hosted/${PROVIDER}/support_matrix/
else
	ginkgo ${STANDARD_TEST_OPTIONS} --nodes 2 --focus "SupportMatrixProvisioning" ./hosted/${PROVIDER}/support_matrix/
endif



e2e-k8s-chart-support-import-tests-upgrade: deps ## Run the 'K8sChartSupportUpgradeImport' test suite for a given ${PROVIDER}
	ginkgo ${STANDARD_TEST_OPTIONS} --focus "K8sChartSupportUpgradeImport" ./hosted/${PROVIDER}/k8s_chart_support/upgrade

e2e-k8s-chart-support-provisioning-tests-upgrade: deps ## Run the 'K8sChartSupportUpgradeProvisioning' test suite for a given ${PROVIDER}
	ginkgo ${STANDARD_TEST_OPTIONS} --focus "K8sChartSupportUpgradeProvisioning" ./hosted/${PROVIDER}/k8s_chart_support/upgrade

e2e-k8s-chart-support-import-tests: deps ## Run the 'K8sChartSupportImport' test suite for a given ${PROVIDER}
	ginkgo ${STANDARD_TEST_OPTIONS} --focus "K8sChartSupportImport" ./hosted/${PROVIDER}/k8s_chart_support

e2e-k8s-chart-support-provisioning-tests: deps ## Run the 'K8sChartSupportProvisioning' test suite for a given ${PROVIDER}
	ginkgo ${STANDARD_TEST_OPTIONS} --focus "K8sChartSupportProvisioning" ./hosted/${PROVIDER}/k8s_chart_support

e2e-sync-provisioning-tests: deps ## Run "SyncProvisioning" test suite for a given ${PROVIDER}
	ginkgo ${STANDARD_TEST_OPTIONS} --nodes 2 --focus "SyncProvisioning" ./hosted/${PROVIDER}/p1

e2e-sync-import-tests: deps ## Run "SyncImport" test suite for a given ${PROVIDER}
	ginkgo ${STANDARD_TEST_OPTIONS} --focus "SyncImport" ./hosted/${PROVIDER}/p1

clean-k3s:	## Uninstall k3s cluster
	/usr/local/bin/k3s-uninstall.sh

clean-all: clean-k3s	## Cleanup the Helm repo
	/usr/local/bin/helm repo remove rancher-latest jetstack

########################
### LOCAL DEPLOYMENT ###
########################

help: ## Show this Makefile's help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
