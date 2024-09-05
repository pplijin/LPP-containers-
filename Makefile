##################
### USED BY CI ###
##################

STANDARD_TEST_OPTIONS= -v -r --timeout=2h --keep-going --randomize-all --randomize-suites

install-k3s: ## Install K3s with default options
	curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=${K3S_VERSION} sh -s - --write-kubeconfig-mode 644
	## Wait for K3s to start
	timeout 2m bash -c "until ! kubectl get pod -A 2>/dev/null | grep -Eq 'ContainerCreating|CrashLoopBackOff'; do sleep 1; done"

install-helm: ## Install Helm
	curl --silent --location https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz | tar xz -C .
	sudo mv linux-amd64/helm /usr/local/bin
	sudo chown root:root /usr/local/bin/helm
	sudo rm -rf linux-amd64/ helm-*.tar.gz

install-cert-manager:
	kubectl create namespace cert-manager
	helm repo add jetstack https://charts.jetstack.io
	helm repo update
	helm install cert-manager --namespace cert-manager jetstack/cert-manager \
		--set installCRDs=true \
		--set extraArgs[0]=--enable-certificate-owner-ref=true
	kubectl rollout status deployment cert-manager -n cert-manager --timeout=120s

install-rancher: ## Install Rancher via Helm
	helm repo add rancher-latest https://releases.rancher.com/server-charts/latest
	helm repo update
	helm install rancher --devel rancher-latest/rancher --version ${RANCHER_VERSION} \
		--namespace cattle-system \
		--create-namespace \
		--set global.cattle.psp.enabled=false \
		--set hostname=${RANCHER_HOSTNAME} \
		--set bootstrapPassword=rancherpassword \
		--set replicas=1 \
		--set rancherImageTag=v${RANCHER_VERSION} \
		--wait
	kubectl rollout status deployment rancher -n cattle-system --timeout=300s

deps: 
	go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo
	go install -mod=mod github.com/onsi/gomega
	go mod tidy

prepare-e2e-ci-rancher: install-k3s install-helm install-cert-manager install-rancher ## Tests

e2e-import-tests: deps
	ginkgo ${STANDARD_TEST_OPTIONS} -p --focus "P0Importing" ./hosted/${PROVIDER}/p0/

e2e-provisioning-tests: deps
	ginkgo ${STANDARD_TEST_OPTIONS} -p --focus "P0Provisioning" ./hosted/${PROVIDER}/p0/

#TODO: Parallelize supportmatrix tests
e2e-support-matrix-importing-tests: deps
	ginkgo ${STANDARD_TEST_OPTIONS} --focus "SupportMatrixImporting" ./hosted/${PROVIDER}/support_matrix/

e2e-support-matrix-provisioning-tests: deps
	ginkgo ${STANDARD_TEST_OPTIONS} --focus "SupportMatrixProvisioning" ./hosted/${PROVIDER}/support_matrix/

clean-k3s:
	/usr/local/bin/k3s-uninstall.sh

clean-all: clean-k3s
	/usr/local/bin/helm repo remove rancher-latest jetstack

########################
### LOCAL DEPLOYMENT ###
########################

help: ## Show this Makefile's help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
