package k8s_chart_support_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"

	"github.com/rancher/hosted-providers-e2e/hosted/eks/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

var _ = Describe("K8sChartSupportImport", func() {
	var cluster *management.Cluster
	BeforeEach(func() {
		err := helper.CreateEKSClusterOnAWS(region, clusterName, k8sVersion, "1", helpers.GetCommonMetadataLabels())
		Expect(err).To(BeNil())

		cluster, err = helper.ImportEKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCredID, region)
		Expect(err).To(BeNil())
		cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
		Expect(err).To(BeNil())

	})
	AfterEach(func() {
		if ctx.ClusterCleanup && cluster != nil {
			err := helper.DeleteEKSHostCluster(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())
			err = helper.DeleteEKSClusterOnAWS(region, clusterName)
			Expect(err).To(BeNil())
		} else {
			fmt.Println("Skipping downstream cluster deletion: ", clusterName)
		}
	})

	It("should successfully test k8s chart support import", func() {
		testCaseID = 65 // Report to Qase
		commonchecks(ctx.RancherAdminClient, cluster)
	})
})
