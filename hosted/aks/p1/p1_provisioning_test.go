package p1_test

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters/aks"
	"k8s.io/utils/pointer"

	"github.com/rancher/hosted-providers-e2e/hosted/aks/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

var _ = Describe("P1Provisioning", func() {
	var cluster *management.Cluster
	var k8sVersion string
	BeforeEach(func() {
		var err error
		k8sVersion, err = helper.GetK8sVersion(ctx.RancherAdminClient, ctx.CloudCred.ID, location, false)
		Expect(err).NotTo(HaveOccurred())
		GinkgoLogr.Info(fmt.Sprintf("Using K8s version %s for cluster %s", k8sVersion, clusterName))
	})
	AfterEach(func() {
		if ctx.ClusterCleanup && (cluster != nil && cluster.ID != "") {
			err := helper.DeleteAKSHostCluster(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())
		} else {
			fmt.Println("Skipping downstream cluster deletion: ", clusterName)
		}
	})
	When("a cluster with invalid config is created", func() {
		It("should fail to create a cluster with 0 nodecount", func() {
			testCaseID = 186
			updateFunc := func(aksConfig *aks.ClusterConfig) {
				nodepools := *aksConfig.NodePools
				for i := range nodepools {
					nodepools[i].NodeCount = pointer.Int64(0)
				}
				aksConfig.NodePools = &nodepools
			}
			var err error
			cluster, err = helper.CreateAKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, k8sVersion, location, updateFunc)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				cluster, err = ctx.RancherAdminClient.Management.Cluster.ByID(cluster.ID)
				Expect(err).NotTo(HaveOccurred())
				return cluster.Transitioning == "error" && strings.Contains(cluster.TransitioningMessage, "agentPoolProfile.count was 0. It must be greater or equal to minCount:1 and less than or equal to maxCount:1000")
			}, "1m", "2s").Should(BeTrue())
		})
		It("should fail to create a cluster with 0 nodepool", func() {
			testCaseID = 187
			updateFunc := func(aksConfig *aks.ClusterConfig) {
				aksConfig.NodePools = &[]aks.NodePool{}
			}
			var err error
			cluster, err = helper.CreateAKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, k8sVersion, location, updateFunc)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				cluster, err = ctx.RancherAdminClient.Management.Cluster.ByID(cluster.ID)
				Expect(err).NotTo(HaveOccurred())
				return cluster.Transitioning == "error" && cluster.TransitioningMessage == "at least one NodePool with mode System is required"
			}, "1m", "2s").Should(BeTrue())
		})
	})
	When("a cluster is created", func() {
		BeforeEach(func() {
			var err error
			cluster, err = helper.CreateAKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, k8sVersion, location, nil)
			Expect(err).To(BeNil())
			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should be able to update autoscaling", func() {
			testCaseID = 176
			updateAutoScaling(cluster, ctx.RancherAdminClient)
		})

		It("recreating a cluster while it is being deleted should recreate the cluster", func() {
			testCaseID = 219

			err := helper.DeleteAKSHostCluster(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())

			// Wait until the cluster begins deletion process before recreating
			Eventually(func() bool {
				exists, err := helper.ClusterExistsOnAzure(clusterName, cluster.AKSConfig.ResourceGroup)
				Expect(err).To(BeNil())
				return exists
			}, "1m", "5s").Should(BeFalse())

			cluster, err = helper.CreateAKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, k8sVersion, location, nil)
			Expect(err).To(BeNil())

			// wait until the error is visible on the provisioned cluster
			Eventually(func() bool {
				cluster, err = ctx.RancherAdminClient.Management.Cluster.ByID(cluster.ID)
				Expect(err).To(BeNil())
				return cluster.State == "provisioning" && cluster.Transitioning == "error" && strings.Contains(cluster.TransitioningMessage, "an AKSClusterConfig exists with the same name")
			}, "30s", "2s").Should(BeTrue())

			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())
		})

	})

	When("a cluster is created for upgrade", func() {
		BeforeEach(func() {
			var err error
			k8sVersion, err = helper.GetK8sVersion(ctx.RancherAdminClient, ctx.CloudCred.ID, location, true)
			Expect(err).NotTo(HaveOccurred())
			GinkgoLogr.Info(fmt.Sprintf("Using K8s version %s for cluster %s", k8sVersion, clusterName))

			cluster, err = helper.CreateAKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, k8sVersion, location, nil)
			Expect(err).To(BeNil())
			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
			Expect(err).NotTo(HaveOccurred())
		})

		It("NP cannot be upgraded to k8s version greater than CP k8s version", func() {
			testCaseID = 183
			npUpgradeToVersionGTCPCheck(cluster, ctx.RancherAdminClient)
		})
	})

	It("deleting a cluster while it is in creation state should delete it from rancher and cloud console", func() {
		testCaseID = 218
		var err error
		cluster, err = helper.CreateAKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, k8sVersion, location, nil)
		Expect(err).To(BeNil())

		// Wait for the cluster to appear on cloud console before deleting it
		Eventually(func() bool {
			exists, err := helper.ClusterExistsOnAzure(clusterName, cluster.AKSConfig.ResourceGroup)
			// ignore the error that occurs when resource group or cluster could not be found
			if err != nil {
				if strings.Contains(err.Error(), fmt.Sprintf("Resource group '%s' could not be found", cluster.AKSConfig.ResourceGroup)) || strings.Contains(err.Error(), "not found") {
					err = nil
				}
			}
			Expect(err).To(BeNil())
			return exists
		}, "1m", "5s").Should(BeTrue())

		cluster, err = ctx.RancherAdminClient.Management.Cluster.ByID(cluster.ID)
		Expect(err).To(BeNil())
		err = helper.DeleteAKSHostCluster(cluster, ctx.RancherAdminClient)
		Expect(err).To(BeNil())

		// Wait until the cluster finishes provisioning and then begins deletion process
		Eventually(func() bool {
			exists, err := helper.ClusterExistsOnAzure(clusterName, cluster.AKSConfig.ResourceGroup)
			if err != nil {
				if strings.Contains(err.Error(), fmt.Sprintf("Resource group '%s' could not be found", cluster.AKSConfig.ResourceGroup)) || strings.Contains(err.Error(), "not found") {
					err = nil
				}
			}
			Expect(err).To(BeNil())
			return exists
		}, "10m", "10s").Should(BeFalse())

		// Keep the cluster variable as is so that there is no error in AfterEach; failed delete operation will return an empty cluster
		cluster, err = ctx.RancherAdminClient.Management.Cluster.ByID(cluster.ID)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not found"))
	})

	It("should not be able to select NP K8s version; CP K8s version should take precedence", func() {
		testCaseID = 182

		k8sVersions, err := helper.ListSingleVariantAKSAvailableVersions(ctx.RancherAdminClient, ctx.CloudCred.ID, location)
		Expect(err).To(BeNil())
		Expect(len(k8sVersions)).To(BeNumerically(">=", 2))
		cpK8sVersion := k8sVersions[0]
		npK8sVersion := k8sVersions[1]

		GinkgoLogr.Info(fmt.Sprintf("Using NP K8s version: %s and CP K8s version: %s", npK8sVersion, cpK8sVersion))

		updateFunc := func(clusterConfig *aks.ClusterConfig) {
			nodePools := *clusterConfig.NodePools
			for i := range nodePools {
				nodePools[i].OrchestratorVersion = &npK8sVersion
			}
			*clusterConfig.NodePools = nodePools
		}

		cluster, err = helper.CreateAKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, cpK8sVersion, location, updateFunc)
		Expect(err).To(BeNil())

		Expect(*cluster.AKSConfig.KubernetesVersion).To(Equal(cpK8sVersion))
		for _, np := range cluster.AKSConfig.NodePools {
			Expect(*np.OrchestratorVersion).To(Equal(cpK8sVersion))
		}

		cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
		Expect(err).To(BeNil())

		Eventually(func() bool {
			GinkgoLogr.Info("Waiting for the k8s upgrade to appear in AKSStatus.UpstreamSpec...")
			clusterState, err := ctx.RancherAdminClient.Management.Cluster.ByID(cluster.ID)
			Expect(err).To(BeNil())
			if *clusterState.AKSStatus.UpstreamSpec.KubernetesVersion != cpK8sVersion {
				return false
			}

			for _, np := range clusterState.AKSStatus.UpstreamSpec.NodePools {
				if *np.OrchestratorVersion != cpK8sVersion {
					return false
				}
			}
			return true
		}, "5m", "5s").Should(BeTrue(), "Failed while waiting for k8s upgrade.")
	})

	When("a cluster is created for with user and system mode nodepool", func() {
		BeforeEach(func() {
			updateFunc := func(clusterConfig *aks.ClusterConfig) {
				nodePools := *clusterConfig.NodePools
				npTemplate := nodePools[0]
				var updatedNodePools []aks.NodePool
				for _, mode := range []string{"User", "System"} {
					np := aks.NodePool{
						AvailabilityZones:   npTemplate.AvailabilityZones,
						EnableAutoScaling:   npTemplate.EnableAutoScaling,
						MaxPods:             npTemplate.MaxPods,
						MaxCount:            npTemplate.MaxCount,
						MinCount:            npTemplate.MinCount,
						Mode:                mode,
						Name:                pointer.String(fmt.Sprintf("%spool", strings.ToLower(mode))),
						NodeCount:           npTemplate.NodeCount,
						OrchestratorVersion: &k8sVersion,
						OsDiskSizeGB:        npTemplate.OsDiskSizeGB,
						OsDiskType:          npTemplate.OsDiskType,
						OsType:              npTemplate.OsType,
						VMSize:              npTemplate.VMSize,
					}
					updatedNodePools = append(updatedNodePools, np)

				}
				*clusterConfig.NodePools = updatedNodePools
			}
			var err error
			cluster, err = helper.CreateAKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, k8sVersion, location, updateFunc)
			Expect(err).To(BeNil())
			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())
		})

		It("should successfully create the cluster", func() {
			testCaseID = 189
			helpers.ClusterIsReadyChecks(cluster, ctx.RancherAdminClient, clusterName)

			Expect(len(cluster.AKSConfig.NodePools)).To(Equal(2))
			Expect(len(cluster.AKSStatus.UpstreamSpec.NodePools)).To(Equal(2))
		})

		It("should to able to delete a nodepool and add a new one", func() {
			testCaseID = 190
			deleteAndAddNpCheck(cluster, ctx.RancherAdminClient)
		})

		It("should not be able to remove system nodepool", func() {
			testCaseID = 191
			removeSystemNpCheck(cluster, ctx.RancherAdminClient)
		})
	})

})
