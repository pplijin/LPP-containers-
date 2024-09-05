package p1_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"

	"github.com/rancher/hosted-providers-e2e/hosted/eks/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

var _ = Describe("SyncProvisioning", func() {
	var cluster *management.Cluster

	When("a cluster is created for sync", func() {

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

		It("Upgrade k8s version of cluster from EKS and verify it is synced back to Rancher", func() {
			testCaseID = 159

			By("upgrading the ControlPlane & NodeGroup", func() {
				syncK8sVersionUpgradeCheck(cluster, ctx.RancherAdminClient, true)
			})
		})

		It("Sync from Rancher to AWS console after a sync from AWS console to Rancher", func() {
			testCaseID = 157
			syncRancherToAWSCheck(cluster, ctx.RancherAdminClient)
		})
	})

})
