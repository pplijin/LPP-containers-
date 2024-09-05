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
			if ctx.ClusterCleanup && cluster != nil {
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
			cluster, err = helper.CreateEKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, k8sVersion, region, updateFunc)
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
			cluster, err = helper.CreateEKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, k8sVersion, region, updateFunc)
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
			cluster, err = helper.CreateEKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, k8sVersion, region, updateFunc)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("subnets must be provided if security groups are provided")))
		})

		It("Fail to update invalid values for Public Access Endpoints", func() {
			testCaseID = 146

			var err error
			cidr := []string{namegen.AppendRandomString("invalid")}
			cluster, err = helper.CreateEKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, k8sVersion, region, nil)
			Expect(err).To(BeNil())
			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())
			cluster, _ = helper.UpdatePublicAccessSources(cluster, ctx.RancherAdminClient, cidr, false)

			Eventually(func() bool {
				cluster, err := ctx.RancherAdminClient.Management.Cluster.ByID(cluster.ID)
				Expect(err).To(BeNil())
				return cluster.Transitioning == "error" && strings.Contains(cluster.TransitioningMessage, "InvalidParameterException: The following CIDRs are invalid in publicAccessCidrs")
			}, "1m", "3s").Should(BeTrue())
		})

		It("Fail to update both Public/Private access as false", func() {
			testCaseID = 147

			var err error
			cluster, err = helper.CreateEKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, k8sVersion, region, nil)
			Expect(err).To(BeNil())
			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())

			_, err = helper.UpdateAccess(cluster, ctx.RancherAdminClient, false, false, false)
			Expect(err).To(MatchError(ContainSubstring("public access, private access, or both must be enabled")))
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

			cluster, err = helper.CreateEKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, k8sVersion, region, nil)
			Expect(err).To(BeNil())
			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			if ctx.ClusterCleanup && cluster != nil {
				err := helper.DeleteEKSHostCluster(cluster, ctx.RancherAdminClient)
				Expect(err).To(BeNil())
			} else {
				fmt.Println("Skipping downstream cluster deletion: ", clusterName)
			}
		})

		It("Upgrade version of node group only", func() {
			testCaseID = 126

			By("upgrading only the NodeGroups", func() {
				var err error
				upgradeToVersion, err := helper.GetK8sVersion(ctx.RancherAdminClient, false)
				Expect(err).To(BeNil())
				GinkgoLogr.Info("Upgrading Nodegroup's EKS version")
				cluster, err = helper.UpgradeNodeKubernetesVersion(cluster, upgradeToVersion, ctx.RancherAdminClient, false, false)
				Expect(err).To(BeNil())
			})

			// wait until the error is visible on the cluster
			Eventually(func() bool {
				cluster, err := ctx.RancherAdminClient.Management.Cluster.ByID(cluster.ID)
				Expect(err).To(BeNil())
				return cluster.Transitioning == "error" && strings.Contains(cluster.TransitioningMessage, "all nodegroup kubernetes versionsmust be equal to or one minor version lower than the cluster kubernetes version")
			}, "1m", "3s").Should(BeTrue())
		})

		It("Update k8s version of cluster and add node groups", func() {
			testCaseID = 125

			var err error
			By("upgrading the ControlPlane", func() {
				cluster, err = helper.UpgradeClusterKubernetesVersion(cluster, upgradeToVersion, ctx.RancherAdminClient, true)
				Expect(err).To(BeNil())
			})

			By("adding a NodeGroup", func() {
				cluster, err = helper.AddNodeGroup(cluster, 1, ctx.RancherAdminClient, true, false)
				Expect(err).To(BeNil())
			})

			By("deleting the NodeGroup", func() {
				cluster, err = helper.DeleteNodeGroup(cluster, ctx.RancherAdminClient, true, false)
				Expect(err).To(BeNil())
			})

			// wait until the update is visible on the cluster
			Eventually(func() bool {
				GinkgoLogr.Info("Waiting for the versin of new nodegroup to appear in EKSStatus.UpstreamSpec ...")
				cluster, err = ctx.RancherAdminClient.Management.Cluster.ByID(cluster.ID)
				Expect(err).To(BeNil())
				for _, ng := range cluster.EKSStatus.UpstreamSpec.NodeGroups {
					if *ng.Version != upgradeToVersion {
						return false
					}
				}
				return true
			}, "5m", "15s").Should(BeTrue())
		})

		It("should successfully update a cluster while it is still in updating state", func() {
			testCaseID = 148
			updateClusterInUpdatingState(cluster, ctx.RancherAdminClient)
		})
	})

	When("a cluster is created", func() {

		BeforeEach(func() {
			var err error
			cluster, err = helper.CreateEKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, k8sVersion, region, nil)
			Expect(err).To(BeNil())
			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())
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
