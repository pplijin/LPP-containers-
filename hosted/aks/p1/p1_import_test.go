package p1_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"

	"github.com/rancher/hosted-providers-e2e/hosted/aks/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

var _ = Describe("P1Import", func() {
	var k8sVersion string
	var cluster *management.Cluster
	BeforeEach(func() {
		var err error
		k8sVersion, err = helper.GetK8sVersion(ctx.RancherAdminClient, ctx.CloudCred.ID, location, false)
		Expect(err).NotTo(HaveOccurred())
		GinkgoLogr.Info(fmt.Sprintf("Using K8s version %s for cluster %s", k8sVersion, clusterName))
	})

	AfterEach(func() {
		if ctx.ClusterCleanup {
			if cluster != nil && cluster.ID != "" {
				err := helper.DeleteAKSHostCluster(cluster, ctx.RancherAdminClient)
				Expect(err).To(BeNil())
			}
			err := helper.DeleteAKSClusteronAzure(clusterName)
			Expect(err).To(BeNil())

		} else {
			fmt.Println("Skipping downstream cluster deletion: ", clusterName)
		}
	})

	When("a cluster is created", func() {
		BeforeEach(func() {
			var err error
			err = helper.CreateAKSClusterOnAzure(location, clusterName, k8sVersion, "1", helpers.GetCommonMetadataLabels())
			Expect(err).To(BeNil())

			cluster, err = helper.ImportAKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, location, helpers.GetCommonMetadataLabels())
			Expect(err).To(BeNil())
			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())
		})

		It("should be able to update autoscaling", func() {
			testCaseID = 266
			updateAutoScaling(cluster, ctx.RancherAdminClient)
		})

		It("should be able to update tags", func() {
			testCaseID = 270
			updateTagsCheck(cluster, ctx.RancherAdminClient)
		})

		It("should be able to update cluster monitoring", func() {
			testCaseID = 271
			updateMonitoringCheck(cluster, ctx.RancherAdminClient)
		})

	})

	It("should be able to register a cluster with no rbac", func() {
		testCaseID = 237
		err := helper.CreateAKSClusterOnAzure(location, clusterName, k8sVersion, "1", helpers.GetCommonMetadataLabels(), "--disable-rbac")
		Expect(err).To(BeNil())

		cluster, err = helper.ImportAKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, location, helpers.GetCommonMetadataLabels())
		Expect(err).To(BeNil())
		cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
		Expect(err).To(BeNil())
		helpers.ClusterIsReadyChecks(cluster, ctx.RancherAdminClient, clusterName)
	})

	When("a cluster is created with multiple nodepools", func() {
		BeforeEach(func() {
			var err error
			err = helper.CreateAKSClusterOnAzure(location, clusterName, k8sVersion, "1", helpers.GetCommonMetadataLabels())
			Expect(err).To(BeNil())

			// AKS already consists on one system pool, so we add a user pool
			err = helper.AddNodePoolOnAzure("userpool", clusterName, clusterName, "2", "--mode", "User")
			Expect(err).To(BeNil())

			cluster, err = helper.ImportAKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, location, helpers.GetCommonMetadataLabels())
			Expect(err).To(BeNil())
			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())
		})

		XIt("should not be able to remove system nodepool", func() {
			// Blocked on https://github.com/rancher/aks-operator/issues/619
			testCaseID = 267
			removeSystemNpCheck(cluster, ctx.RancherAdminClient)
		})

		It("should to able to delete a nodepool and add a new one", func() {
			testCaseID = 268
			deleteAndAddNpCheck(cluster, ctx.RancherAdminClient)
		})

	})

	When("a cluster is created and imported for upgrade", func() {
		BeforeEach(func() {
			var err error
			k8sVersion, err = helper.GetK8sVersion(ctx.RancherAdminClient, ctx.CloudCred.ID, location, true)
			Expect(err).NotTo(HaveOccurred())
			GinkgoLogr.Info(fmt.Sprintf("Using K8s version %s for cluster %s", k8sVersion, clusterName))

			err = helper.CreateAKSClusterOnAzure(location, clusterName, k8sVersion, "1", helpers.GetCommonMetadataLabels())
			Expect(err).To(BeNil())

			cluster, err = helper.ImportAKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, location, helpers.GetCommonMetadataLabels())
			Expect(err).To(BeNil())
			cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())
		})

		It("NP cannot be upgraded to k8s version greater than CP k8s version", func() {
			testCaseID = 269
			npUpgradeToVersionGTCPCheck(cluster, ctx.RancherAdminClient)
		})
	})

})
