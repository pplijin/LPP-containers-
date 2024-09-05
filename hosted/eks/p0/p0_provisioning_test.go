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
	"github.com/rancher/shepherd/pkg/config"

	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/eks"
	nodestat "github.com/rancher/shepherd/extensions/nodes"
	"github.com/rancher/shepherd/extensions/workloads/pods"

	"github.com/rancher/hosted-providers-e2e/hosted/eks/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

var _ = Describe("P0Provisioning", func() {
	var cluster *management.Cluster

	When("a cluster is created", func() {

		BeforeEach(func() {
			var err error
			eksConfig := new(eks.ClusterConfig)
			config.LoadAndUpdateConfig(eks.EKSClusterConfigConfigurationFileKey, eksConfig, func() {
				eksConfig.Region = region
				eksConfig.Tags = helper.GetTags()
				eksConfig.KubernetesVersion = &k8sVersion
			})
			cluster, err = eks.CreateEKSHostedCluster(ctx.RancherClient, clusterName, ctx.CloudCred.ID, false, false, false, false, map[string]string{})
			Expect(err).To(BeNil())
			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherClient)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			// TODO Check if EKS cluster deleted on AWS
			err := helper.DeleteEKSHostCluster(cluster, ctx.RancherClient)
			Expect(err).To(BeNil())
		})

		It("should successfully provision the cluster & add, delete, scale nodepool", func() {

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

			currentNodeGroupNumber := len(cluster.EKSConfig.NodeGroups)
			initialNodeCount := *cluster.EKSConfig.NodeGroups[0].DesiredSize

			By("scaling up the NodeGroup", func() {
				var err error
				cluster, err = helper.ScaleNodeGroup(cluster, ctx.RancherClient, initialNodeCount+1)
				Expect(err).To(BeNil())
				err = clusters.WaitClusterToBeUpgraded(ctx.RancherClient, cluster.ID)
				Expect(err).To(BeNil())
				for i := range cluster.EKSConfig.NodeGroups {
					Expect(*cluster.EKSConfig.NodeGroups[i].DesiredSize).To(BeNumerically("==", initialNodeCount+1))
				}
			})

			By("scaling down the NodeGroup", func() {
				var err error
				cluster, err = helper.ScaleNodeGroup(cluster, ctx.RancherClient, initialNodeCount)
				Expect(err).To(BeNil())
				err = clusters.WaitClusterToBeUpgraded(ctx.RancherClient, cluster.ID)
				Expect(err).To(BeNil())
				for i := range cluster.EKSConfig.NodeGroups {
					Expect(*cluster.EKSConfig.NodeGroups[i].DesiredSize).To(BeNumerically("==", initialNodeCount))
				}
			})

			By("adding a NodeGroup", func() {
				var err error
				cluster, err = helper.AddNodeGroup(cluster, increaseBy, ctx.RancherClient)
				Expect(err).To(BeNil())
				err = clusters.WaitClusterToBeUpgraded(ctx.RancherClient, cluster.ID)
				Expect(err).To(BeNil())
				Expect(len(cluster.EKSConfig.NodeGroups)).To(BeNumerically("==", currentNodeGroupNumber+1))
			})
			By("deleting the NodeGroup", func() {
				var err error
				cluster, err = helper.DeleteNodeGroup(cluster, ctx.RancherClient)
				Expect(err).To(BeNil())
				err = clusters.WaitClusterToBeUpgraded(ctx.RancherClient, cluster.ID)
				Expect(err).To(BeNil())
				Expect(len(cluster.EKSConfig.NodeGroups)).To(BeNumerically("==", currentNodeGroupNumber))

			})

		})

		Context("Upgrading K8s version", func() {
			var upgradeToVersion *string
			BeforeEach(func() {
				versions, err := helper.ListEKSAvailableVersions(ctx.RancherClient, cluster.ID)
				Expect(err).To(BeNil())
				Expect(versions).ToNot(BeEmpty())
				upgradeToVersion = &versions[0]
			})

			It("should be able to upgrade k8s version of the cluster", func() {
				By("upgrading the ControlPlane", func() {
					var err error
					cluster, err = helper.UpgradeClusterKubernetesVersion(cluster, upgradeToVersion, ctx.RancherClient)
					Expect(err).To(BeNil())
					err = clusters.WaitClusterToBeUpgraded(ctx.RancherClient, cluster.ID)
					Expect(err).To(BeNil())
					Expect(cluster.EKSConfig.KubernetesVersion).To(BeEquivalentTo(upgradeToVersion))
				})

				By("upgrading the NodeGroups", func() {
					var err error
					cluster, err = helper.UpgradeNodeKubernetesVersion(cluster, upgradeToVersion, ctx.RancherClient)
					Expect(err).To(BeNil())
					err = clusters.WaitClusterToBeUpgraded(ctx.RancherClient, cluster.ID)
					Expect(err).To(BeNil())
					for _, ng := range cluster.EKSConfig.NodeGroups {
						Expect(ng.Version).To(BeEquivalentTo(upgradeToVersion))
					}
				})
			})
		})

	})

})
