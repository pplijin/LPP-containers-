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

package p0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/gke"
	nodestat "github.com/rancher/shepherd/extensions/nodes"
	"github.com/rancher/shepherd/extensions/workloads/pods"
	"github.com/rancher/shepherd/pkg/config"

	"github.com/rancher/hosted-providers-e2e/hosted/gke/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

var _ = Describe("P0Importing", func() {

	When("a cluster is created", func() {
		var cluster *management.Cluster

		BeforeEach(func() {
			var err error
			err = helper.CreateGKEClusterOnGCloud(zone, clusterName, project, k8sVersion)
			Expect(err).To(BeNil())

			gkeConfig := new(helper.ImportClusterConfig)
			config.LoadAndUpdateConfig(gke.GKEClusterConfigConfigurationFileKey, gkeConfig, func() {
				gkeConfig.ProjectID = project
				gkeConfig.Zone = zone
				labels := helper.GetLabels()
				gkeConfig.Labels = &labels
				for _, np := range gkeConfig.NodePools {
					np.Version = &k8sVersion
				}
			})
			cluster, err = helper.ImportGKEHostedCluster(ctx.RancherClient, clusterName, ctx.CloudCred.ID, false, false, false, false, map[string]string{})
			Expect(err).To(BeNil())
			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherClient)
			Expect(err).To(BeNil())
			// Workaround to add new Nodegroup till https://github.com/rancher/aks-operator/issues/251 is fixed
			cluster.GKEConfig = cluster.GKEStatus.UpstreamSpec
		})
		AfterEach(func() {
			err := helper.DeleteGKEHostCluster(cluster, ctx.RancherClient)
			Expect(err).To(BeNil())
			err = helper.DeleteGKEClusterOnGCloud(zone, project, clusterName)
			Expect(err).To(BeNil())
		})

		It("should successfully import the cluster & add, delete, scale nodepool", func() {

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

			currentNodePoolNumber := len(cluster.GKEConfig.NodePools)
			initialNodeCount := *cluster.GKEConfig.NodePools[0].InitialNodeCount

			By("scaling up the nodepool", func() {
				var err error
				cluster, err = helper.ScaleNodePool(cluster, ctx.RancherClient, initialNodeCount+1)
				Expect(err).To(BeNil())
				err = clusters.WaitClusterToBeUpgraded(ctx.RancherClient, cluster.ID)
				Expect(err).To(BeNil())
				for i := range cluster.GKEConfig.NodePools {
					Expect(*cluster.GKEConfig.NodePools[i].InitialNodeCount).To(BeNumerically("==", initialNodeCount+1))
				}
			})

			By("scaling down the nodepool", func() {
				var err error
				cluster, err = helper.ScaleNodePool(cluster, ctx.RancherClient, initialNodeCount)
				Expect(err).To(BeNil())
				err = clusters.WaitClusterToBeUpgraded(ctx.RancherClient, cluster.ID)
				Expect(err).To(BeNil())
				for i := range cluster.GKEConfig.NodePools {
					Expect(*cluster.GKEConfig.NodePools[i].InitialNodeCount).To(BeNumerically("==", initialNodeCount))
				}
			})

			By("adding a nodepool", func() {
				var err error
				cluster, err = helper.AddNodePool(cluster, increaseBy, ctx.RancherClient)
				Expect(err).To(BeNil())
				err = clusters.WaitClusterToBeUpgraded(ctx.RancherClient, cluster.ID)
				Expect(err).To(BeNil())
				Expect(len(cluster.GKEConfig.NodePools)).To(BeNumerically("==", currentNodePoolNumber+1))
			})
			By("deleting the nodepool", func() {
				var err error
				cluster, err = helper.DeleteNodePool(cluster, ctx.RancherClient)
				Expect(err).To(BeNil())
				err = clusters.WaitClusterToBeUpgraded(ctx.RancherClient, cluster.ID)
				Expect(err).To(BeNil())
				Expect(len(cluster.GKEConfig.NodePools)).To(BeNumerically("==", currentNodePoolNumber))

			})

		})
		Context("Upgrading K8s version", func() {
			var upgradeToVersion *string
			BeforeEach(func() {
				versions, err := helper.ListGKEAvailableVersions(ctx.RancherClient, cluster.ID)
				Expect(err).To(BeNil())
				Expect(versions).ToNot(BeEmpty())
				upgradeToVersion = &versions[0]
			})

			It("should be able to upgrade k8s version of the cluster", func() {
				By("upgrading the Controlplane & NodePools", func() {
					var err error
					cluster, err = helper.UpgradeKubernetesVersion(cluster, upgradeToVersion, ctx.RancherClient, true)
					Expect(err).To(BeNil())
					err = clusters.WaitClusterToBeUpgraded(ctx.RancherClient, cluster.ID)
					Expect(err).To(BeNil())

					Expect(cluster.GKEConfig.KubernetesVersion).To(BeEquivalentTo(upgradeToVersion))
					for _, np := range cluster.GKEConfig.NodePools {
						Expect(np.Version).To(BeEquivalentTo(upgradeToVersion))
					}
				})
			})
		})

	})
})
