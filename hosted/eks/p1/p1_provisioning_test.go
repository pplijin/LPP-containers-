package p1_test

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters/eks"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"

	"github.com/rancher/hosted-providers-e2e/hosted/eks/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
	"k8s.io/utils/pointer"
)

var _ = Describe("P1Provisioning", func() {
	var cluster *management.Cluster

	var _ = BeforeEach(func() {
		var err error
		k8sVersion, err = helper.GetK8sVersion(ctx.RancherAdminClient, false)
		Expect(err).To(BeNil())
		GinkgoLogr.Info(fmt.Sprintf("While provisioning, using kubernetes version %s for cluster %s", k8sVersion, clusterName))
	})

	Context("Provisioning/Editing a cluster with invalid config", func() {

		AfterEach(func() {
			if ctx.ClusterCleanup && (cluster != nil && cluster.ID != "") {
				if cluster != nil {
					err := helper.DeleteEKSHostCluster(cluster, ctx.RancherAdminClient)
					Expect(err).To(BeNil())
				}
			}
		})

		It("should error out to provision a cluster with no nodegroups", func() {
			testCaseID = 141

			updateFunc := func(clusterConfig *eks.ClusterConfig) {
				*clusterConfig.NodeGroupsConfig = nil
			}

			var err error
			cluster, err = helper.CreateEKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCredID, k8sVersion, region, updateFunc)
			Expect(err).To(BeNil())

			Eventually(func() bool {
				cluster, err := ctx.RancherAdminClient.Management.Cluster.ByID(cluster.ID)
				Expect(err).To(BeNil())
				return cluster.Transitioning == "error" && strings.Contains(cluster.TransitioningMessage, "Cluster must have at least one managed nodegroup or one self-managed node")
			}, "10m", "30s").Should(BeTrue())

		})

		It("should fail to provision a cluster with duplicate nodegroup names", func() {
			testCaseID = 255

			var err error
			updateFunc := func(clusterConfig *eks.ClusterConfig) {
				var updatedNodeGroupsList []eks.NodeGroupConfig
				*clusterConfig, err = helper.AddNodeGroupToConfig(*clusterConfig, 2)
				Expect(err).To(BeNil())

				for _, ng := range *clusterConfig.NodeGroupsConfig {
					ng.NodegroupName = pointer.String("duplicate")
					updatedNodeGroupsList = append([]eks.NodeGroupConfig{ng}, updatedNodeGroupsList...)
				}
				*clusterConfig.NodeGroupsConfig = updatedNodeGroupsList
			}
			cluster, err = helper.CreateEKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCredID, k8sVersion, region, updateFunc)
			Expect(err).To(BeNil())

			Eventually(func() bool {
				cluster, err := ctx.RancherAdminClient.Management.Cluster.ByID(cluster.ID)
				Expect(err).To(BeNil())
				return cluster.Transitioning == "error" && strings.Contains(cluster.TransitioningMessage, "node group names must be unique")
			}, "1m", "3s").Should(BeTrue())
		})

		It("Fail to create cluster with only Security groups", func() {
			testCaseID = 120

			sg := []string{namegen.AppendRandomString("sg-"), namegen.AppendRandomString("sg-")}
			updateFunc := func(clusterConfig *eks.ClusterConfig) {
				clusterConfig.SecurityGroups = sg
			}
			var err error
			cluster, err = helper.CreateEKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCredID, k8sVersion, region, updateFunc)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("subnets must be provided if security groups are provided")))
		})

		It("Fail to update both Public/Private access as false and invalid values of the access", func() {
			testCaseID = 147 // also covers 146

			var err error
			cluster, err = helper.CreateEKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCredID, k8sVersion, region, nil)
			Expect(err).To(BeNil())
			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())
			invalidEndpointCheck(cluster, ctx.RancherAdminClient)
			invalidAccessValuesCheck(cluster, ctx.RancherAdminClient)
		})
	})

	When("a cluster is created for upgrade", func() {

		BeforeEach(func() {
			var err error
			k8sVersion, err = helper.GetK8sVersion(ctx.RancherAdminClient, true)
			Expect(err).To(BeNil())
			upgradeToVersion, err = helper.GetK8sVersion(ctx.RancherAdminClient, false)
			Expect(err).To(BeNil())
			GinkgoLogr.Info(fmt.Sprintf("While provisioning, using kubernetes version %s for cluster %s", k8sVersion, clusterName))

			cluster, err = helper.CreateEKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCredID, k8sVersion, region, nil)
			Expect(err).To(BeNil())
			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			if ctx.ClusterCleanup && (cluster != nil && cluster.ID != "") {
				err := helper.DeleteEKSHostCluster(cluster, ctx.RancherAdminClient)
				Expect(err).To(BeNil())
			} else {
				fmt.Println("Skipping downstream cluster deletion: ", clusterName)
			}
		})

		It("Upgrade version of node group only", func() {
			testCaseID = 126
			upgradeNodeKubernetesVersionGTCPCheck(cluster, ctx.RancherAdminClient)
		})

		It("Update k8s version of cluster and add node groups", func() {
			testCaseID = 125
			upgradeCPAndAddNgCheck(cluster, ctx.RancherAdminClient)
		})

		// eks-operator/issues/752
		XIt("should successfully update a cluster while it is still in updating state", func() {
			testCaseID = 148
			updateClusterInUpdatingState(cluster, ctx.RancherAdminClient)
		})
	})

	When("a cluster is created", func() {

		BeforeEach(func() {
			var err error
			cluster, err = helper.CreateEKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCredID, k8sVersion, region, nil)
			Expect(err).To(BeNil())
			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			if ctx.ClusterCleanup && (cluster != nil && cluster.ID != "") {
				err := helper.DeleteEKSHostCluster(cluster, ctx.RancherAdminClient)
				Expect(err).To(BeNil())
			} else {
				fmt.Println("Skipping downstream cluster deletion: ", clusterName)
			}
		})

		It("Update cluster logging types", func() {
			testCaseID = 128

			var err error
			loggingTypes := []string{"api", "audit", "authenticator", "controllerManager", "scheduler"}
			By("Adding the LoggingTypes", func() {
				cluster, err = helper.UpdateLogging(cluster, ctx.RancherAdminClient, loggingTypes, true)
				Expect(err).To(BeNil())
			})

			By("Removing the LoggingTypes", func() {
				cluster, err = helper.UpdateLogging(cluster, ctx.RancherAdminClient, []string{loggingTypes[0]}, true)
				Expect(err).To(BeNil())
			})
		})

		It("Update Tags and Labels", func() {
			testCaseID = 131

			var err error
			tags := map[string]string{
				"foo":        "bar",
				"testCaseID": "144",
			}

			labels := map[string]string{
				"testCaseID": "142",
			}

			By("Adding cluster tags", func() {
				cluster, err = helper.UpdateClusterTags(cluster, ctx.RancherAdminClient, tags, true)
				Expect(err).To(BeNil())
			})

			By("Adding Nodegroup tags & labels", func() {
				cluster, err = helper.UpdateNodegroupMetadata(cluster, ctx.RancherAdminClient, tags, labels, true)
				Expect(err).To(BeNil())
			})
		})
	})
})
