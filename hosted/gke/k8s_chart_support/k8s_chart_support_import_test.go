package k8s_chart_support_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"

	"github.com/rancher/hosted-providers-e2e/hosted/gke/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

var _ = Describe("K8sChartSupportImport", func() {
	var (
		cluster *management.Cluster
	)

	BeforeEach(func() {
		err := helper.CreateGKEClusterOnGCloud(zone, clusterName, project, k8sVersion)
		Expect(err).To(BeNil())

		cluster, err = helper.ImportGKEHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCredID, zone, project)
		Expect(err).To(BeNil())
		cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		if ctx.ClusterCleanup && cluster != nil {
			err := helper.DeleteGKEHostCluster(cluster, ctx.RancherAdminClient)
			Expect(err).To(BeNil())
			err = helper.DeleteGKEClusterOnGCloud(zone, project, clusterName)
			Expect(err).To(BeNil())

		} else {
			fmt.Println("Skipping downstream cluster deletion: ", clusterName)
		}
	})

	It("should successfully test k8s chart support import", func() {
		testCaseID = 65 // Report to Qase
		commonChartSupport(ctx.RancherAdminClient, cluster)
	})
})
