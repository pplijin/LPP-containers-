package p1_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"

	"github.com/rancher/hosted-providers-e2e/hosted/gke/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

var _ = Describe("SyncProvisioning", func() {
	for _, testData := range []struct {
		qaseID    int64
		isUpgrade bool
		testBody  func(cluster *management.Cluster, client *rancher.Client)
		testTitle string
	}{
		{
			qaseID:    39,
			isUpgrade: true,
			testBody:  syncK8sVersionUpgradeCheck,
			testTitle: "should Sync from GCE to Rancher - changed k8s version",
		},
		{
			qaseID:    40,
			isUpgrade: false,
			testBody:  syncNodepoolsCheck,
			testTitle: "should Sync from GCE to Rancher - add/delete nodepool",
		},
		{
			qaseID:    46,
			isUpgrade: true,
			testBody:  syncSameK8sVersionUpgradeCheck,
			testTitle: "Updating to the same k8s version from GCP and Rancher",
		},
	} {
		testData := testData
		When("a cluster is created", func() {
			var cluster *management.Cluster

			BeforeEach(func() {
				k8sVersion, err := helper.GetK8sVersion(ctx.RancherAdminClient, project, ctx.CloudCred.ID, zone, "", testData.isUpgrade)
				Expect(err).NotTo(HaveOccurred())
				GinkgoLogr.Info(fmt.Sprintf("Using K8s version %s for cluster %s", k8sVersion, clusterName))

				cluster, err = helper.CreateGKEHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, k8sVersion, zone, project, 1)
				Expect(err).To(BeNil())
				cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
				Expect(err).To(BeNil())
			})

			AfterEach(func() {
				if ctx.ClusterCleanup && cluster != nil {
					err := helper.DeleteGKEHostCluster(cluster, ctx.RancherAdminClient)
					Expect(err).To(BeNil())
				} else {
					fmt.Println("Skipping downstream cluster deletion: ", clusterName)
				}
			})

			It(testData.testTitle, func() {
				testCaseID = testData.qaseID
				testData.testBody(cluster, ctx.RancherAdminClient)
			})
		})
	}
})
