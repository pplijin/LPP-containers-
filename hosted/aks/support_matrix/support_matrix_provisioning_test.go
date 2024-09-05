/*
Copyright © 2023 - 2024 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package support_matrix_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"fmt"

	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/aks"
	nodestat "github.com/rancher/shepherd/extensions/nodes"
	"github.com/rancher/shepherd/extensions/workloads/pods"
	"github.com/rancher/shepherd/pkg/config"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"

	"github.com/rancher/hosted-providers-e2e/hosted/aks/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

var _ = Describe("SupportMatrixProvisioning", func() {

	for _, version := range availableVersionList {
		version := version

		When(fmt.Sprintf("a cluster is created with kubernetes version %s", version), func() {
			var (
				clusterName string
				cluster     *management.Cluster
			)
			BeforeEach(func() {
				clusterName = namegen.AppendRandomString("akshostcluster")
				var err error
				aksConfig := new(aks.ClusterConfig)
				config.LoadAndUpdateConfig(aks.AKSClusterConfigConfigurationFileKey, aksConfig, func() {
					aksConfig.ResourceGroup = clusterName
					dnsPrefix := clusterName + "-dns"
					aksConfig.DNSPrefix = &dnsPrefix
					aksConfig.ResourceLocation = location
					aksConfig.KubernetesVersion = &version
					aksConfig.Tags = helper.GetTags()
				})
				cluster, err = aks.CreateAKSHostedCluster(ctx.RancherClient, clusterName, ctx.CloudCred.ID, false, false, false, false, map[string]string{})
				Expect(err).To(BeNil())
				cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherClient)
				Expect(err).To(BeNil())
			})
			AfterEach(func() {
				err := helper.DeleteAKSHostCluster(cluster, ctx.RancherClient)
				Expect(err).To(BeNil())
				err = helper.DeleteAKSClusteronAzure(clusterName)
				Expect(err).To(BeNil())
			})

			It("should successfully provision the cluster", func() {
				By("checking cluster name is same", func() {
					Expect(cluster.Name).To(BeEquivalentTo(clusterName))
				})

				By("checking service account token secret", func() {
					success, err := clusters.CheckServiceAccountTokenSecret(ctx.RancherClient, clusterName)
					Expect(err).To(BeNil())
					Expect(success).To(BeTrue())
				})

				By("checking all management nodes are ready", func() {
					err := nodestat.AllManagementNodeReady(ctx.RancherClient, cluster.ID, helpers.Timeout)
					Expect(err).To(BeNil())
				})

				By("checking all pods are ready", func() {
					podErrors := pods.StatusPods(ctx.RancherClient, cluster.ID)
					Expect(podErrors).To(BeEmpty())
				})
			})
		})
	}
})
